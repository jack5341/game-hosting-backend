package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var clientSet *kubernetes.Clientset

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.POST("/server", createServer)
	r.Run()
}

type SERVER_TYPES string

const (
	PZ_SM SERVER_TYPES = "PZ_SM"
	PZ_MD SERVER_TYPES = "PZ_MD"
	PZ_LG SERVER_TYPES = "PZ_LG"
)

type GAMES string

const (
	PZ GAMES = "PZ"
)

type CreateServerAttributes struct {
	Name        string       `form:"name" binding:"required"`
	ServerType  SERVER_TYPES `form:"serverTypes" binding:"required"`
	Game        GAMES        `form:"game" binding:"required"`
	Description string       `form:"description"`
}

const (
	templatesDir = "./templates"
	manifestsDir = "./manifests"
)

func createServer(c *gin.Context) {
	var attributes CreateServerAttributes
	if err := c.Bind(&attributes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to bind attributes: %s", err.Error())})
		return
	}

	id := uuid.New()
	templatePath := filepath.Join(templatesDir, fmt.Sprintf("%s.yml", strings.ToLower(string(attributes.Game))))
	templateContent, err := ioutil.ReadFile(templatePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read template file: %s", err.Error())})
		return
	}

	tmpl, err := template.New("serverTemplate").Parse(string(templateContent))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to parse template: %s", err.Error())})
		return
	}

	var outputBuffer bytes.Buffer
	if err := tmpl.Execute(&outputBuffer, map[string]string{
		"ID":   id.String(),
		"Size": string(attributes.ServerType),
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to execute template: %s", err.Error())})
		return
	}

	outputFilePath := filepath.Join(manifestsDir, fmt.Sprintf("%s-%s.yml", string(attributes.Game), id.String()))
	if err := ioutil.WriteFile(outputFilePath, outputBuffer.Bytes(), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to write to output file: %s", err.Error())})
		return
	}

	manifestBytes, err := ioutil.ReadFile(outputFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read manifest file: %s", err.Error())})
		return
	}

	_, err = clientSet.CoreV1().RESTClient().Patch(types.ApplyPatchType).
		SetHeader("Content-Type", "application/apply-patch+yaml").
		Body(manifestBytes).
		DoRaw(context.TODO())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to apply manifest: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Server created successfully"})
}
