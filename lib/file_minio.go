package lib

import (
	"context"
	"io"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const bucketName = "tempfile"
const endpoint = "43.153.112.166:9000"
const accessKeyID = "qPjlzaM6Mxk8XiluX88z"
const secretAccessKey = "ZSVaI5wOvJvZ6N3p2phRuhnM5EPVVwtuMwHeuvvI"
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

func UploadFileToMinio(id string, fileModel FileModel, fileStream io.Reader) error {

	ctx := context.Background()

	//保存文件名
	reader := strings.NewReader(fileModel.Name)
	_, err = minioClient.PutObject(ctx, bucketName, id+".name", reader, int64(len(fileModel.Name)), minio.PutObjectOptions{ContentType: "text/plain"})
	if err != nil {
		return err
	}

	//保存文件内容
	_, err = minioClient.PutObject(ctx, bucketName, id+".content", fileStream, fileModel.Size, minio.PutObjectOptions{})
	if err != nil {
		return err
	}

	return nil
}

type MinioFileResponse struct {
	Name    io.Reader
	Content io.Reader
}

func DownloadFileFromMinio(id string) (*MinioFileResponse, error) {
	ctx := context.Background()
	content, err := minioClient.GetObject(ctx, bucketName, id+".content", minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	name, err := minioClient.GetObject(ctx, bucketName, id+".name", minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	return &MinioFileResponse{
		Name:    name,
		Content: content,
	}, nil

}
