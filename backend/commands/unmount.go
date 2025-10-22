package commands

import (
	stores "backend/stores"
	structures "backend/structures"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// UNMOUNT estructura que representa el comando unmount
type UNMOUNT struct {
	id string // ID de la partición a desmontar
}

// ParseUnmount parsea el comando unmount y desmonta la partición
func ParseUnmount(tokens []string) (string, error) {
	cmd := &UNMOUNT{}

	args := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-id="[^"]+"|-id=[^\s]+`)
	matches := re.FindAllString(args, -1)

	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		if len(kv) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", match)
		}
		key, value := strings.ToLower(kv[0]), kv[1]

		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		switch key {
		case "-id":
			if value == "" {
				return "", errors.New("el ID no puede estar vacío")
			}
			cmd.id = strings.ToUpper(value)
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.id == "" {
		return "", errors.New("faltan parámetros requeridos: -id")
	}

	// Desmontar la partición
	return commandUnmount(cmd)
}

func commandUnmount(unmount *UNMOUNT) (string, error) {
	// Verificar que la partición esté montada
	diskPath, exists := stores.MountedPartitions[unmount.id]
	if !exists {
		return "", fmt.Errorf("ERROR: no existe una partición montada con el ID: %s", unmount.id)
	}

	// Leer el MBR del disco
	var mbr structures.MBR
	err := mbr.Deserialize(diskPath)
	if err != nil {
		return "", fmt.Errorf("error leyendo MBR: %v", err)
	}

	// Buscar la partición por ID
	partition, err := mbr.GetPartitionByID(unmount.id)
	if err != nil {
		return "", fmt.Errorf("ERROR: %v", err)
	}

	// Obtener el nombre de la partición antes de desmontarla (para el mensaje)
	partName := strings.TrimRight(string(partition.Part_name[:]), "\x00")

	// Desmontar la partición (cambiar estado y resetear correlativo)
	err = partition.UnmountPartition()
	if err != nil {
		return "", fmt.Errorf("error desmontando partición: %v", err)
	}

	// Encontrar el índice de la partición en el MBR y actualizarla
	for i := 0; i < 4; i++ {
		currentID := strings.TrimRight(string(mbr.Mbr_partitions[i].Part_id[:]), "\x00")
		if currentID == unmount.id {
			mbr.Mbr_partitions[i] = *partition
			break
		}
	}

	// Serializar el MBR actualizado
	err = mbr.Serialize(diskPath)
	if err != nil {
		return "", fmt.Errorf("error guardando MBR: %v", err)
	}

	// Eliminar de las particiones montadas
	delete(stores.MountedPartitions, unmount.id)

	return fmt.Sprintf("UNMOUNT: Partición '%s' desmontada exitosamente\n"+
		"-> ID: %s\n"+
		"-> Path: %s",
		partName, unmount.id, diskPath), nil
}
