package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		runValidate()
	case "build":
		buildServicesDocs()
	default:
		fmt.Fprintf(os.Stderr, "Неизвестная команда: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Использование: docgen <команда> [аргументы]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Команды:")
	fmt.Fprintln(os.Stderr, "  validate [путь_к_services]   Валидация документации всех сервисов")
	fmt.Fprintln(os.Stderr, "  build    [путь_к_services]   Сборка JSON из документации сервисов")
}
