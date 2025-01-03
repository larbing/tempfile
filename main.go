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

	id := lib.GenerateID(8)

	go func() {
		lib.UploadFileToMinio(id, fileModel, fileStream)
	}()

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func download(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	resp, err := lib.GetFileFromMinio(id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fileName := resp.Info.Name

	body, err := io.ReadAll(resp.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read response body: %v", err)})
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))

	// 返回响应内容
	c.Data(http.StatusOK, "application/octet-stream", body)
}

func info(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	resp, err := lib.GetFileInfoFromMinio(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fileInfo := resp.Info

	c.JSON(http.StatusOK, gin.H{"error": nil, "data": fileInfo})
}

func options(c *gin.Context) {
	c.String(http.StatusOK, "")
}

func notFound(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "https://tempfile.itoolkit.top")
}

func main() {

	r := gin.Default()

	// Enable CORS
	r.Use(CORS())

	// Set the maximum upload size
	r.MaxMultipartMemory = (8 << 20) * 6

	r.POST("/api/files", upload)
	r.GET("/api/files/:id/download", download)
	r.GET("/api/files/:id", info)
	r.OPTIONS("/api/files", options)
	r.NoRoute(notFound)

	r.Run(":8080")
}
