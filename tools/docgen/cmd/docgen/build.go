package main

import (
	"docgen/internal/builder"
	"docgen/internal/scaner"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultOutputPath   = "output.json"
	sharedModelsRelPath = "docs/shared/models.yaml"
)

func buildServicesDocs() {
	servicesDir := defaultServicesDir
	if len(os.Args) > 2 {
		servicesDir = os.Args[2]
	}

	// Сканируем сервисы
	services, err := scaner.ScanServices(servicesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сканирования: %s\n", err)
		os.Exit(1)
	}

	if len(services) == 0 {
		fmt.Println("Сервисы с документацией не найдены.")
		return
	}

	fmt.Printf("Найдено сервисов: %d\n", len(services))

	// Путь к shared models — относительно корня проекта (на уровень выше services/)
	projectRoot := filepath.Dir(servicesDir)
	sharedModelsPath := filepath.Join(projectRoot, sharedModelsRelPath)

	// Сборка
	b := builder.New(services)
	output, err := b.Build(sharedModelsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сборки: %s\n", err)
		os.Exit(1)
	}

	// Запись JSON
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сериализации JSON: %s\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(defaultOutputPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка записи файла: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Собрано сервисов: %d\n", len(output.Services))
	fmt.Printf("Записано в: %s\n", defaultOutputPath)
}
