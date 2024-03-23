package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"text/template"

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
	PROJECT_ZOMBOID GAMES = "PROJECT_ZOMBOID"
)

type CreateServerAttributes struct {
	Name        string       `form:"name" binding:"required"`
	ServerType  SERVER_TYPES `form:"serverTypes" binding:"required"`
	Game        SERVER_TYPES `form:"serverTypes" binding:"required"`
	Description string       `form:"description"`
}

func createServer(c *gin.Context) {
	var attributes CreateServerAttributes
	err := c.Bind(&attributes)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missed attribute %s", err.Error())})
		return
	}

	tmp, err := template.New("./templates/pz.yaml").Parse(string(""))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missed attribute %s", err.Error())})
		return
	}

	id, err := uuid.NewUUID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ID could not be created %s", err.Error())})
		return
	}

	manifestName := fmt.Sprintf("%s-%s", PROJECT_ZOMBOID, id.String())
	dir := fmt.Sprintf("./manifests/%s.yaml", manifestName)
	outputFile, err := os.Create(dir)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missed attribute %s", err.Error())})
		return
	}

	// Close the file when done
	defer outputFile.Close()

	err = tmp.Execute(outputFile, map[string]string{
		"ID":   id.String(),
		"Size": string(PZ_MD),
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Template could not be created %s", err.Error())})
		return
	}

	cmd := exec.Command("kubectl", "-f", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Something went wrong %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Server created successfully"})
}
