package commands

import (
	"backend/stores"
	"backend/structures"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// MKUSR estructura que representa el comando mkusr con sus parámetros
type MKUSR struct {
	user string // Nombre del usuario a crear
	pass string // Contraseña del usuario
	grp  string // Grupo al que pertenece el usuario
}

/*
	Ejemplos de uso:
	mkusr -user="usuario1" -pass=password -grp=root
	mkusr -user="user1" -pass=abc -grp=usuarios

	Solo puede ser ejecutado por el usuario root
	El grupo debe existir y no estar eliminado
	El usuario no debe existir previamente
*/

// ParseMkusr analiza los tokens del comando mkusr
func ParseMkusr(tokens []string) (string, error) {
	cmd := &MKUSR{}

	// Procesar cada token
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		lowerToken := strings.ToLower(token)

		if strings.HasPrefix(lowerToken, "-user=") {
			value := token[len("-user="):]
			// Quitar comillas si existen
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = strings.Trim(value, "\"")
			}
			cmd.user = value
		} else if strings.HasPrefix(lowerToken, "-pass=") {
			value := token[len("-pass="):]
			// Quitar comillas si existen
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = strings.Trim(value, "\"")
			}
			cmd.pass = value
		} else if strings.HasPrefix(lowerToken, "-grp=") {
			value := token[len("-grp="):]
			// Quitar comillas si existen
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = strings.Trim(value, "\"")
			}
			cmd.grp = value
		} else if token != "" && token != "mkusr" {
			return "", fmt.Errorf("MKUSR ERROR: parámetro no reconocido '%s'", token)
		}
	}

	// Validar parámetros obligatorios
	if cmd.user == "" {
		return "", errors.New("MKUSR ERROR: el parámetro -user es obligatorio")
	}
	if cmd.pass == "" {
		return "", errors.New("MKUSR ERROR: el parámetro -pass es obligatorio")
	}
	if cmd.grp == "" {
		return "", errors.New("MKUSR ERROR: el parámetro -grp es obligatorio")
	}

	// Validar longitud máxima (10 caracteres según el enunciado)
	if len(cmd.user) > 10 {
		return "", errors.New("MKUSR ERROR: el nombre de usuario no puede exceder 10 caracteres")
	}
	if len(cmd.pass) > 10 {
		return "", errors.New("MKUSR ERROR: la contraseña no puede exceder 10 caracteres")
	}
	if len(cmd.grp) > 10 {
		return "", errors.New("MKUSR ERROR: el nombre del grupo no puede exceder 10 caracteres")
	}

	// Ejecutar el comando
	err := commandMkusr(cmd)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("MKUSR: Usuario '%s' creado exitosamente en el grupo '%s'", cmd.user, cmd.grp), nil
}

// commandMkusr ejecuta la lógica del comando mkusr
func commandMkusr(cmd *MKUSR) error {
	// 1. Verificar que hay una sesión activa
	if !stores.Auth.IsAuthenticated() {
		return errors.New("MKUSR ERROR: No hay una sesión activa. Use el comando LOGIN primero")
	}

	// 2. Verificar que el usuario actual es root
	currentUser, _, _ := stores.Auth.GetCurrentUser()
	if currentUser != "root" {
		return errors.New("MKUSR ERROR: Solo el usuario root puede crear usuarios")
	}

	// 3. Obtener la partición montada y el superbloque
	partitionID := stores.Auth.GetPartitionID()
	sb, partition, diskPath, err := stores.GetMountedPartitionSuperblock(partitionID)
	if err != nil {
		return fmt.Errorf("MKUSR ERROR: error al obtener la partición montada: %w", err)
	}

	// 4. Leer el contenido actual de users.txt
	usersContent, err := readUsersFileMkusr(sb, diskPath)
	if err != nil {
		return fmt.Errorf("MKUSR ERROR: error al leer users.txt: %w", err)
	}

	// 5. Parsear las líneas existentes
	lines := strings.Split(usersContent, "\n")
	var validLines []string
	nextUID := int32(1)
	groupExists := false
	userExists := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		validLines = append(validLines, line)
		parts := strings.Split(line, ",")

		if len(parts) >= 3 {
			// Limpiar espacios
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}

			var id int32
			fmt.Sscanf(parts[0], "%d", &id)

			// Actualizar el siguiente UID disponible
			if id >= nextUID {
				nextUID = id + 1
			}

			// Verificar si el grupo existe y no está eliminado
			if parts[1] == "G" && parts[0] != "0" && parts[2] == cmd.grp {
				groupExists = true
			}

			// Verificar si el usuario ya existe (incluso si está eliminado)
			if len(parts) >= 4 && parts[1] == "U" && parts[3] == cmd.user {
				if parts[0] != "0" {
					userExists = true
				}
			}
		}
	}

	// 6. Validar que el grupo existe
	if !groupExists {
		return fmt.Errorf("MKUSR ERROR: el grupo '%s' no existe o está eliminado", cmd.grp)
	}

	// 7. Validar que el usuario no existe
	if userExists {
		return fmt.Errorf("MKUSR ERROR: el usuario '%s' ya existe", cmd.user)
	}

	// 8. Crear la nueva línea de usuario
	// Formato: UID, Tipo, Grupo, Usuario, Contraseña
	newUserLine := fmt.Sprintf("%d,U,%s,%s,%s", nextUID, cmd.grp, cmd.user, cmd.pass)
	validLines = append(validLines, newUserLine)

	// 9. Reconstruir el contenido completo
	newContent := strings.Join(validLines, "\n") + "\n"

	fmt.Printf("DEBUG MKUSR -> Nuevo contenido de users.txt:\n%s\n", newContent)

	// 10. Escribir el nuevo contenido en users.txt
	err = writeUsersFileMkusr(sb, partition, diskPath, newContent)
	if err != nil {
		return fmt.Errorf("MKUSR ERROR: error al escribir users.txt: %w", err)
	}

	return nil
}

// readUsersFileMkusr lee el contenido completo del archivo users.txt (inodo 1)
func readUsersFileMkusr(sb *structures.SuperBlock, path string) (string, error) {
	// Deserializar el inodo 1 (users.txt)
	inode := &structures.Inode{}
	inodeOffset := int64(sb.S_inode_start + (1 * sb.S_inode_size))
	err := inode.Deserialize(path, inodeOffset)
	if err != nil {
		return "", fmt.Errorf("error al deserializar inodo users.txt: %w", err)
	}

	// Verificar que sea un archivo
	if inode.I_type[0] != '1' {
		return "", errors.New("el inodo 1 no es un archivo")
	}

	// Leer el contenido de todos los bloques
	var content strings.Builder

	for i := 0; i < 12; i++ {
		blockIndex := inode.I_block[i]

		if blockIndex == -1 {
			break
		}

		block := &structures.FileBlock{}
		blockOffset := int64(sb.S_block_start + (blockIndex * sb.S_block_size))
		err := block.Deserialize(path, blockOffset)
		if err != nil {
			return "", fmt.Errorf("error al deserializar bloque %d: %w", blockIndex, err)
		}

		blockContent := string(block.B_content[:])
		blockContent = strings.ReplaceAll(blockContent, "\x00", "")
		content.WriteString(blockContent)
	}

	return content.String(), nil
}

// writeUsersFileMkusr escribe el contenido actualizado en users.txt
func writeUsersFileMkusr(sb *structures.SuperBlock, partition *structures.Partition, path string, content string) error {
	// Deserializar el inodo 1 (users.txt)
	inode := &structures.Inode{}
	inodeOffset := int64(sb.S_inode_start + (1 * sb.S_inode_size))
	err := inode.Deserialize(path, inodeOffset)
	if err != nil {
		return fmt.Errorf("error al deserializar inodo users.txt: %w", err)
	}

	// Verificar que sea un archivo
	if inode.I_type[0] != '1' {
		return errors.New("el inodo 1 no es un archivo")
	}

	// Actualizar el tamaño del archivo
	inode.I_size = int32(len(content))

	// Actualizar timestamps
	inode.I_mtime = float32(time.Now().Unix()) // Última modificación
	inode.I_atime = float32(time.Now().Unix()) // Último acceso

	// Calcular cuántos bloques necesitamos (cada bloque tiene 64 bytes)
	contentBytes := []byte(content)
	blocksNeeded := (len(contentBytes) + 63) / 64 // Redondear hacia arriba

	if blocksNeeded > 12 {
		return errors.New("el contenido de users.txt excede la capacidad de 12 bloques directos")
	}

	fmt.Printf("DEBUG writeUsersFileMkusr -> Bloques necesarios: %d\n", blocksNeeded)
	fmt.Printf("DEBUG writeUsersFileMkusr -> Contenido length: %d bytes\n", len(contentBytes))

	// Abrir el archivo para escritura
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir archivo: %w", err)
	}
	defer file.Close()

	// Escribir el contenido en los bloques
	for i := 0; i < blocksNeeded; i++ {
		blockIndex := inode.I_block[i]

		// Si el bloque no existe, necesitamos asignar uno nuevo
		if blockIndex == -1 {
			// Buscar el siguiente bloque libre en el bitmap
			blockIndex, err = findFreeBlock(sb, file)
			if err != nil {
				return fmt.Errorf("error al buscar bloque libre: %w", err)
			}

			// Asignar el bloque al inodo
			inode.I_block[i] = blockIndex

			// Marcar el bloque como usado en el bitmap
			err = updateBitmapBlockMkusr(sb, file, blockIndex, true)
			if err != nil {
				return fmt.Errorf("error al actualizar bitmap: %w", err)
			}

			// Actualizar contador de bloques libres
			sb.S_free_blocks_count--
		}

		// Preparar el bloque
		block := &structures.FileBlock{}

		// Calcular el rango de bytes para este bloque
		startByte := i * 64
		endByte := startByte + 64
		if endByte > len(contentBytes) {
			endByte = len(contentBytes)
		}

		// Copiar el contenido al bloque (rellenar con zeros el resto)
		for j := 0; j < 64; j++ {
			if startByte+j < len(contentBytes) {
				block.B_content[j] = contentBytes[startByte+j]
			} else {
				block.B_content[j] = 0
			}
		}

		// Escribir el bloque en el disco
		blockOffset := int64(sb.S_block_start + (blockIndex * sb.S_block_size))
		fmt.Printf("DEBUG writeUsersFileMkusr -> Escribiendo bloque %d en offset %d\n", blockIndex, blockOffset)

		err := block.Serialize(path, blockOffset)
		if err != nil {
			return fmt.Errorf("error al escribir bloque %d: %w", blockIndex, err)
		}
	}

	// Limpiar bloques no utilizados (si el archivo se hizo más pequeño)
	for i := blocksNeeded; i < 12; i++ {
		if inode.I_block[i] != -1 {
			// Marcar el bloque como libre
			err = updateBitmapBlockMkusr(sb, file, inode.I_block[i], false)
			if err != nil {
				return fmt.Errorf("error al liberar bloque: %w", err)
			}

			// Limpiar el bloque
			block := &structures.FileBlock{}
			blockOffset := int64(sb.S_block_start + (inode.I_block[i] * sb.S_block_size))
			err := block.Serialize(path, blockOffset)
			if err != nil {
				return fmt.Errorf("error al limpiar bloque %d: %w", i, err)
			}

			// Remover referencia del inodo
			inode.I_block[i] = -1
			sb.S_free_blocks_count++
		}
	}

	// Actualizar el inodo en el disco
	err = inode.Serialize(path, inodeOffset)
	if err != nil {
		return fmt.Errorf("error al escribir inodo users.txt: %w", err)
	}

	// Actualizar el superbloque en el disco
	sbOffset := int64(partition.Part_start)
	err = sb.Serialize(path, sbOffset)
	if err != nil {
		return fmt.Errorf("error al actualizar superbloque: %w", err)
	}

	fmt.Println("DEBUG writeUsersFileMkusr -> Escritura completada exitosamente")
	return nil
}

// findFreeBlock busca el primer bloque libre en el bitmap
func findFreeBlock(sb *structures.SuperBlock, file *os.File) (int32, error) {
	// Leer el bitmap de bloques
	bitmapSize := sb.S_blocks_count
	bitmap := make([]byte, bitmapSize)

	_, err := file.Seek(int64(sb.S_bm_block_start), 0)
	if err != nil {
		return -1, err
	}

	_, err = file.Read(bitmap)
	if err != nil {
		return -1, err
	}

	// Buscar el primer bit libre (0)
	for i := int32(0); i < bitmapSize; i++ {
		if bitmap[i] == 0 {
			return i, nil
		}
	}

	return -1, errors.New("no hay bloques libres disponibles")
}

// updateBitmapBlockMkusr actualiza el bitmap de bloques
func updateBitmapBlockMkusr(sb *structures.SuperBlock, file *os.File, blockIndex int32, used bool) error {
	// Posicionarse en el bitmap de bloques
	bitmapOffset := int64(sb.S_bm_block_start + blockIndex)
	_, err := file.Seek(bitmapOffset, 0)
	if err != nil {
		return err
	}

	// Escribir el bit
	var bit byte
	if used {
		bit = 1
	} else {
		bit = 0
	}

	_, err = file.Write([]byte{bit})
	if err != nil {
		return err
	}

	return nil
}
