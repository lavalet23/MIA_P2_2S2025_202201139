package stores

import (
	structures "backend/structures"
	"errors"
	"fmt"
	"strings"
)

// Carnet de estudiante
const Carnet string = "39" // 202201139

// Declaración de variables globales
var (
	MountedPartitions map[string]string = make(map[string]string)
)

func GetMountedPartition(id string) (*structures.Partition, string, error) {
	// Normalizar id
	id = strings.ToUpper(strings.TrimSpace(id))

	// Obtener el path de la partición montada
	path := MountedPartitions[id]
	if path == "" {
		return nil, "", errors.New("la partición no está montada")
	}

	var mbr structures.MBR
	err := mbr.Deserialize(path)
	if err != nil {
		return nil, "", err
	}

	partition, err := mbr.GetPartitionByID(id)
	if partition == nil {
		return nil, "", err
	}

	return partition, path, nil
}

func GetMountedPartitionRep(id string) (*structures.MBR, *structures.SuperBlock, string, error) {
	// Normalizar id
	id = strings.ToUpper(strings.TrimSpace(id))

	path := MountedPartitions[id]
	if path == "" {
		return nil, nil, "", errors.New("la partición no está montada")
	}

	var mbr structures.MBR
	err := mbr.Deserialize(path)
	if err != nil {
		return nil, nil, "", err
	}

	partition, err := mbr.GetPartitionByID(id)
	if partition == nil {
		return nil, nil, "", err
	}

	var sb structures.SuperBlock
	err = sb.Deserialize(path, int64(partition.Part_start))
	if err != nil {
		return nil, nil, "", err
	}

	return &mbr, &sb, path, nil
}

func GetMountedPartitionSuperblock(id string) (*structures.SuperBlock, *structures.Partition, string, error) {
	// Normalizar id
	id = strings.ToUpper(strings.TrimSpace(id))

	fmt.Println("DEBUG -> Buscando id:", id)
	fmt.Println("DEBUG -> MountedPartitions actuales:", MountedPartitions)

	path := MountedPartitions[id]
	if path == "" {
		return nil, nil, "", errors.New("la partición no está montada")
	}

	var mbr structures.MBR
	err := mbr.Deserialize(path)
	if err != nil {
		return nil, nil, "", err
	}

	partition, err := mbr.GetPartitionByID(id)
	if partition == nil {
		return nil, nil, "", err
	}

	var sb structures.SuperBlock
	err = sb.Deserialize(path, int64(partition.Part_start))
	if err != nil {
		return nil, nil, "", err
	}

	return &sb, partition, path, nil
}
