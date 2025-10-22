package structures

import "fmt"

type Partition struct {
	Part_status      [1]byte  // Estado de la partición
	Part_type        [1]byte  // Tipo de partición
	Part_fit         [1]byte  // Ajuste de la partición
	Part_start       int32    // Byte de inicio de la partición
	Part_size        int32    // Tamaño de la partición
	Part_name        [16]byte // Nombre de la partición
	Part_correlative int32    // Correlativo de la partición
	Part_id          [4]byte  // ID de la partición
}

/*
Part Status:
	N: Disponible
	0: Creado (no montado)
	1: Montado
*/

// CreatePartition crea una partición con los parámetros proporcionados
func (p *Partition) CreatePartition(partStart, partSize int, partType, partFit, partName string) {
	if p == nil {
		fmt.Println("Error: No se puede crear la partición, puntero nil")
		return
	}

	// Asignar status de la partición (creada pero no montada)
	p.Part_status[0] = '0'

	// Asignar el byte de inicio de la partición
	p.Part_start = int32(partStart)

	// Asignar el tamaño de la partición
	p.Part_size = int32(partSize)

	// Asignar el tipo de partición
	if len(partType) > 0 {
		p.Part_type[0] = partType[0]
	}

	// Asignar el ajuste de la partición
	if len(partFit) > 0 {
		p.Part_fit[0] = partFit[0]
	}

	// Asignar el nombre de la partición
	copy(p.Part_name[:], partName)

	// Inicializar correlativo como -1 (no montado)
	p.Part_correlative = -1

	// Inicializar ID vacío
	for i := range p.Part_id {
		p.Part_id[i] = 0
	}
}

// MountPartition monta una partición asignándole correlativo e ID
func (p *Partition) MountPartition(correlative int, id string) error {
	if p == nil {
		return fmt.Errorf("error: No se puede montar la partición, puntero nil")
	}

	// Asignar status de la partición (montada)
	p.Part_status[0] = '1'

	// Asignar correlativo a la partición
	p.Part_correlative = int32(correlative)

	// Asignar ID a la partición
	copy(p.Part_id[:], id)

	return nil
}

// UnmountPartition desmonta una partición (resetea correlativo a 0 según enunciado)
func (p *Partition) UnmountPartition() error {
	if p == nil {
		return fmt.Errorf("error: No se puede desmontar la partición, puntero nil")
	}

	// Cambiar status a creado (no montado)
	p.Part_status[0] = '0'

	// Resetear correlativo a 0 (según enunciado del Proyecto 2)
	p.Part_correlative = 0

	// Limpiar el ID
	for i := range p.Part_id {
		p.Part_id[i] = 0
	}

	return nil
}

// PrintPartition imprime los valores de la partición
func (p *Partition) PrintPartition() {
	if p == nil {
		fmt.Println("Error: No se puede imprimir la partición, puntero nil")
		return
	}

	fmt.Printf("Part_status: %c\n", p.Part_status[0])
	fmt.Printf("Part_type: %c\n", p.Part_type[0])
	fmt.Printf("Part_fit: %c\n", p.Part_fit[0])
	fmt.Printf("Part_start: %d\n", p.Part_start)
	fmt.Printf("Part_size: %d\n", p.Part_size)
	fmt.Printf("Part_name: %s\n", string(p.Part_name[:]))
	fmt.Printf("Part_correlative: %d\n", p.Part_correlative)
	fmt.Printf("Part_id: %s\n", string(p.Part_id[:]))
}
