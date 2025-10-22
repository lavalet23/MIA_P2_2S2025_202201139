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

// RMGRP estructura que representa el comando rmgrp con sus parámetros
type RMGRP struct {
	name string // Nombre del grupo a eliminar
}

/*
	Ejemplos de uso:
	rmgrp -name=mail

	Solo puede ser ejecutado por el usuario root
	El grupo debe existir y no estar ya eliminado
	Elimina lógicamente (cambia el ID a 0)
*/

// ParseRmgrp analiza los tokens del comando rmgrp
func ParseRmgrp(tokens []string) (string, error) {
	cmd := &RMGRP{}

	// Procesar cada token
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		lowerToken := strings.ToLower(token)

		if strings.HasPrefix(lowerToken, "-name=") {
			value := token[len("-name="):]
			// Quitar comillas si existen
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = strings.Trim(value, "\"")
			}
			cmd.name = value
		} else if token != "" && token != "rmgrp" {
			return "", fmt.Errorf("RMGRP ERROR: parámetro no reconocido '%s'", token)
		}
	}

	// Validar parámetro obligatorio
	if cmd.name == "" {
		return "", errors.New("RMGRP ERROR: el parámetro -name es obligatorio")
	}

	// Ejecutar el comando
	err := commandRmgrp(cmd)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("RMGRP: Grupo '%s' eliminado exitosamente", cmd.name), nil
}

// commandRmgrp ejecuta la lógica del comando rmgrp
func commandRmgrp(cmd *RMGRP) error {
	// 1. Verificar que hay una sesión activa
	if !stores.Auth.IsAuthenticated() {
		return errors.New("RMGRP ERROR: No hay una sesión activa. Use el comando LOGIN primero")
	}

	// 2. Verificar que el usuario actual es root
	currentUser, _, _ := stores.Auth.GetCurrentUser()
	if currentUser != "root" {
		return errors.New("RMGRP ERROR: Solo el usuario root puede eliminar grupos")
	}

	// 3. Obtener la partición montada y el superbloque
	partitionID := stores.Auth.GetPartitionID()
	sb, partition, diskPath, err := stores.GetMountedPartitionSuperblock(partitionID)
	if err != nil {
		return fmt.Errorf("RMGRP ERROR: error al obtener la partición montada: %w", err)
	}

	// 4. Leer el contenido actual de users.txt
	usersContent, err := readUsersFileRmgrp(sb, diskPath)
	if err != nil {
		return fmt.Errorf("RMGRP ERROR: error al leer users.txt: %w", err)
	}

	// 5. Parsear las líneas existentes y marcar el grupo como eliminado
	lines := strings.Split(usersContent, "\n")
	var validLines []string
	groupFound := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")

		if len(parts) >= 3 {
			// Limpiar espacios
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}

			// Si es el grupo a eliminar y no está ya eliminado
			if parts[1] == "G" && parts[2] == cmd.name {
				if parts[0] == "0" {
					return fmt.Errorf("RMGRP ERROR: el grupo '%s' ya está eliminado", cmd.name)
				}

				// Marcar como eliminado cambiando el ID a 0
				parts[0] = "0"
				groupFound = true
			}
		}

		// Reconstruir la línea
		validLines = append(validLines, strings.Join(parts, ","))
	}

	// 6. Validar que el grupo existe
	if !groupFound {
		return fmt.Errorf("RMGRP ERROR: el grupo '%s' no existe", cmd.name)
	}

	// 7. Reconstruir el contenido completo
	newContent := strings.Join(validLines, "\n") + "\n"

	fmt.Printf("DEBUG RMGRP -> Nuevo contenido de users.txt:\n%s\n", newContent)

	// 8. Escribir el nuevo contenido en users.txt
	err = writeUsersFileRmgrp(sb, partition, diskPath, newContent)
	if err != nil {
		return fmt.Errorf("RMGRP ERROR: error al escribir users.txt: %w", err)
	}

	return nil
}

// readUsersFileRmgrp lee el contenido completo del archivo users.txt (inodo 1)
func readUsersFileRmgrp(sb *structures.SuperBlock, path string) (string, error) {
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

// writeUsersFileRmgrp escribe el contenido actualizado en users.txt
func writeUsersFileRmgrp(sb *structures.SuperBlock, partition *structures.Partition, path string, content string) error {
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
	inode.I_mtime = float32(time.Now().Unix())
	inode.I_atime = float32(time.Now().Unix())

	// Calcular cuántos bloques necesitamos
	contentBytes := []byte(content)
	blocksNeeded := (len(contentBytes) + 63) / 64

	if blocksNeeded > 12 {
		return errors.New("el contenido de users.txt excede la capacidad de 12 bloques directos")
	}

	// Abrir el archivo para escritura
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir archivo: %w", err)
	}
	defer file.Close()

	// Escribir el contenido en los bloques
	for i := 0; i < blocksNeeded; i++ {
		blockIndex := inode.I_block[i]

		if blockIndex == -1 {
			blockIndex, err = findFreeBlockRmgrp(sb, file)
			if err != nil {
				return fmt.Errorf("error al buscar bloque libre: %w", err)
			}

			inode.I_block[i] = blockIndex

			err = updateBitmapBlockRmgrp(sb, file, blockIndex, true)
			if err != nil {
				return fmt.Errorf("error al actualizar bitmap: %w", err)
			}

			sb.S_free_blocks_count--
		}

		block := &structures.FileBlock{}

		startByte := i * 64
		endByte := startByte + 64
		if endByte > len(contentBytes) {
			endByte = len(contentBytes)
		}

		for j := 0; j < 64; j++ {
			if startByte+j < len(contentBytes) {
				block.B_content[j] = contentBytes[startByte+j]
			} else {
				block.B_content[j] = 0
			}
		}

		blockOffset := int64(sb.S_block_start + (blockIndex * sb.S_block_size))
		err := block.Serialize(path, blockOffset)
		if err != nil {
			return fmt.Errorf("error al escribir bloque %d: %w", blockIndex, err)
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

	return nil
}

// findFreeBlockRmgrp busca el primer bloque libre en el bitmap
func findFreeBlockRmgrp(sb *structures.SuperBlock, file *os.File) (int32, error) {
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

	for i := int32(0); i < bitmapSize; i++ {
		if bitmap[i] == 0 {
			return i, nil
		}
	}

	return -1, errors.New("no hay bloques libres disponibles")
}

// updateBitmapBlockRmgrp actualiza el bitmap de bloques
func updateBitmapBlockRmgrp(sb *structures.SuperBlock, file *os.File, blockIndex int32, used bool) error {
	bitmapOffset := int64(sb.S_bm_block_start + blockIndex)
	_, err := file.Seek(bitmapOffset, 0)
	if err != nil {
		return err
	}

	var bit byte
	if used {
		bit = 1
	} else {
		bit = 0
	}

	_, err = file.Write([]byte{bit})
	return err
}
