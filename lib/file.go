package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

type FileModel struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type Response map[string]interface{}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func DownloadFile(id string) (*http.Response, error) {
	baseURL, err := url.Parse("https://tmpsend.com/download")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}
	queryParams := url.Values{}
	queryParams.Add("d", id)
	baseURL.RawQuery = queryParams.Encode()

	refererURL, err := url.Parse("https://tmpsend.com/thank-you")
	if err != nil {
		return nil, fmt.Errorf("failed to parse referer URL: %v", err)
	}
	refererQueryParams := url.Values{}
	refererQueryParams.Add("d", id)
	refererURL.RawQuery = refererQueryParams.Encode()

	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Referer", refererURL.String())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: 10 * time.Second, // 10秒超时
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	// defer resp.Body.Close()

	return resp, nil
}

func UploadFile(fileModel FileModel, fileStream multipart.File) (Response, error) {
	url := "https://tmpsend.com/upload"

	// First request to get the ID
	payload := map[string]interface{}{
		"action": "add",
		"name":   fileModel.Name,
		"size":   fileModel.Size,
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	err := writer.WriteField("action", payload["action"].(string))
	if err != nil {
		return nil, err
	}

	err = writer.WriteField("name", payload["name"].(string))
	if err != nil {
		return nil, err
	}

	err = writer.WriteField("size", fmt.Sprintf("%d", payload["size"].(int64)))
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, writer.FormDataContentType(), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get ID: %s", resp.Status)
	}

	var responseMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
		return nil, err
	}

	id, ok := responseMap["id"].(string)
	if !ok {
		return nil, fmt.Errorf("ID not found in response")
	}

	payload = map[string]interface{}{
		"action": "upload",
		"id":     id,
		"name":   fileModel.Name,
		"size":   fileModel.Size,
	}

	response, err := uploadInChunks(url, fileStream, 2*1024*1024, payload)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func uploadInChunks(url string, fileStream multipart.File, chunkSize int64, payload map[string]interface{}) (Response, error) {
	totalSize := payload["size"].(int64)
	start := int64(0)

	for start < totalSize {
		end := min(start+chunkSize, totalSize)
		payload["start"] = start
		payload["end"] = end

		chunk := make([]byte, end-start)
		_, err := fileStream.Read(chunk)
		if err != nil && err != io.EOF {
			return nil, err
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("data", payload["name"].(string))
		if err != nil {
			return nil, err
		}

		_, err = part.Write(chunk)
		if err != nil {
			return nil, err
		}

		for key, val := range payload {
			err = writer.WriteField(key, fmt.Sprintf("%v", val))
			if err != nil {
				return nil, err
			}
		}

		err = writer.Close()
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to upload chunk: %s", resp.Status)
		}

		var responseMap Response
		if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
			return nil, err
		}

		if hasError, ok := responseMap["hasError"].(bool); ok && hasError {
			return nil, fmt.Errorf("error uploading file")
		}

		start = end

		if end == totalSize {
			return responseMap, nil
		}
	}

	return nil, fmt.Errorf("start < totalSize")
}
