package lib

import (
	"context"
	"encoding/json"
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

func UploadFileToMinio(id string, fileInfo FileModel, fileStream io.Reader) error {

	ctx := context.Background()

	infoData, err := json.Marshal(fileInfo)
	if err != nil {
		return err
	}

	//保存文件名
	reader := strings.NewReader(string(infoData))
	_, err = minioClient.PutObject(ctx, bucketName, id+".info", reader, int64(len(infoData)), minio.PutObjectOptions{ContentType: "application/json"})
	if err != nil {
		return err
	}

	//保存文件内容
	_, err = minioClient.PutObject(ctx, bucketName, id+".content", fileStream, fileInfo.Size, minio.PutObjectOptions{})
	if err != nil {
		return err
	}

	return nil
}

type MinioFileResponse struct {
	Info    FileModel
	Content io.Reader
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
