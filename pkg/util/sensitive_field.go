package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

// SensitiveField 敏感字段加密/解密/哈希工具
// 适用场景：需要加密存储且支持重复性检查的敏感数据（如手机号、身份证号等）
//
// 设计原则：
// 1. 使用 HashForUnique 生成确定性哈希，用于唯一性检查和查询
// 2. 使用 Encrypt 生成随机密文，用于存储和解密
// 3. 使用 Decrypt 解密获取原始值
//
// 使用示例：
//   hash, err := util.HashSensitiveField("手机号")
//   encrypted, err := util.EncryptSensitiveField("手机号")
//   decrypted, err := util.DecryptSensitiveField(encrypted)

var (
	// encryptKey AES加密密钥（32字节）
	encryptKey = []byte("sensitive-encrypt-key-32bytes!!!")

	// hashSalt 哈希盐值（32字节）
	hashSalt = []byte("sensitive-hash-salt-must-be-32")
)

// SensitiveFieldPair 敏感字段处理结果对
type SensitiveFieldPair struct {
	Hash      string // 用于唯一性检查
	Encrypted string // 用于存储和解密
}

// HashSensitiveField 生成敏感字段的哈希值
// 用于唯一性检查、去重、快速查询
//
// 特性：
// - 确定性：相同输入总是产生相同哈希值
// - 单向：无法从哈希值推导出原始值
// - 固定长度：SHA-256 输出固定64字符的十六进制字符串
func HashSensitiveField(value string) (string, error) {
	if value == "" {
		return "", errors.New("值不能为空")
	}

	h := sha256.New()
	h.Write(hashSalt)
	h.Write([]byte(value))

	return hex.EncodeToString(h.Sum(nil)), nil
}

// EncryptSensitiveField 加密敏感字段
// 用于存储需要解密的原始数据
//
// 特性：
// - 随机性：每次加密相同输入产生不同密文（更强的安全性）
// - 可逆：可以解密获取原始值
// - 完整性：使用 AES-GCM 模式，提供数据完整性保护
func EncryptSensitiveField(value string) (string, error) {
	if value == "" {
		return "", errors.New("值不能为空")
	}

	// 创建AES加密块
	block, err := aes.NewCipher(encryptKey)
	if err != nil {
		return "", err
	}

	// 创建GCM模式
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 创建随机nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(value), nil)

	// 返回Base64编码的加密结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSensitiveField 解密敏感字段
func DecryptSensitiveField(encryptedValue string) (string, error) {
	if encryptedValue == "" {
		return "", errors.New("加密值不能为空")
	}

	// 解码Base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedValue)
	if err != nil {
		return "", err
	}

	// 创建AES加密块
	block, err := aes.NewCipher(encryptKey)
	if err != nil {
		return "", err
	}

	// 创建GCM模式
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 获取nonce大小
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("加密数据格式错误")
	}

	// 分离nonce和加密数据
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// 解密
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// BatchHashSensitiveField 批量生成敏感字段的哈希值
func BatchHashSensitiveField(values []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, v := range values {
		hash, err := HashSensitiveField(v)
		if err != nil {
			return nil, err
		}
		result[v] = hash
	}
	return result, nil
}

// BatchEncryptSensitiveField 批量加密敏感字段
func BatchEncryptSensitiveField(values []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, v := range values {
		encrypted, err := EncryptSensitiveField(v)
		if err != nil {
			return nil, err
		}
		result[v] = encrypted
	}
	return result, nil
}

// BatchDecryptSensitiveField 批量解密敏感字段
func BatchDecryptSensitiveField(encryptedValues []string) ([]string, error) {
	result := make([]string, len(encryptedValues))
	for i, v := range encryptedValues {
		decrypted, err := DecryptSensitiveField(v)
		if err != nil {
			return nil, err
		}
		result[i] = decrypted
	}
	return result, nil
}

// ProcessSensitiveField 处理敏感字段，同时生成哈希和加密值
// 返回一个包含两种处理结果的结构体
func ProcessSensitiveField(value string) (*SensitiveFieldPair, error) {
	if value == "" {
		return nil, errors.New("值不能为空")
	}

	hash, err := HashSensitiveField(value)
	if err != nil {
		return nil, err
	}

	encrypted, err := EncryptSensitiveField(value)
	if err != nil {
		return nil, err
	}

	return &SensitiveFieldPair{
		Hash:      hash,
		Encrypted: encrypted,
	}, nil
}

// BatchProcessSensitiveField 批量处理敏感字段
func BatchProcessSensitiveField(values []string) (map[string]*SensitiveFieldPair, error) {
	result := make(map[string]*SensitiveFieldPair)
	for _, v := range values {
		pair, err := ProcessSensitiveField(v)
		if err != nil {
			return nil, err
		}
		result[v] = pair
	}
	return result, nil
}

// GetMobileMask 手机号脱敏，保留前3位和后4位
// 例如: 13812345678 -> 138****5678
func GetMobileMask(mobile string) string {
	if len(mobile) != 11 {
		return mobile
	}
	return mobile[:3] + "****" + mobile[7:]
}
