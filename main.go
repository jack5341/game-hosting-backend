package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
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

	cmd := exec.Command("kubectl", "apply", "-f", outputFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to apply manifest: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Server created successfully"})
}
