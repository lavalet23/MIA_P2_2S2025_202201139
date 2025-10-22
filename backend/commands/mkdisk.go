package commands

import (
	structures "backend/structures"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type MKDISK struct {
	size int
	unit string
	fit  string
	path string
}

func ParseMkdisk(tokens []string) (string, error) {
	cmd := &MKDISK{}

	args := strings.Join(tokens, " ")

	// Regex para -clave=valor
	re := regexp.MustCompile(`-(\w+)=(".*?"|\S+)`)
	matches := re.FindAllStringSubmatch(args, -1)

	if len(matches) == 0 {
		return "", errors.New("ERROR: no se detectaron parámetros válidos para mkdisk")
	}

	allowed := map[string]bool{
		"-size": true,
		"-unit": true,
		"-fit":  true,
		"-path": true,
	}

	// Validar si hay parámetros desconocidos
	for _, m := range matches {
		key := "-" + strings.ToLower(m[1])
		if _, ok := allowed[key]; !ok {
			return "", fmt.Errorf("ERROR: parámetro desconocido: %s", key)
		}
	}

	for _, m := range matches {
		key := "-" + strings.ToLower(m[1])
		value := m[2]
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") && len(value) >= 2 {
			value = value[1 : len(value)-1]
		}

		if _, ok := allowed[key]; !ok {
			return "", fmt.Errorf("ERROR: parámetro desconocido: %s", key)
		}

		switch key {
		case "-size":
			sz, err := strconv.Atoi(value)
			if err != nil || sz <= 0 {
				return "", errors.New("ERROR: el tamaño debe ser un número entero positivo mayor que cero")
			}
			cmd.size = sz
		case "-unit":
			valUp := strings.ToUpper(value)
			if valUp != "K" && valUp != "M" {
				return "", errors.New("ERROR: la unidad debe ser K o M")
			}
			cmd.unit = valUp
		case "-fit":
			valUp := strings.ToUpper(value)
			if valUp != "BF" && valUp != "FF" && valUp != "WF" {
				return "", errors.New("ERROR: el ajuste debe ser BF, FF o WF")
			}
			cmd.fit = valUp
		case "-path":
			if value == "" {
				return "", errors.New("ERROR: el path no puede estar vacío")
			}
			if !filepath.IsAbs(value) {
				return "", errors.New("ERROR: el path debe ser absoluto")
			}
			cmd.path = value
		}
	}

	// Validación de parámetros obligatorios
	if cmd.size == 0 {
		return "", errors.New("ERROR: faltan parámetros requeridos: -size")
	}
	if cmd.path == "" {
		return "", errors.New("ERROR: faltan parámetros requeridos: -path")
	}

	// Defaults
	if cmd.unit == "" {
		cmd.unit = "M"
	}
	if cmd.fit == "" {
		cmd.fit = "FF"
	}

	// Crear disco
	if err := commandMkdisk(cmd); err != nil {
		return "", err
	}

	return fmt.Sprintf("MKDISK: Disco creado exitosamente\n-> Path: %s\n-> Tamaño: %d%s\n-> Fit: %s\n",
		cmd.path, cmd.size, cmd.unit, cmd.fit), nil
}

func commandMkdisk(mkdisk *MKDISK) error {
	sizeBytes, err := convertSizeToBytes(mkdisk.size, mkdisk.unit)
	if err != nil {
		return err
	}

	if err := createDisk(mkdisk, sizeBytes); err != nil {
		return err
	}

	if err := createMBR(mkdisk, sizeBytes); err != nil {
		_ = os.Remove(mkdisk.path)
		return err
	}

	// *** MODIFICACIÓN AQUÍ ***
	// Crear una nueva instancia de MBR para deserializar y luego imprimir
	newMBR := &structures.MBR{}
	if err := newMBR.Deserialize(mkdisk.path); err != nil {
		// Manejar el error de deserialización si ocurre
		return fmt.Errorf("ERROR: no se pudo deserializar el MBR para imprimirlo: %w", err)
	}

	// Imprimir la información del MBR
	fmt.Println("--- MBR Creado ---")
	newMBR.PrintMBR()        // Imprime los detalles del MBR
	newMBR.PrintPartitions() // Opcionalmente, imprime los detalles de las particiones (que estarán vacías inicialmente)
	fmt.Println("------------------")
	// *** FIN DE LA MODIFICACIÓN ***

	return nil
}

func convertSizeToBytes(size int, unit string) (int64, error) {
	switch strings.ToUpper(unit) {
	case "K":
		return int64(size) * 1024, nil
	case "M":
		return int64(size) * 1024 * 1024, nil
	default:
		return 0, errors.New("ERROR: unidad desconocida, use K o M")
	}
}

func createDisk(mkdisk *MKDISK, sizeBytes int64) error {
	if err := os.MkdirAll(filepath.Dir(mkdisk.path), 0o755); err != nil {
		return fmt.Errorf("ERROR: error creando carpetas padres: %w", err)
	}

	if _, err := os.Stat(mkdisk.path); err == nil {
		return fmt.Errorf("ERROR: ya existe un disco en la ruta especificada: %s", mkdisk.path)
	}

	f, err := os.Create(mkdisk.path)
	if err != nil {
		return fmt.Errorf("ERROR: error creando archivo: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 1024)
	remaining := sizeBytes
	for remaining > 0 {
		toWrite := int64(len(buf))
		if remaining < toWrite {
			toWrite = remaining
		}
		n, err := f.Write(buf[:toWrite])
		if err != nil {
			return fmt.Errorf("ERROR: error escribiendo archivo: %w", err)
		}
		if int64(n) != toWrite {
			return fmt.Errorf("ERROR: escritura incompleta")
		}
		remaining -= toWrite
	}

	return f.Sync()
}

func createMBR(mkdisk *MKDISK, sizeBytes int64) error {
	var fitByte byte
	switch mkdisk.fit {
	case "FF":
		fitByte = 'F'
	case "BF":
		fitByte = 'B'
	case "WF":
		fitByte = 'W'
	}

	mbr := &structures.MBR{
		Mbr_size:           int32(sizeBytes),
		Mbr_creation_date:  float32(time.Now().Unix()),
		Mbr_disk_signature: rand.Int31(),
		Mbr_disk_fit:       [1]byte{fitByte},
		Mbr_partitions:     [4]structures.Partition{},
	}

	for i := 0; i < len(mbr.Mbr_partitions); i++ {
		mbr.Mbr_partitions[i].Part_start = -1
		mbr.Mbr_partitions[i].Part_size = 0
		mbr.Mbr_partitions[i].Part_status = [1]byte{'0'}
		mbr.Mbr_partitions[i].Part_type = [1]byte{'0'}
		mbr.Mbr_partitions[i].Part_fit = [1]byte{'0'}
		for j := range mbr.Mbr_partitions[i].Part_name {
			mbr.Mbr_partitions[i].Part_name[j] = 0
		}
		for j := range mbr.Mbr_partitions[i].Part_id {
			mbr.Mbr_partitions[i].Part_id[j] = 0
		}
	}

	return mbr.Serialize(mkdisk.path)
}
