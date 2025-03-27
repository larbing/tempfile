package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"tempfile/lib"
	"time"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	}
}

// 限制并发请求数的中间件
func MaxConcurrency(n int) gin.HandlerFunc {
	sem := make(chan struct{}, n)
	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		default:
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "服务器繁忙，请稍后重试",
			})
			c.Abort()
		}
	}
}

// 请求超时中间件
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 对于流媒体请求，不设置超时
		if c.Request.URL.Path == "/api/files/:id/stream" {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			done <- struct{}{}
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{
				"error": "请求超时",
			})
		}
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

	// go func() {
	lib.UploadFileToMinio(id, fileModel, fileStream)
	// }()

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

func streamDownload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 不能为空"})
		return
	}

	fileObject, err := lib.GetFileObjectFromMinio(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 假设 resp.Content 是 io.Reader，如果是 io.ReadCloser，可改为 defer resp.Content.Close()
	defer fileObject.Close()

	// fileName := fileObject.Name
	// fileSize := fileObject.Size
	stat, err := fileObject.Stat()
	if err != nil {
		c.JSON(500, gin.H{"error": "获取文件信息失败"})
		return
	}

	fileSize := stat.Size
	contentType := stat.ContentType

	// 设置通用响应头
	// c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", fileName))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", contentType)
	c.Header("Connection", "keep-alive")

	// 处理 Range 请求
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		var start, end int64
		_, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
		if err != nil {
			_, err = fmt.Sscanf(rangeHeader, "bytes=%d-", &start)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Range 格式无效"})
				return
			}
			end = fileSize - 1
		}

		if start >= fileSize {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		if end >= fileSize {
			end = fileSize - 1
		}

		contentLength := end - start + 1
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		c.Header("Content-Length", fmt.Sprintf("%d", contentLength))

		// 在写入响应体前检查数据可用性
		bufReader := bufio.NewReader(fileObject)
		if start > 0 {
			_, err = io.CopyN(io.Discard, bufReader, start)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "无法读取文件起始部分"})
				return
			}
		}

		// 设置状态码并开始传输
		c.Status(http.StatusPartialContent)
		limitReader := io.LimitReader(bufReader, contentLength)
		if _, err = io.Copy(c.Writer, limitReader); err != nil {
			// 如果这里出错，已开始写入，无法更改状态码，只能记录日志
			fmt.Printf("Error during streaming: %v\n", err)
			return
		}
	} else {
		// 完整文件下载
		c.Header("Content-Length", fmt.Sprintf("%d", fileSize))

		bufReader := bufio.NewReader(fileObject)
		c.Status(http.StatusOK) // 设置 200
		limitReader := io.LimitReader(bufReader, fileSize)
		if _, err = io.Copy(c.Writer, limitReader); err != nil {
			fmt.Printf("Error during streaming: %v\n", err)
			return
		}
	}
}

func streamVideo(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 不能为空"})
		return
	}

	fileObject, err := lib.GetFileObjectFromMinio(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer fileObject.Close()

	stat, err := fileObject.Stat()
	if err != nil {
		c.JSON(500, gin.H{"error": "获取文件信息失败"})
		return
	}

	fileSize := stat.Size
	fileName := stat.Key
	contentType := "video/mp4"
	if stat.ContentType != "" {
		contentType = stat.ContentType
	}

	// 设置通用响应头
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", fileName))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", contentType)
	c.Header("Connection", "keep-alive")
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Header("X-Content-Type-Options", "nosniff")

	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		var start, end int64

		// 尝试解析范围格式 "bytes=start-end"
		_, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
		if err != nil {
			// 尝试解析格式 "bytes=start-"
			_, err = fmt.Sscanf(rangeHeader, "bytes=%d-", &start)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Range 格式无效"})
				return
			}
			// 如果没有指定结束位置，设置为文件末尾
			end = fileSize - 1
		}

		// 验证范围合法性
		if start < 0 {
			start = 0
		}

		if start >= fileSize {
			// 范围超出文件大小
			c.Header("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
			c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		if end >= fileSize {
			end = fileSize - 1
		}

		// 计算要发送的内容长度
		contentLength := end - start + 1

		// 设置部分内容响应头
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		c.Header("Content-Length", fmt.Sprintf("%d", contentLength))
		c.Status(http.StatusPartialContent)

		// 跳过不需要的字节
		if start > 0 {
			_, err = io.CopyN(io.Discard, fileObject, start)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "无法读取文件起始部分"})
				return
			}
		}

		// 使用带缓冲的读取器和有限读取器来流式传输
		bufReader := bufio.NewReaderSize(fileObject, 32*1024) // 32KB 缓冲区
		limitReader := io.LimitReader(bufReader, contentLength)

		// 流式传输到客户端
		_, err = io.Copy(c.Writer, limitReader)
		if err != nil {
			// 错误已经发生，只能记录，无法修改响应
			log.Printf("视频流传输错误: %v", err)
		}
	} else {

		c.Header("Content-Length", fmt.Sprintf("%d", fileSize))
		c.Status(http.StatusOK)

		// 使用带缓冲的读取器来提高性能
		bufReader := bufio.NewReaderSize(fileObject, 32*1024) // 32KB 缓冲区

		// 流式传输到客户端
		_, err = io.Copy(c.Writer, bufReader)
		if err != nil {
			log.Printf("视频流传输错误: %v", err)
		}
	}
}

// 根据文件扩展名推断 Content-Type
func getContentType(fileName string) string {
	ext := filepath.Ext(fileName)
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

func main() {
	// 设置为发布模式
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// Enable CORS
	r.Use(CORS())

	// // 限制最大并发请求为 200
	r.Use(MaxConcurrency(200))

	// // 添加 30 秒超时限制（流媒体请求除外）
	// r.Use(TimeoutMiddleware(30 * time.Second))

	// Set the maximum upload size
	r.MaxMultipartMemory = (8 << 20) * 6

	// // 添加请求时间中间件
	// r.Use(func(c *gin.Context) {
	// 	start := time.Now()
	// 	c.Next()
	// 	if c.Writer.Status() == http.StatusOK {
	// 		c.Header("X-Response-Time", fmt.Sprintf("%dms", time.Since(start).Milliseconds()))
	// 	}
	// })

	// 路由配置
	r.POST("/api/files", upload)
	r.GET("/api/files/:id/download", download)
	r.GET("/api/files/:id", info)
	r.OPTIONS("/api/files", options)
	r.GET("/api/files/:id/stream", streamVideo)
	r.GET("/api/files/:id/video", streamVideo)
	r.NoRoute(notFound)

	// 创建自定义 http.Server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
		// 读取请求的最大时间
		ReadTimeout: 60 * time.Second, // 增加到 60 秒
		// 写入响应的最大时间
		WriteTimeout: 60 * time.Second, // 增加到 60 秒
		// 请求头的最大时间
		IdleTimeout: 120 * time.Second,
		// 最大请求头大小
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// 优雅关闭配置
	go func() {
		// 等待中断信号
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server forced to shutdown: %v\n", err)
		}
	}()

	// 启动服务器
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server start failed: %v\n", err)
	}
}
