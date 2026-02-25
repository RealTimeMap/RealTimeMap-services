package main

import (
	"docgen/internal/models"
	"docgen/internal/scaner"
	"docgen/internal/validator"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const defaultServicesDir = "../../services"

func runValidate() {
	servicesDir := defaultServicesDir
	if len(os.Args) > 2 {
		servicesDir = os.Args[2]
	}

	services, err := scaner.ScanServices(servicesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сканирования: %s\n", err)
		os.Exit(1)
	}

	if len(services) == 0 {
		fmt.Println("Сервисы с документацией не найдены.")
		return
	}

	fmt.Printf("Найдено сервисов: %d\n\n", len(services))

	totalErrors := 0

	for _, svc := range services {
		svcErrors := validateService(svc)
		totalErrors += svcErrors
	}

	fmt.Println("---")
	if totalErrors == 0 {
		fmt.Println("Все сервисы прошли валидацию!")
	} else {
		fmt.Printf("Всего ошибок: %d\n", totalErrors)
		os.Exit(1)
	}
}

func validateService(svc scaner.ServiceDocs) int {
	fmt.Printf("=== %s ===\n", svc.Name)

	errorCount := 0

	// Валидация meta.yaml
	if svc.MetaPath == "" {
		fmt.Println("  [WARN] meta.yaml не найден")
	} else {
		errs := validateMetaFromFile(svc.MetaPath)
		errorCount += printFileErrors("meta.yaml", errs)
	}

	// Валидация models.yaml
	if svc.ModelsPath == "" {
		fmt.Println("  [WARN] models.yaml не найден")
	} else {
		errs := validateModelsFromFile(svc.ModelsPath)
		errorCount += printFileErrors("models.yaml", errs)
	}

	// api.yaml — пока только проверяем наличие
	if svc.APIPath == "" {
		fmt.Println("  [WARN] api.yaml не найден")
	} else {
		fmt.Println("  [OK]   api.yaml найден")
	}

	fmt.Println()
	return errorCount
}

func validateMetaFromFile(path string) []error {
	data, err := os.ReadFile(path)
	if err != nil {
		return []error{fmt.Errorf("не удалось прочитать файл: %w", err)}
	}

	var meta models.Meta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return []error{fmt.Errorf("ошибка парсинга YAML: %w", err)}
	}

	return validator.ValidateMeta(meta)
}

func validateModelsFromFile(path string) []error {
	data, err := os.ReadFile(path)
	if err != nil {
		return []error{fmt.Errorf("не удалось прочитать файл: %w", err)}
	}

	var modelMap map[string]models.Model
	if err := yaml.Unmarshal(data, &modelMap); err != nil {
		return []error{fmt.Errorf("ошибка парсинга YAML: %w", err)}
	}

	return validator.ValidateModels(modelMap)
}

func printFileErrors(filename string, errs []error) int {
	if len(errs) == 0 {
		fmt.Printf("  [OK]   %s\n", filename)
		return 0
	}

	fmt.Printf("  [FAIL] %s — ошибок: %d\n", filename, len(errs))
	for _, e := range errs {
		fmt.Printf("         - %s\n", e.Error())
	}
	return len(errs)
}
