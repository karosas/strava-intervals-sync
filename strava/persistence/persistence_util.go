package persistence

import (
	"log"
	"os"
	"path"
)

// WriteAccessToken writes token to file `access_token` inside configured `TOKEN_STORAGE_DIR`
// it's good enough for my personal use-case, but obviously storing access token in plain text on a file is not ideal
func WriteAccessToken(value string) error {
	return writeFile("access_token", value)
}

func ReadAccessToken() (string, error) {
	return readFile("access_token")
}

func WriteRefreshToken(value string) error {
	return writeFile("refresh_token", value)
}

func ReadRefreshToken() (string, error) {
	return readFile("refresh_token")
}

func writeFile(fileName string, value string) error {
	if err := os.MkdirAll(os.Getenv("TOKEN_STORAGE_DIR"), os.ModePerm); err != nil {
		log.Println("Failed to create token storage directory", err)
		return err
	}

	f, err := os.Create(path.Join(os.Getenv("TOKEN_STORAGE_DIR"), fileName))
	if err != nil {
		return err
	}

	if _, err = f.WriteString(value); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}

func readFile(fileName string) (string, error) {
	_, err := os.Stat(path.Join(os.Getenv("TOKEN_STORAGE_DIR"), fileName))
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path.Join(os.Getenv("TOKEN_STORAGE_DIR"), fileName))
	if err != nil {
		return "", err
	}

	return string(data), nil
}
