package commands

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	stores "backend/stores"
)

// ParseMounted procesa el comando 'mounted'
func ParseMounted(tokens []string) (string, error) {
	joined := strings.TrimSpace(strings.Join(tokens, " "))
	if joined != "" {
		return "", errors.New("parámetro desconocido para 'mounted'")
	}
	return commandMounted()
}

// commandMounted construye la salida del comando mostrando los IDs montados
func commandMounted() (string, error) {
	if len(stores.MountedPartitions) == 0 {
		return "No hay particiones montadas actualmente.", nil
	}

	ids := make([]string, 0, len(stores.MountedPartitions))
	for id := range stores.MountedPartitions {
		ids = append(ids, id)
	}
	// Ordenar para salida determinística
	sort.Strings(ids)

	return fmt.Sprintf("Particiones montadas: %s", strings.Join(ids, ", ")), nil
}
