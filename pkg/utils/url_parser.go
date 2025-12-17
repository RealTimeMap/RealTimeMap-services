package utils

import (
	"net/url"
	"path"
	"strings"
)

// GetFileNameFromUrl парсит URL и возвращает имя файла
func GetFileNameFromUrl(rawUrl string) (string, error) {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}
	urlPath := parsedUrl.Path

	urlPath = strings.ReplaceAll(urlPath, "\\", "/")

	fileName := path.Base(urlPath)

	fileNameWithoutExt := strings.TrimSuffix(fileName, path.Ext(fileName))

	return fileNameWithoutExt, nil

}
