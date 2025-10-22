package global

import (
	structures "backend/structures"
	"errors"
)

// Carnet de estudiante
const Carnet string = "39"

// Estructura para guardar información completa de particiones montadas
type MountedPartition struct {
	Id   string
	Name string
	Disk string
	Path string
}

// Declaración de las particiones montadas
var (
	MountedPartitions map[string]MountedPartition = make(map[string]MountedPartition)
)

// GetMountedPartition obtiene la partición montada con el id especificado
func GetMountedPartition(id string) (*structures.Partition, string, error) {
	mount, ok := MountedPartitions[id]
	if !ok {
		return nil, "", errors.New("la partición no está montada")
	}

	var mbr structures.MBR
	if err := mbr.Deserialize(mount.Path); err != nil {
		return nil, "", err
	}

	partition, err := mbr.GetPartitionByID(id)
	if partition == nil {
		return nil, "", err
	}

	return partition, mount.Path, nil
}
