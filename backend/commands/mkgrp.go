package commands

import (
	"backend/stores"
	"backend/structures"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MKGRP estructura que representa el comando mkgrp con sus parámetros
type MKGRP struct {
	name string // Nombre del grupo
}

/*
	mkgrp -name=usuarios
	Ejemplo archivo users.txt:
	1,G,root
	1,U,root,root,123
	2,G,usuarios
*/

func ParseMkgrp(tokens []string) (string, error) {
	cmd := &MKGRP{} // Crea una nueva instancia de MKGRP

	// Unir tokens en una sola cadena y luego dividir por espacios, respetando las comillas
	args := strings.Join(tokens, " ")

	// Expresión regular para encontrar los parámetros del comando mkgrp
	re := regexp.MustCompile(`-name=[^\s]+`)

	// Encuentra todas las coincidencias de la expresión regular en la cadena de argumentos
	matches := re.FindAllString(args, -1)

	// Verificar que todos los tokens fueron reconocidos por la expresión regular
	if len(matches) != len(tokens) {
		// Identificar el parámetro inválido
		for _, token := range tokens {
			if !re.MatchString(token) {
				return "", fmt.Errorf("parámetro inválido: %s", token)
			}
		}
	}

	// Itera sobre cada coincidencia encontrada
	for _, match := range matches {
		// Divide cada parte en clave y valor usando "=" como delimitador
		kv := strings.SplitN(match, "=", 2)
		if len(kv) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", match)
		}
		key, value := strings.ToLower(kv[0]), kv[1]

		// Remove quotes from value if present
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		// Switch para manejar diferentes parámetros
		switch key {
		case "-name":
			if value == "" {
				return "", errors.New("el nombre del grupo no puede estar vacío")
			}
			if len(value) > 10 {
				return "", errors.New("el nombre del grupo no puede exceder 10 caracteres")
			}
			cmd.name = value
		default:
			// Si el parámetro no es reconocido, devuelve un error
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	// Verifica que el parámetro -name haya sido proporcionado
	if cmd.name == "" {
		return "", errors.New("faltan parámetros requeridos: -name")
	}

	// Ejecutar el comando mkgrp
	err := commandMkgrp(cmd)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("MKGRP: Grupo '%s' creado exitosamente", cmd.name), nil
}

func commandMkgrp(mkgrp *MKGRP) error {
	// 1. Verificar que hay una sesión activa
	if !stores.Auth.IsAuthenticated() {
		return errors.New("ERROR: No hay una sesión activa. Use el comando LOGIN primero")
	}

	// 2. Verificar que el usuario es root
	currentUser, _, _ := stores.Auth.GetCurrentUser()
	if currentUser != "root" {
		return errors.New("ERROR: Solo el usuario root puede crear grupos")
	}

	// 3. Obtener la partición montada y el superbloque
	partitionID := stores.Auth.GetPartitionID()
	partitionSuperblock, partition, partitionPath, err := stores.GetMountedPartitionSuperblock(partitionID)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	// 4. Leer el archivo users.txt
	usersContent, err := readUsersFile(partitionSuperblock, partitionPath)
	if err != nil {
		return fmt.Errorf("error al leer el archivo users.txt: %w", err)
	}

	// 5. Verificar que el grupo no existe (case-sensitive)
	if groupExists(usersContent, mkgrp.name) {
		return fmt.Errorf("ERROR: El grupo '%s' ya existe", mkgrp.name)
	}

	// 6. Obtener el siguiente GID
	nextGID := getNextGID(usersContent)

	// 7. Crear el nuevo registro del grupo
	newGroupLine := fmt.Sprintf("%d,G,%s\n", nextGID, mkgrp.name)

	// 8. Agregar el nuevo grupo al contenido
	newContent := usersContent + newGroupLine

	// 9. Escribir el contenido actualizado en users.txt
	err = writeUsersFile(partitionSuperblock, partitionPath, partition.Part_start, newContent)
	if err != nil {
		return fmt.Errorf("error al actualizar el archivo users.txt: %w", err)
	}

	return nil
}

// readUsersFile lee el contenido del archivo users.txt desde el inodo 1
func readUsersFile(sb *structures.SuperBlock, path string) (string, error) {
	// Ir al inodo 1 (users.txt)
	inode := &structures.Inode{}

	// Deserializar el inodo 1
	err := inode.Deserialize(path, int64(sb.S_inode_start+(1*sb.S_inode_size)))
	if err != nil {
		return "", err
	}

	// Verificar que sea un archivo
	if inode.I_type[0] != '1' {
		return "", errors.New("el inodo 1 no es un archivo")
	}

	// Leer el contenido de todos los bloques del archivo
	var content strings.Builder

	// Iterar sobre los bloques directos (primeros 12)
	for i := 0; i < 12; i++ {
		blockIndex := inode.I_block[i]

		// Si el bloque no existe, terminar
		if blockIndex == -1 {
			break
		}

		// Deserializar el bloque de archivo
		block := &structures.FileBlock{}
		err := block.Deserialize(path, int64(sb.S_block_start+(blockIndex*sb.S_block_size)))
		if err != nil {
			return "", err
		}

		// Agregar el contenido del bloque
		blockContent := string(block.B_content[:])
		// Eliminar padding null
		blockContent = strings.ReplaceAll(blockContent, "\x00", "")
		content.WriteString(blockContent)
	}

	return content.String(), nil
}

// writeUsersFile escribe el contenido actualizado en users.txt
func writeUsersFile(sb *structures.SuperBlock, path string, partitionStart int32, content string) error {
	// Abrir el archivo del disco
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Ir al inodo 1 (users.txt)
	inode := &structures.Inode{}
	inodeOffset := int64(partitionStart) + int64(sb.S_inode_start+(1*sb.S_inode_size))

	// Deserializar el inodo 1
	err = inode.Deserialize(path, inodeOffset)
	if err != nil {
		return err
	}

	// Convertir el contenido a bytes
	contentBytes := []byte(content)
	blockSize := 64
	blocksNeeded := (len(contentBytes) + blockSize - 1) / blockSize

	// Verificar si necesitamos más bloques de los disponibles
	if blocksNeeded > 12 {
		return fmt.Errorf("el contenido es demasiado grande, se necesitan %d bloques pero solo hay 12 disponibles", blocksNeeded)
	}

	// Escribir el contenido en los bloques
	for i := 0; i < blocksNeeded; i++ {
		start := i * blockSize
		end := start + blockSize
		if end > len(contentBytes) {
			end = len(contentBytes)
		}

		// Crear el bloque de archivo
		fileBlock := &structures.FileBlock{}
		copy(fileBlock.B_content[:], contentBytes[start:end])

		// Si no hay bloque asignado, asignar uno nuevo
		if inode.I_block[i] == -1 {
			// Asignar un nuevo bloque
			newBlockIndex := sb.S_blocks_count
			inode.I_block[i] = newBlockIndex

			// Actualizar el bitmap de bloques
			bitmapPos := int64(partitionStart) + int64(sb.S_bm_block_start) + int64(newBlockIndex)
			file.Seek(bitmapPos, 0)
			file.Write([]byte{1})

			// Actualizar contadores en el superbloque
			sb.S_blocks_count++
			sb.S_free_blocks_count--
			sb.S_first_blo += sb.S_block_size
		}

		// Escribir el bloque
		blockOffset := int64(partitionStart) + int64(sb.S_block_start+(inode.I_block[i]*sb.S_block_size))
		err = fileBlock.Serialize(path, blockOffset)
		if err != nil {
			return err
		}
	}

	// Limpiar bloques no utilizados (si el nuevo contenido es más pequeño)
	for i := blocksNeeded; i < 12; i++ {
		if inode.I_block[i] != -1 {
			// Marcar bloque como libre en el bitmap
			bitmapPos := int64(partitionStart) + int64(sb.S_bm_block_start) + int64(inode.I_block[i])
			file.Seek(bitmapPos, 0)
			file.Write([]byte{0})

			// Limpiar el apuntador
			inode.I_block[i] = -1

			// Actualizar contadores
			sb.S_free_blocks_count++
		}
	}

	// Actualizar el tamaño del archivo en el inodo
	inode.I_size = int32(len(contentBytes))

	// Actualizar tiempo de modificación
	inode.I_mtime = float32(time.Now().Unix())

	// Serializar el inodo actualizado
	err = inode.Serialize(path, inodeOffset)
	if err != nil {
		return err
	}

	// Serializar el superbloque actualizado
	sbOffset := int64(partitionStart)
	err = sb.Serialize(path, sbOffset)
	if err != nil {
		return err
	}

	return nil
}

// groupExists verifica si un grupo ya existe (case-sensitive)
func groupExists(usersContent, groupName string) bool {
	lines := strings.Split(usersContent, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 3 {
			// Verificar si es un grupo (tipo G) y si no está eliminado (GID != 0)
			gid := strings.TrimSpace(parts[0])
			tipo := strings.TrimSpace(parts[1])
			nombre := strings.TrimSpace(parts[2])

			// Case-sensitive: comparar exactamente
			if tipo == "G" && gid != "0" && nombre == groupName {
				return true
			}
		}
	}

	return false
}

// getNextGID obtiene el siguiente GID disponible
func getNextGID(usersContent string) int {
	maxGID := 0
	lines := strings.Split(usersContent, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			tipo := strings.TrimSpace(parts[1])
			if tipo == "G" {
				gid, err := strconv.Atoi(strings.TrimSpace(parts[0]))
				if err == nil && gid > maxGID {
					maxGID = gid
				}
			}
		}
	}

	return maxGID + 1
}
