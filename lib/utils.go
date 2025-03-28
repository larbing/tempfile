package lib

import (
	"crypto/rand" // 用于生成安全的随机数
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

func GenerateID(length int) string {
	// 获取当前时间戳
	timestamp := time.Now().UnixNano()

	// 生成16字节的随机数据
	randBytes := make([]byte, 16)
	_, err := rand.Read(randBytes)
	if err != nil {
		// 在生产环境中需要更好的错误处理
		return ""
	}

	// 组合时间戳和随机字节
	uniqueStr := string(randBytes) + strconv.FormatInt(timestamp, 10)

	// 使用 SHA-256 哈希
	hash := sha256.New()
	hash.Write([]byte(uniqueStr))
	hashBytes := hash.Sum(nil)

	// 转为十六进制字符串
	hashStr := hex.EncodeToString(hashBytes) // 长度为64个字符

	// 截取指定长度
	if length > len(hashStr) {
		length = len(hashStr)
	}
	return hashStr[:length]
}
