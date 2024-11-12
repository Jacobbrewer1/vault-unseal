package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"os"
	"strconv"
	"time"
)

var bytes = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}

func GetCryptoKey() string {
	key := os.Getenv("CRYPT_KEY")
	if key == "" {
		key = strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	return key
}

func Encrypt(text, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	plainText := []byte(text)
	cfb := cipher.NewCFBEncrypter(block, bytes)
	cipherText := make([]byte, len(plainText))
	cfb.XORKeyStream(cipherText, plainText)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func Decrypt(text, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	cipherText, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		panic(err)
	}
	cfb := cipher.NewCFBDecrypter(block, bytes)
	plainText := make([]byte, len(cipherText))
	cfb.XORKeyStream(plainText, cipherText)
	return string(plainText), nil
}
