package lib

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/tags"
)

const bucketName = "tempfile"
const endpoint = "172.17.0.1:9000"
const accessKeyID = "fO1LiJbIykmUvSLa5ttV"
const secretAccessKey = "PEg1ELGiQBzSxyXehH2ePvvp3UhlKFQwv5u8Y7yI"
const useSSL = false

var minioClient *minio.Client
var err error

func init() {
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		log.Fatalln(err)
	}
}

func UploadFileToMinio(id string, fileInfo FileModel, fileStream io.Reader) error {
	ctx := context.Background()

	// 创建标签
	fileTags, err := tags.NewTags(map[string]string{
		"tag": fileInfo.Tag,
	}, true) // true indicates object tags

	if err != nil {
		return err
	}

	// 将 FileModel 序列化为 JSON
	infoData, err := json.Marshal(fileInfo)
	if err != nil {
		return err
	}

	// 保存文件名（.info 文件）
	reader := strings.NewReader(string(infoData))
	_, err = minioClient.PutObject(ctx, bucketName, id+".info", reader, int64(len(infoData)), minio.PutObjectOptions{
		ContentType: "application/json",
		UserTags:    fileTags.ToMap(), // 设置标签
	})
	if err != nil {
		return err
	}

	// 保存文件内容（.content 文件）
	_, err = minioClient.PutObject(ctx, bucketName, id+".content", fileStream, fileInfo.Size, minio.PutObjectOptions{
		UserTags: fileTags.ToMap(), // 设置标签
	})
	if err != nil {
		return err
	}

	return nil
}

type MinioFileResponse struct {
	Info    FileModel
	Content io.ReadCloser
	Object  *minio.Object
}

func GetFileObjectFromMinio(id string) (*minio.Object, error) {
	ctx := context.Background()
	content, err := minioClient.GetObject(ctx, bucketName, id+".content", minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	return content, nil
}

func GetFileFromMinio(id string) (*MinioFileResponse, error) {
	ctx := context.Background()
	content, err := minioClient.GetObject(ctx, bucketName, id+".content", minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	infoObject, err := minioClient.GetObject(ctx, bucketName, id+".info", minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	infoData, err := io.ReadAll(infoObject)
	if err != nil {
		return nil, err
	}

	var newFileInfo FileModel
	err = json.Unmarshal(infoData, &newFileInfo)
	if err != nil {
		return nil, err
	}

	return &MinioFileResponse{
		Info:    newFileInfo,
		Content: content,
	}, nil

}

func GetFileInfoFromMinio(id string) (*MinioFileResponse, error) {
	ctx := context.Background()
	fileInfoObject, err := minioClient.GetObject(ctx, bucketName, id+".info", minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	fileInfoData, err := io.ReadAll(fileInfoObject)
	if err != nil {
		return nil, err
	}

	var fileInfo FileModel
	err = json.Unmarshal(fileInfoData, &fileInfo)
	if err != nil {
		return nil, err
	}

	return &MinioFileResponse{
		Info:    fileInfo,
		Content: nil,
	}, nil
}
