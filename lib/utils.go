package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

func GenerateID(length int) string {
	// 获取当前时间戳（以秒为单位）
	timestamp := time.Now().Unix()

	// 将时间戳转换为字符串
	timestampStr := fmt.Sprintf("%d", timestamp)

	// 使用 SHA-256 哈希算法
	hash := sha256.New()
	hash.Write([]byte(timestampStr))
	hashBytes := hash.Sum(nil)

	// 将哈希值转成十六进制字符串
	hashStr := hex.EncodeToString(hashBytes)

	// 截取前 'length' 个字符
	if length > len(hashStr) {
		length = len(hashStr)
	}
	return hashStr[:length]
}
