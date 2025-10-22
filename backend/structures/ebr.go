package structures

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

// EBR (Extended Boot Record) - Descriptor de partición lógica
// Según el enunciado del proyecto
type EBR struct {
	Part_mount [1]byte  // Indica si la partición está montada o no
	Part_fit   [1]byte  // Tipo de ajuste de la partición (B=Best, F=First, W=Worst)
	Part_start int32    // Indica en qué byte del disco inicia la partición
	Part_size  int32    // Contiene el tamaño total de la partición en bytes
	Part_next  int32    // Byte en el que está el próximo EBR. -1 si no hay siguiente
	Part_name  [16]byte // Nombre de la partición
}

// PrintEBR imprime la información del EBR
func (ebr *EBR) PrintEBR() {
	fmt.Println("╔════════════════════════════════════╗")
	fmt.Println("║          EBR INFORMATION           ║")
	fmt.Println("╠════════════════════════════════════╣")
	fmt.Printf("║ Mount Status: %-20c║\n", ebr.Part_mount[0])
	fmt.Printf("║ Fit:          %-20c║\n", ebr.Part_fit[0])
	fmt.Printf("║ Start:        %-20d║\n", ebr.Part_start)
	fmt.Printf("║ Size:         %-20d║\n", ebr.Part_size)
	fmt.Printf("║ Next:         %-20d║\n", ebr.Part_next)

	// Limpiar el nombre
	name := string(ebr.Part_name[:])
	if idx := len(name); idx > 0 {
		for i := 0; i < len(name); i++ {
			if name[i] == 0 {
				name = name[:i]
				break
			}
		}
	}
	fmt.Printf("║ Name:         %-20s║\n", name)
	fmt.Println("╚════════════════════════════════════╝")
}

// Serialize escribe el EBR en un archivo binario en una posición específica
func (ebr *EBR) Serialize(path string, position int) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo archivo para escribir EBR: %w", err)
	}
	defer file.Close()

	// Posicionar en la ubicación del EBR
	if _, err := file.Seek(int64(position), 0); err != nil {
		return fmt.Errorf("error posicionando en el archivo: %w", err)
	}

	// Escribir la estructura EBR
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.LittleEndian, ebr); err != nil {
		return fmt.Errorf("error serializando EBR: %w", err)
	}

	if _, err := file.Write(buffer.Bytes()); err != nil {
		return fmt.Errorf("error escribiendo EBR: %w", err)
	}

	return file.Sync()
}

// Deserialize lee el EBR desde un archivo binario en una posición específica
func (ebr *EBR) Deserialize(path string, position int) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error abriendo archivo para leer EBR: %w", err)
	}
	defer file.Close()

	// Posicionar en la ubicación del EBR
	if _, err := file.Seek(int64(position), 0); err != nil {
		return fmt.Errorf("error posicionando en el archivo: %w", err)
	}

	// Leer la estructura EBR
	if err := binary.Read(file, binary.LittleEndian, ebr); err != nil {
		return fmt.Errorf("error deserializando EBR: %w", err)
	}

	return nil
}

// CreateEBR crea un nuevo EBR con los parámetros dados
func (ebr *EBR) CreateEBR(start int, size int, fit string, name string, next int) {
	ebr.Part_mount = [1]byte{'1'}
	ebr.Part_fit = [1]byte{fit[0]}
	ebr.Part_start = int32(start)
	ebr.Part_size = int32(size)
	ebr.Part_next = int32(next)
	copy(ebr.Part_name[:], name)
}

// IsEmpty verifica si el EBR está vacío
func (ebr *EBR) IsEmpty() bool {
	return ebr.Part_start == -1 || ebr.Part_size == 0
}

// Clear limpia el EBR (lo marca como vacío)
func (ebr *EBR) Clear() {
	ebr.Part_mount = [1]byte{'0'}
	ebr.Part_fit = [1]byte{'0'}
	ebr.Part_start = -1
	ebr.Part_size = 0
	ebr.Part_next = -1
	for i := range ebr.Part_name {
		ebr.Part_name[i] = 0
	}
}
