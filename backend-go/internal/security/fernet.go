package security

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	fernetVersion  = byte(0x80)
	fernetKeySize  = 32
	fernetHMACSize = sha256.Size
	fernetIVSize   = aes.BlockSize
)

func EncryptString(secretKey string, plainValue string) (string, error) {
	key, err := parseFernetKey(secretKey)
	if err != nil {
		return "", err
	}
	iv := make([]byte, fernetIVSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("generate fernet iv: %w", err)
	}
	return encryptFernet(key, []byte(plainValue), iv, time.Now().Unix())
}

func DecryptString(secretKey string, encryptedValue string) (string, error) {
	key, err := parseFernetKey(secretKey)
	if err != nil {
		return "", err
	}
	plaintext, err := decryptFernet(key, encryptedValue)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func parseFernetKey(secretKey string) ([]byte, error) {
	key, err := base64.URLEncoding.DecodeString(secretKey)
	if err != nil {
		key, err = base64.RawURLEncoding.DecodeString(secretKey)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid fernet key: %w", err)
	}
	if len(key) != fernetKeySize {
		return nil, fmt.Errorf("invalid fernet key length: %d", len(key))
	}
	return key, nil
}

func encryptFernet(key []byte, plaintext []byte, iv []byte, timestamp int64) (string, error) {
	if len(iv) != fernetIVSize {
		return "", fmt.Errorf("invalid fernet iv length: %d", len(iv))
	}
	signingKey := key[:16]
	encryptionKey := key[16:]
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("create fernet cipher: %w", err)
	}
	padded := pkcs7Pad(plaintext, aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	message := make([]byte, 0, 1+8+len(iv)+len(ciphertext)+fernetHMACSize)
	message = append(message, fernetVersion)
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(timestamp))
	message = append(message, timestampBytes...)
	message = append(message, iv...)
	message = append(message, ciphertext...)
	mac := hmac.New(sha256.New, signingKey)
	_, _ = mac.Write(message)
	message = append(message, mac.Sum(nil)...)
	return base64.URLEncoding.EncodeToString(message), nil
}

func decryptFernet(key []byte, encryptedValue string) ([]byte, error) {
	token, err := base64.URLEncoding.DecodeString(encryptedValue)
	if err != nil {
		token, err = base64.RawURLEncoding.DecodeString(encryptedValue)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid fernet token: %w", err)
	}
	if len(token) < 1+8+fernetIVSize+aes.BlockSize+fernetHMACSize {
		return nil, errors.New("invalid fernet token length")
	}
	if token[0] != fernetVersion {
		return nil, errors.New("invalid fernet token version")
	}
	signingKey := key[:16]
	encryptionKey := key[16:]
	message := token[:len(token)-fernetHMACSize]
	tag := token[len(token)-fernetHMACSize:]
	mac := hmac.New(sha256.New, signingKey)
	_, _ = mac.Write(message)
	if !hmac.Equal(tag, mac.Sum(nil)) {
		return nil, errors.New("invalid fernet token signature")
	}

	ciphertext := token[1+8+fernetIVSize : len(token)-fernetHMACSize]
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("invalid fernet ciphertext length")
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create fernet cipher: %w", err)
	}
	plaintext := make([]byte, len(ciphertext))
	iv := token[1+8 : 1+8+fernetIVSize]
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)
	return pkcs7Unpad(plaintext, aes.BlockSize)
}

func pkcs7Pad(value []byte, blockSize int) []byte {
	padding := blockSize - len(value)%blockSize
	return append(value, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

func pkcs7Unpad(value []byte, blockSize int) ([]byte, error) {
	if len(value) == 0 || len(value)%blockSize != 0 {
		return nil, errors.New("invalid pkcs7 data length")
	}
	padding := int(value[len(value)-1])
	if padding == 0 || padding > blockSize || padding > len(value) {
		return nil, errors.New("invalid pkcs7 padding")
	}
	for _, item := range value[len(value)-padding:] {
		if int(item) != padding {
			return nil, errors.New("invalid pkcs7 padding")
		}
	}
	return value[:len(value)-padding], nil
}
