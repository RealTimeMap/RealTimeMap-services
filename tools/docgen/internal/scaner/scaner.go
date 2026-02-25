package scaner

import (
	"fmt"
	"os"
	"path/filepath"
)

type ServiceDocs struct {
	Name string
	Dir  string

	// Пути до документации
	MetaPath   string
	ModelsPath string
	APIPath    string
}

func ScanServices(servicesDir string) ([]ServiceDocs, error) {
	entries, err := os.ReadDir(servicesDir)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать %s: %w", servicesDir, err)
	}

	var results []ServiceDocs

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		docsDir := filepath.Join(servicesDir, entry.Name(), "docs")

		// Проверяем, есть ли docs/
		info, err := os.Stat(docsDir)
		if err != nil || !info.IsDir() {
			continue
		}

		svc := ServiceDocs{
			Name: entry.Name(),
			Dir:  docsDir,
		}

		// Проверяем наличие каждого файла
		for _, f := range []struct {
			name string
			dest *string
		}{
			{"meta.yaml", &svc.MetaPath},
			{"models.yaml", &svc.ModelsPath},
			{"api.yaml", &svc.APIPath},
		} {
			path := filepath.Join(docsDir, f.name)
			if _, err := os.Stat(path); err == nil {
				*f.dest = path
			}
		}

		results = append(results, svc)
	}

	return results, nil
}
