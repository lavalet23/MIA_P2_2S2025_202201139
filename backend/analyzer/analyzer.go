package analyzer

import (
	commands "backend/commands"
	"errors"
	"fmt"
	"strings"
)

// 🔹 Función principal del analizador (no se modifica)
func Analyzer(input string) (string, error) {

	// Ignorar líneas de comentarios o vacías desde el principio
	if strings.TrimSpace(input) == "" || strings.HasPrefix(strings.TrimSpace(input), "#") {
		return "", nil
	}

	tokens := strings.Fields(input)

	if len(tokens) == 0 {
		return "", errors.New("no se proporcionó ningún comando")
	}

	switch tokens[0] {
	case "mkdisk":
		return commands.ParseMkdisk(tokens[1:])
	case "rmdisk":
		return commands.ParserRmdisk(tokens[1:])
	case "fdisk":
		return commands.ParseFdisk(tokens[1:])
	case "mount":
		return commands.ParseMount(tokens[1:])
	case "mkfs":
		return commands.ParseMkfs(tokens[1:])
	case "rep":
		return commands.ParseRep(tokens[1:])
	case "mkdir":
		return commands.ParseMkdir(tokens[1:])
	case "mkfile":
		return commands.ParserMkfile(tokens[1:])
	case "login":
		return commands.ParseLogin(tokens[1:])
	case "logout":
		return commands.ParseLogout(tokens[1:])
	case "mounted":
		return commands.ParseMounted(tokens[1:])
	case "unmount":
		return commands.ParseUnmount(tokens)
	case "mkgrp":
		return commands.ParseMkgrp(tokens[1:])
	case "cat":
		return commands.ParseCat(tokens[1:])
	default:
		return "", fmt.Errorf("comando desconocido: %s", tokens[0])
	}
}

// 🔹 Nueva función: permite ejecutar varios comandos seguidos
func AnalyzerMulti(input string) (string, error) {
	// Elimina espacios al inicio y final
	input = strings.TrimSpace(input)

	// Dividir por salto de línea o por ';'
	commandsList := strings.FieldsFunc(input, func(r rune) bool {
		return r == '\n' || r == ';'
	})

	var output strings.Builder

	for _, line := range commandsList {
		line = strings.TrimSpace(line)

		// Ignorar comentarios o líneas vacías
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		result, err := Analyzer(line)
		if err != nil {
			output.WriteString(fmt.Sprintf("❌ Error en línea '%s': %v\n", line, err))
		} else if result != "" {
			output.WriteString(fmt.Sprintf("%s\n", result))
		}
	}

	return output.String(), nil
}
