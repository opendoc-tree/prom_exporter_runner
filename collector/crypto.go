package collector

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"golang.org/x/term"
)

var secret string
var secret_file string = ""

func LoadSecretKey() {
	file, err := os.ReadFile(secret_file)
	if err != nil {
		// default secret key
		secret = "agg3mmaa3ama13mm3maaaama12222agm"
		return
	}
	secret = string(file)
}

func CreateSecret() {
	stdin := int(syscall.Stdin)
	oldState, err := term.GetState(stdin)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		term.Restore(stdin, oldState)
		os.Exit(1)
	}()
	fmt.Print("Enter 32 bit secret key : ")
	password, err := term.ReadPassword(stdin)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// write secret key to file
	fileDir := filepath.Dir(secret_file)
	err = os.Mkdir(fileDir, 0755)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	err = os.WriteFile(secret_file, password, 0644)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	os.Exit(1)
}

func GetEncryptTxt() {
	LoadSecretKey()
	stdin := int(syscall.Stdin)
	oldState, err := term.GetState(stdin)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		term.Restore(stdin, oldState)
		os.Exit(1)
	}()
	fmt.Print("Enter text to encrypt : ")
	txt, err := term.ReadPassword(stdin)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	encryptTxt, _ := Encrypt(string(txt))
	fmt.Println()
	fmt.Println(encryptTxt)
	os.Exit(0)
}

// Encrypt encrypts a plaintext using AES-GCM with the provided key.
func Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher([]byte(secret))
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(plaintext))
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a ciphertext using AES-GCM with the provided key.
func Decrypt(encryptedText string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(secret))
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext is too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}
