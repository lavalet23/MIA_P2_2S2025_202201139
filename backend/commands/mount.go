package commands

import (
	stores "backend/stores"
	structures "backend/structures"
	utils "backend/utils"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// MOUNT estructura que representa el comando mount con sus parámetros
type MOUNT struct {
	path string // Ruta del archivo del disco
	name string // Nombre de la partición
}

// ParseMount parsea el comando mount y devuelve una instancia de MOUNT
func ParseMount(tokens []string) (string, error) {
	cmd := &MOUNT{}

	args := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-name="[^"]+"|-name=[^\s]+`)
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
		case "-path":
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		case "-name":
			if value == "" {
				return "", errors.New("el nombre no puede estar vacío")
			}
			cmd.name = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}
	if cmd.name == "" {
		return "", errors.New("faltan parámetros requeridos: -name")
	}

	// Montamos la partición
	idPartition, err := commandMount(cmd)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("MOUNT: Partición montada exitosamente\n"+
		"-> Path: %s\n"+
		"-> Nombre: %s\n"+
		"-> ID: %s",
		cmd.path, cmd.name, idPartition), nil
}

func commandMount(mount *MOUNT) (string, error) {
	var mbr structures.MBR

	// Deserializar MBR
	err := mbr.Deserialize(mount.path)
	if err != nil {
		return "", fmt.Errorf("error leyendo MBR: %v", err)
	}

	// Buscar la partición con el nombre especificado
	partition, indexPartition := mbr.GetPartitionByName(mount.name)
	if partition == nil {
		return "", errors.New("ERROR: la partición no existe")
	}

	// VALIDACIÓN: Solo se pueden montar particiones PRIMARIAS
	if partition.Part_type[0] != 'P' {
		return "", errors.New("ERROR: solo se pueden montar particiones primarias")
	}

	// VALIDACIÓN: Verificar que la partición no esté ya montada
	if partition.Part_status[0] == '1' {
		return "", fmt.Errorf("ERROR: la partición '%s' ya está montada", mount.name)
	}

	// VALIDACIÓN ADICIONAL: Verificar en MountedPartitions si ya está montada
	existingID := strings.TrimRight(string(partition.Part_id[:]), "\x00")
	if existingID != "" {
		// Verificar si este ID existe en las particiones montadas
		if _, exists := stores.MountedPartitions[existingID]; exists {
			return "", fmt.Errorf("ERROR: la partición '%s' ya está montada con ID: %s", mount.name, existingID)
		}
	}

	// Generar un ID único para la partición
	idPartition, partitionCorrelative, err := generatePartitionID(mount)
	if err != nil {
		return "", fmt.Errorf("error generando ID de partición: %v", err)
	}

	// Normalizar ID a mayúsculas
	idPartition = strings.ToUpper(strings.TrimSpace(idPartition))

	// Guardar la partición montada en la lista de montajes globales
	stores.MountedPartitions[idPartition] = mount.path

	// Modificar la partición para indicar que está montada
	partition.MountPartition(partitionCorrelative, idPartition)

	// Guardar la partición modificada en el MBR
	mbr.Mbr_partitions[indexPartition] = *partition

	// Serializar MBR
	err = mbr.Serialize(mount.path)
	if err != nil {
		return "", fmt.Errorf("error guardando MBR: %v", err)
	}

	return idPartition, nil
}

func generatePartitionID(mount *MOUNT) (string, int, error) {
	// Asignar letra y obtener correlativo de partición
	letter, partitionCorrelative, err := utils.GetLetterAndPartitionCorrelative(mount.path)
	if err != nil {
		return "", 0, err
	}

	// Crear ID de partición: últimos 2 dígitos del carnet + número partición + letra
	// Ejemplo: carnet = 202401234 -> 34 + 1 + A = 341A
	idPartition := fmt.Sprintf("%s%d%s", stores.Carnet, partitionCorrelative, strings.ToUpper(letter))

	return idPartition, partitionCorrelative, nil
}
