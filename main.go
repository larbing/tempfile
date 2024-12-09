package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"tempfile/lib"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	}
}

func upload(c *gin.Context) {

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fileModel := lib.FileModel{
		Name: file.Filename,
		Size: file.Size,
	}

	fileStream, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer fileStream.Close()

	response, err := lib.UploadFile(fileModel, fileStream)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": response["data"].(map[string]interface{})["id"].(string)})
}

func download(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	resp, err := lib.DownloadFile(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("received non-OK status code: %d", resp.StatusCode)})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read response body: %v", err)})
		return
	}

	// 设置响应头
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Header("Content-Disposition", resp.Header.Get("Content-Disposition"))

	// 返回响应内容
	c.Data(http.StatusOK, resp.Header.Get("Content-Type"), body)
}

func options(c *gin.Context) {
	c.String(http.StatusOK, "")
}

func main() {

	r := gin.Default()

	// Enable CORS
	r.Use(CORS())

	// Set the maximum upload size
	r.MaxMultipartMemory = (8 << 20) * 6

	r.POST("/upload", upload)
	r.GET("/download/:id", download)
	r.OPTIONS("/upload", options)

	r.Run(":8080")
}
