package commands

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type RMDISK struct {
	path string
}

// ParserRmdisk ejecuta el comando RMDISK conforme al Proyecto 2 (sin confirmación interactiva).
func ParserRmdisk(tokens []string) (string, error) {
	cmd := &RMDISK{}

	args := strings.Join(tokens, " ")

	// Regex que soporta -path="..." o -path=/ruta
	re := regexp.MustCompile(`-(?i)path=(?:"[^"]+"|\S+)`)
	matches := re.FindAllString(args, -1)

	if len(matches) == 0 {
		return "", errors.New("ERROR: faltan parámetros requeridos: -path")
	}

	// Validar parámetros desconocidos
	for _, token := range tokens {
		if !strings.HasPrefix(strings.ToLower(token), "-path=") {
			if strings.HasPrefix(token, "-") {
				return "", fmt.Errorf("ERROR: parámetro desconocido: %s", token)
			}
		}
	}

	// Procesar parámetro -path
	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		if len(kv) != 2 {
			return "", fmt.Errorf("ERROR: formato de parámetro inválido: %s", match)
		}

		value := kv[1]
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		if value == "" {
			return "", errors.New("ERROR: el path no puede estar vacío")
		}

		cmd.path = value
	}

	// Verificar existencia del disco
	if _, err := os.Stat(cmd.path); os.IsNotExist(err) {
		return "", fmt.Errorf("ERROR: el disco no existe en la ruta indicada -> %s", cmd.path)
	}

	// Intentar eliminar disco
	if err := os.Remove(cmd.path); err != nil {
		return "", fmt.Errorf("ERROR: no se pudo eliminar el disco: %w", err)
	}

	// Mensaje limpio y claro para el script
	return fmt.Sprintf("RMDISK: Disco eliminado correctamente -> Path: %s", cmd.path), nil
}
