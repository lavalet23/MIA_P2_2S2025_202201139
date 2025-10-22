package commands

import (
	structures "backend/structures"
	utils "backend/utils"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// FDISK estructura que representa el comando fdisk con sus parámetros
type FDISK struct {
	size   int    // Tamaño de la partición
	unit   string // Unidad de medida del tamaño (B, K o M)
	fit    string // Tipo de ajuste (BF, FF, WF)
	path   string // Ruta del archivo del disco
	typ    string // Tipo de partición (P, E, L)
	name   string // Nombre de la partición
	delete string // Tipo de eliminación (fast, full)
	add    int    // Espacio a agregar o quitar (puede ser negativo)
}

// ParseFdisk parsea el comando fdisk y ejecuta la operación correspondiente
func ParseFdisk(tokens []string) (string, error) {
	cmd := &FDISK{}

	args := strings.Join(tokens, " ")

	// Regex mejorada para capturar todos los parámetros
	re := regexp.MustCompile(`-(\w+)=(-?\d+|-?\d+\.\d+|"[^"]+"|[^\s]+)`)
	matches := re.FindAllStringSubmatch(args, -1)

	if len(matches) == 0 {
		return "", errors.New("ERROR: no se detectaron parámetros válidos para fdisk")
	}

	for _, match := range matches {
		key := strings.ToLower(match[1])
		value := match[2]

		// Remover comillas si existen
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		switch key {
		case "size":
			size, err := strconv.Atoi(value)
			if err != nil || size <= 0 {
				return "", errors.New("ERROR: el tamaño debe ser un número entero positivo")
			}
			cmd.size = size

		case "unit":
			val := strings.ToUpper(value)
			if val != "B" && val != "K" && val != "M" {
				return "", errors.New("ERROR: la unidad debe ser B, K o M")
			}
			cmd.unit = val

		case "fit":
			value = strings.ToUpper(value)
			if value != "BF" && value != "FF" && value != "WF" {
				return "", errors.New("ERROR: el ajuste debe ser BF, FF o WF")
			}
			cmd.fit = value

		case "path":
			if value == "" {
				return "", errors.New("ERROR: el path no puede estar vacío")
			}
			cmd.path = value

		case "type":
			value = strings.ToUpper(value)
			if value != "P" && value != "E" && value != "L" {
				return "", errors.New("ERROR: el tipo debe ser P, E o L")
			}
			cmd.typ = value

		case "name":
			if value == "" {
				return "", errors.New("ERROR: el nombre no puede estar vacío")
			}
			cmd.name = value

		case "delete":
			value = strings.ToLower(value)
			if value != "fast" && value != "full" {
				return "", errors.New("ERROR: delete debe ser 'fast' o 'full'")
			}
			cmd.delete = value

		case "add":
			add, err := strconv.Atoi(value)
			if err != nil {
				return "", errors.New("ERROR: add debe ser un número entero")
			}
			cmd.add = add

		default:
			return "", fmt.Errorf("ERROR: parámetro desconocido: -%s", key)
		}
	}

	// Validar que path y name siempre estén presentes
	if cmd.path == "" {
		return "", errors.New("ERROR: faltan parámetros requeridos: -path")
	}
	if cmd.name == "" {
		return "", errors.New("ERROR: faltan parámetros requeridos: -name")
	}

	// Verificar que el archivo existe
	if _, err := os.Stat(cmd.path); os.IsNotExist(err) {
		return "", fmt.Errorf("ERROR: el disco no existe en la ruta: %s", cmd.path)
	}

	// Determinar la operación a realizar
	if cmd.delete != "" {
		return deletePartition(cmd)
	} else if cmd.add != 0 {
		return modifyPartitionSize(cmd)
	} else {
		if cmd.size == 0 {
			return "", errors.New("ERROR: faltan parámetros requeridos: -size")
		}
		return createPartition(cmd)
	}
}

// createPartition crea una nueva partición
func createPartition(cmd *FDISK) (string, error) {
	// Defaults
	if cmd.unit == "" {
		cmd.unit = "K"
	}
	if cmd.fit == "" {
		cmd.fit = "WF"
	}
	if cmd.typ == "" {
		cmd.typ = "P"
	}

	// Convertir tamaño a bytes
	sizeBytes, err := utils.ConvertToBytes(cmd.size, cmd.unit)
	if err != nil {
		return "", fmt.Errorf("ERROR: error convirtiendo tamaño: %v", err)
	}

	// Leer MBR
	var mbr structures.MBR
	if err := mbr.Deserialize(cmd.path); err != nil {
		return "", fmt.Errorf("ERROR: error leyendo MBR: %v", err)
	}

	// Verificar que no exista una partición con el mismo nombre en particiones primarias/extendidas
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start != -1 {
			partName := strings.TrimRight(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00")
			if partName == cmd.name {
				return "", fmt.Errorf("ERROR: ya existe una partición con el nombre '%s'", cmd.name)
			}
		}
	}

	// Verificar nombres en particiones lógicas si aplica
	if err := checkLogicalPartitionNames(cmd, &mbr); err != nil {
		return "", err
	}

	// Crear según el tipo
	switch cmd.typ {
	case "P":
		return createPrimaryPartition(cmd, &mbr, sizeBytes)
	case "E":
		return createExtendedPartition(cmd, &mbr, sizeBytes)
	case "L":
		return createLogicalPartition(cmd, &mbr, sizeBytes)
	default:
		return "", errors.New("ERROR: tipo de partición no válido")
	}
}

// checkLogicalPartitionNames verifica que el nombre no exista en particiones lógicas
func checkLogicalPartitionNames(cmd *FDISK, mbr *structures.MBR) error {
	// Buscar la partición extendida
	extIndex := -1
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start != -1 && mbr.Mbr_partitions[i].Part_type[0] == 'E' {
			extIndex = i
			break
		}
	}

	if extIndex == -1 {
		return nil // No hay partición extendida, no hay particiones lógicas
	}

	extended := &mbr.Mbr_partitions[extIndex]
	extStart := int(extended.Part_start)

	// Leer el primer EBR
	currentEBR := structures.EBR{}
	if err := currentEBR.Deserialize(cmd.path, extStart); err != nil {
		return nil // Si no se puede leer, no hay lógicas
	}

	// Recorrer todos los EBRs
	currentPos := extStart
	for {
		if currentEBR.Part_start != -1 {
			ebrName := strings.TrimRight(string(currentEBR.Part_name[:]), "\x00")
			if ebrName == cmd.name {
				return fmt.Errorf("ERROR: ya existe una partición lógica con el nombre '%s'", cmd.name)
			}
		}

		if currentEBR.Part_next == -1 {
			break
		}

		currentPos = int(currentEBR.Part_next)
		if err := currentEBR.Deserialize(cmd.path, currentPos); err != nil {
			break
		}
	}

	return nil
}

// createPrimaryPartition crea una partición primaria
func createPrimaryPartition(cmd *FDISK, mbr *structures.MBR, sizeBytes int) (string, error) {
	// Contar TODAS las particiones que tienen espacio asignado (Part_start != -1)
	// Esto incluye primarias Y extendidas
	usedSlots := 0
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start != -1 {
			usedSlots++
		}
	}

	// Validar límite: máximo 4 espacios en el MBR
	if usedSlots >= 4 {
		return "", errors.New("ERROR: no se pueden crear más particiones (límite: 4 primarias/extendidas)")
	}

	// Buscar espacio disponible según el ajuste
	startByte, err := findAvailableSpace(mbr, sizeBytes, cmd.fit)
	if err != nil {
		return "", err
	}

	// Buscar la primera partición disponible en el MBR
	partIndex := -1
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start == -1 {
			partIndex = i
			break
		}
	}

	if partIndex == -1 {
		return "", errors.New("ERROR: no hay espacios disponibles en el MBR")
	}

	// Crear la partición usando el método CreatePartition
	mbr.Mbr_partitions[partIndex].CreatePartition(startByte, sizeBytes, "P", cmd.fit, cmd.name)

	// Serializar el MBR
	if err := mbr.Serialize(cmd.path); err != nil {
		return "", fmt.Errorf("ERROR: error escribiendo MBR: %v", err)
	}

	msg := fmt.Sprintf("FDISK: Partición primaria '%s' creada exitosamente\n"+
		"-> Tamaño: %d bytes\n"+
		"-> Inicio: %d\n"+
		"-> Fit: %s",
		cmd.name, sizeBytes, startByte, cmd.fit)

	return msg, nil
}

// createExtendedPartition crea una partición extendida
func createExtendedPartition(cmd *FDISK, mbr *structures.MBR, sizeBytes int) (string, error) {
	// Verificar que no exista ya una partición extendida
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start != -1 && mbr.Mbr_partitions[i].Part_type[0] == 'E' {
			return "", errors.New("ERROR: ya existe una partición extendida en el disco")
		}
	}

	// Contar particiones que ocupan slots (Part_start != -1)
	usedSlots := 0
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start != -1 {
			usedSlots++
		}
	}

	if usedSlots >= 4 {
		return "", errors.New("ERROR: no se pueden crear más particiones (límite: 4)")
	}

	// Buscar espacio disponible
	startByte, err := findAvailableSpace(mbr, sizeBytes, cmd.fit)
	if err != nil {
		return "", err
	}

	// Buscar la primera partición disponible
	partIndex := -1
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start == -1 {
			partIndex = i
			break
		}
	}

	if partIndex == -1 {
		return "", errors.New("ERROR: no hay espacios disponibles en el MBR")
	}

	// Crear la partición extendida
	mbr.Mbr_partitions[partIndex].CreatePartition(startByte, sizeBytes, "E", cmd.fit, cmd.name)

	// Crear el primer EBR vacío
	ebr := structures.EBR{
		Part_mount: [1]byte{'0'},
		Part_fit:   [1]byte{cmd.fit[0]},
		Part_start: -1,
		Part_size:  0,
		Part_next:  -1,
	}

	// Escribir el EBR al inicio de la partición extendida
	if err := ebr.Serialize(cmd.path, startByte); err != nil {
		return "", fmt.Errorf("ERROR: error escribiendo EBR inicial: %v", err)
	}

	// Serializar el MBR
	if err := mbr.Serialize(cmd.path); err != nil {
		return "", fmt.Errorf("ERROR: error escribiendo MBR: %v", err)
	}

	msg := fmt.Sprintf("FDISK: Partición extendida '%s' creada exitosamente\n"+
		"-> Tamaño: %d bytes\n"+
		"-> Inicio: %d",
		cmd.name, sizeBytes, startByte)

	return msg, nil
}

// createLogicalPartition crea una partición lógica dentro de la extendida
func createLogicalPartition(cmd *FDISK, mbr *structures.MBR, sizeBytes int) (string, error) {
	// Buscar la partición extendida
	extIndex := -1
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start != -1 && mbr.Mbr_partitions[i].Part_type[0] == 'E' {
			extIndex = i
			break
		}
	}

	if extIndex == -1 {
		return "", errors.New("ERROR: no existe una partición extendida para crear particiones lógicas")
	}

	extended := &mbr.Mbr_partitions[extIndex]
	extStart := int(extended.Part_start)
	extSize := int(extended.Part_size)
	ebrSize := binary.Size(structures.EBR{})

	// Verificar que haya espacio suficiente (al menos para EBR + datos)
	if sizeBytes+ebrSize > extSize {
		return "", errors.New("ERROR: no hay suficiente espacio en la partición extendida")
	}

	// Leer el primer EBR
	currentEBR := structures.EBR{}
	if err := currentEBR.Deserialize(cmd.path, extStart); err != nil {
		return "", fmt.Errorf("ERROR: error leyendo primer EBR: %v", err)
	}

	// Si el primer EBR está vacío (primera partición lógica)
	if currentEBR.Part_start == -1 {
		currentEBR.Part_mount = [1]byte{'1'}
		currentEBR.Part_fit = [1]byte{cmd.fit[0]}
		currentEBR.Part_start = int32(extStart + ebrSize)
		currentEBR.Part_size = int32(sizeBytes)
		currentEBR.Part_next = -1
		copy(currentEBR.Part_name[:], cmd.name)

		// Escribir el EBR actualizado
		if err := currentEBR.Serialize(cmd.path, extStart); err != nil {
			return "", fmt.Errorf("ERROR: error escribiendo EBR: %v", err)
		}

		msg := fmt.Sprintf("FDISK: Partición lógica '%s' creada exitosamente\n"+
			"-> Tamaño: %d bytes\n"+
			"-> Inicio: %d",
			cmd.name, sizeBytes, extStart+ebrSize)

		return msg, nil
	}

	// Recorrer la lista de EBRs para encontrar el último
	currentPos := extStart
	for currentEBR.Part_next != -1 {
		currentPos = int(currentEBR.Part_next)
		if err := currentEBR.Deserialize(cmd.path, currentPos); err != nil {
			return "", fmt.Errorf("ERROR: error leyendo siguiente EBR: %v", err)
		}
	}

	// Calcular posición del nuevo EBR
	nextEBRPos := int(currentEBR.Part_start) + int(currentEBR.Part_size)

	// Verificar que no exceda la partición extendida
	if nextEBRPos+ebrSize+sizeBytes > extStart+extSize {
		return "", errors.New("ERROR: no hay suficiente espacio en la partición extendida")
	}

	// Actualizar el EBR actual para que apunte al nuevo
	currentEBR.Part_next = int32(nextEBRPos)

	// Escribir el EBR actualizado
	if err := currentEBR.Serialize(cmd.path, currentPos); err != nil {
		return "", fmt.Errorf("ERROR: error actualizando EBR: %v", err)
	}

	// Crear el nuevo EBR
	newEBR := structures.EBR{
		Part_mount: [1]byte{'1'},
		Part_fit:   [1]byte{cmd.fit[0]},
		Part_start: int32(nextEBRPos + ebrSize),
		Part_size:  int32(sizeBytes),
		Part_next:  -1,
	}
	copy(newEBR.Part_name[:], cmd.name)

	// Escribir el nuevo EBR
	if err := newEBR.Serialize(cmd.path, nextEBRPos); err != nil {
		return "", fmt.Errorf("ERROR: error escribiendo nuevo EBR: %v", err)
	}

	msg := fmt.Sprintf("FDISK: Partición lógica '%s' creada exitosamente\n"+
		"-> Tamaño: %d bytes\n"+
		"-> Inicio: %d",
		cmd.name, sizeBytes, nextEBRPos+ebrSize)

	return msg, nil
}

// findAvailableSpace encuentra espacio disponible según el ajuste
func findAvailableSpace(mbr *structures.MBR, sizeBytes int, fit string) (int, error) {
	mbrSize := binary.Size(structures.MBR{})
	diskSize := int(mbr.Mbr_size)

	// Crear lista de espacios ocupados
	type space struct {
		start int
		end   int
	}

	var occupied []space
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start != -1 && mbr.Mbr_partitions[i].Part_size > 0 {
			occupied = append(occupied, space{
				start: int(mbr.Mbr_partitions[i].Part_start),
				end:   int(mbr.Mbr_partitions[i].Part_start) + int(mbr.Mbr_partitions[i].Part_size),
			})
		}
	}

	// Ordenar espacios ocupados por posición de inicio
	for i := 0; i < len(occupied); i++ {
		for j := i + 1; j < len(occupied); j++ {
			if occupied[j].start < occupied[i].start {
				occupied[i], occupied[j] = occupied[j], occupied[i]
			}
		}
	}

	// Encontrar espacios libres
	var freeSpaces []space
	currentPos := mbrSize

	for _, occ := range occupied {
		if occ.start > currentPos {
			freeSpaces = append(freeSpaces, space{
				start: currentPos,
				end:   occ.start,
			})
		}
		currentPos = occ.end
	}

	// Agregar espacio después de la última partición
	if currentPos < diskSize {
		freeSpaces = append(freeSpaces, space{
			start: currentPos,
			end:   diskSize,
		})
	}

	// Aplicar el ajuste
	var selectedSpace *space

	switch fit {
	case "FF": // First Fit
		for i := range freeSpaces {
			if freeSpaces[i].end-freeSpaces[i].start >= sizeBytes {
				selectedSpace = &freeSpaces[i]
				break
			}
		}

	case "BF": // Best Fit
		minWaste := diskSize + 1
		for i := range freeSpaces {
			available := freeSpaces[i].end - freeSpaces[i].start
			if available >= sizeBytes {
				waste := available - sizeBytes
				if waste < minWaste {
					minWaste = waste
					selectedSpace = &freeSpaces[i]
				}
			}
		}

	case "WF": // Worst Fit
		maxSpace := -1
		for i := range freeSpaces {
			available := freeSpaces[i].end - freeSpaces[i].start
			if available >= sizeBytes && available > maxSpace {
				maxSpace = available
				selectedSpace = &freeSpaces[i]
			}
		}
	}

	if selectedSpace == nil {
		return -1, errors.New("ERROR: no hay suficiente espacio contiguo disponible")
	}

	return selectedSpace.start, nil
}

// deletePartition elimina una partición
func deletePartition(cmd *FDISK) (string, error) {
	var mbr structures.MBR
	if err := mbr.Deserialize(cmd.path); err != nil {
		return "", fmt.Errorf("ERROR: error leyendo MBR: %v", err)
	}

	// Buscar la partición a eliminar
	partIndex := -1
	for i := 0; i < 4; i++ {
		partName := strings.TrimRight(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00")
		if partName == cmd.name && mbr.Mbr_partitions[i].Part_start != -1 {
			partIndex = i
			break
		}
	}

	if partIndex == -1 {
		return "", fmt.Errorf("ERROR: no existe la partición '%s'", cmd.name)
	}

	partition := &mbr.Mbr_partitions[partIndex]

	// Si es extendida y delete=full, limpiar todo
	if partition.Part_type[0] == 'E' && cmd.delete == "full" {
		file, err := os.OpenFile(cmd.path, os.O_WRONLY, 0644)
		if err != nil {
			return "", fmt.Errorf("ERROR: error abriendo el disco: %v", err)
		}
		defer file.Close()

		if _, err := file.Seek(int64(partition.Part_start), 0); err != nil {
			return "", fmt.Errorf("ERROR: error posicionando en el disco: %v", err)
		}

		zeros := make([]byte, partition.Part_size)
		if _, err := file.Write(zeros); err != nil {
			return "", fmt.Errorf("ERROR: error limpiando la partición: %v", err)
		}
	} else if cmd.delete == "full" {
		// Llenar con \0 la partición
		file, err := os.OpenFile(cmd.path, os.O_WRONLY, 0644)
		if err != nil {
			return "", fmt.Errorf("ERROR: error abriendo el disco: %v", err)
		}
		defer file.Close()

		if _, err := file.Seek(int64(partition.Part_start), 0); err != nil {
			return "", fmt.Errorf("ERROR: error posicionando en el disco: %v", err)
		}

		zeros := make([]byte, partition.Part_size)
		if _, err := file.Write(zeros); err != nil {
			return "", fmt.Errorf("ERROR: error limpiando la partición: %v", err)
		}
	}

	// Marcar como vacía en el MBR
	mbr.Mbr_partitions[partIndex].Part_status = [1]byte{'0'}
	mbr.Mbr_partitions[partIndex].Part_type = [1]byte{'0'}
	mbr.Mbr_partitions[partIndex].Part_fit = [1]byte{'0'}
	mbr.Mbr_partitions[partIndex].Part_start = -1
	mbr.Mbr_partitions[partIndex].Part_size = 0
	mbr.Mbr_partitions[partIndex].Part_correlative = -1
	for j := range mbr.Mbr_partitions[partIndex].Part_name {
		mbr.Mbr_partitions[partIndex].Part_name[j] = 0
	}
	for j := range mbr.Mbr_partitions[partIndex].Part_id {
		mbr.Mbr_partitions[partIndex].Part_id[j] = 0
	}

	// Serializar el MBR
	if err := mbr.Serialize(cmd.path); err != nil {
		return "", fmt.Errorf("ERROR: error escribiendo MBR: %v", err)
	}

	return fmt.Sprintf("FDISK: Partición '%s' eliminada correctamente (%s)", cmd.name, cmd.delete), nil
}

// modifyPartitionSize modifica el tamaño de una partición
func modifyPartitionSize(cmd *FDISK) (string, error) {
	if cmd.unit == "" {
		cmd.unit = "K"
	}

	// Convertir el valor de add a bytes
	addBytes, err := utils.ConvertToBytes(cmd.add, cmd.unit)
	if err != nil {
		return "", fmt.Errorf("ERROR: error convirtiendo tamaño: %v", err)
	}

	var mbr structures.MBR
	if err := mbr.Deserialize(cmd.path); err != nil {
		return "", fmt.Errorf("ERROR: error leyendo MBR: %v", err)
	}

	// Buscar la partición
	partIndex := -1
	for i := 0; i < 4; i++ {
		partName := strings.TrimRight(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00")
		if partName == cmd.name && mbr.Mbr_partitions[i].Part_start != -1 {
			partIndex = i
			break
		}
	}

	if partIndex == -1 {
		return "", fmt.Errorf("ERROR: no existe la partición '%s'", cmd.name)
	}

	partition := &mbr.Mbr_partitions[partIndex]
	newSize := int(partition.Part_size) + addBytes

	// Verificar que el nuevo tamaño sea positivo
	if newSize <= 0 {
		return "", errors.New("ERROR: el tamaño resultante debe ser positivo")
	}

	// Si se está agregando espacio, verificar que haya espacio disponible después
	if addBytes > 0 {
		partEnd := int(partition.Part_start) + int(partition.Part_size)
		nextPartStart := int(mbr.Mbr_size)

		// Buscar la siguiente partición
		for i := 0; i < 4; i++ {
			if i != partIndex && mbr.Mbr_partitions[i].Part_start != -1 {
				if int(mbr.Mbr_partitions[i].Part_start) > partEnd &&
					int(mbr.Mbr_partitions[i].Part_start) < nextPartStart {
					nextPartStart = int(mbr.Mbr_partitions[i].Part_start)
				}
			}
		}

		availableSpace := nextPartStart - partEnd
		if addBytes > availableSpace {
			return "", errors.New("ERROR: no hay suficiente espacio contiguo disponible")
		}
	}

	// Actualizar el tamaño
	partition.Part_size = int32(newSize)

	// Serializar el MBR
	if err := mbr.Serialize(cmd.path); err != nil {
		return "", fmt.Errorf("ERROR: error escribiendo MBR: %v", err)
	}

	operation := "agregados"
	if addBytes < 0 {
		operation = "removidos"
	}

	return fmt.Sprintf("FDISK: Partición '%s' modificada correctamente (%d bytes %s)",
		cmd.name, utils.Abs(addBytes), operation), nil
}
