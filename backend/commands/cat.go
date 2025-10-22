package commands

import (
	"backend/stores"
	"backend/structures"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// CAT estructura que representa el comando cat con sus parámetros
type CAT struct {
	files []string // Lista de archivos a leer
}

// ParseCat analiza los tokens del comando cat
func ParseCat(tokens []string) (string, error) {
	cmd := &CAT{
		files: make([]string, 0),
	}

	// Unir tokens en una sola cadena
	args := strings.Join(tokens, " ")

	// Expresión regular para encontrar parámetros -fileN
	re := regexp.MustCompile(`-file\d+=[^\s]+`)

	// Encuentra todas las coincidencias
	matches := re.FindAllString(args, -1)

	if len(matches) == 0 {
		return "", errors.New("debe proporcionar al menos un archivo con -file1=<ruta>")
	}

	// Procesar cada parámetro
	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		if len(kv) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", match)
		}

		value := kv[1]

		// Quitar comillas si existen
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		// Validar que la ruta no esté vacía
		if value == "" {
			return "", errors.New("la ruta del archivo no puede estar vacía")
		}

		cmd.files = append(cmd.files, value)
	}

	// Ejecutar el comando
	content, err := commandCat(cmd)
	if err != nil {
		return "", err
	}

	return content, nil
}

// commandCat ejecuta la lógica del comando cat
func commandCat(cat *CAT) (string, error) {
	// 1. Verificar que hay una sesión activa
	if !stores.Auth.IsAuthenticated() {
		return "", errors.New("ERROR: No hay una sesión activa. Use el comando LOGIN primero")
	}

	// 2. Obtener la partición montada y el superbloque
	partitionID := stores.Auth.GetPartitionID()
	partitionSuperblock, partition, partitionPath, err := stores.GetMountedPartitionSuperblock(partitionID)
	if err != nil {
		return "", fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	fmt.Printf("DEBUG CAT -> Partition Start: %d\n", partition.Part_start)
	fmt.Printf("DEBUG CAT -> S_inode_start: %d\n", partitionSuperblock.S_inode_start)
	fmt.Printf("DEBUG CAT -> S_block_start: %d\n", partitionSuperblock.S_block_start)

	// 3. Obtener información del usuario actual
	currentUser, _, _ := stores.Auth.GetCurrentUser()

	// Obtener UID y GID del usuario actual
	userUID, userGID, err := getUserInfo(partitionSuperblock, partition, partitionPath, currentUser)
	if err != nil {
		return "", fmt.Errorf("error al obtener información del usuario: %w", err)
	}

	fmt.Printf("DEBUG CAT -> Usuario: %s, UID: %d, GID: %d\n", currentUser, userUID, userGID)

	// 4. Concatenar el contenido de todos los archivos
	var result strings.Builder

	for i, filePath := range cat.files {
		fmt.Printf("DEBUG CAT -> Buscando archivo: '%s'\n", filePath)

		// Leer el contenido del archivo
		content, err := readFileContent(partitionSuperblock, partition, partitionPath, filePath, userUID, userGID, currentUser)
		if err != nil {
			return "", fmt.Errorf("error al leer %s: %w", filePath, err)
		}

		// Agregar el contenido al resultado
		result.WriteString(content)

		// Agregar salto de línea entre archivos (excepto el último)
		if i < len(cat.files)-1 {
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

// getUserInfo obtiene el UID y GID del usuario actual
func getUserInfo(sb *structures.SuperBlock, partition *structures.Partition, path string, username string) (int32, int32, error) {
	// Leer el archivo users.txt
	usersContent, err := readUsersFileCat(sb, partition, path)
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(usersContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 5 {
			// Limpiar espacios
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}

			// Verificar si es un usuario (tipo U) y si no está eliminado
			if parts[1] == "U" && parts[0] != "0" && parts[3] == username {
				uid := int32(0)
				gid := int32(0)
				fmt.Sscanf(parts[0], "%d", &uid)

				// Buscar el GID del grupo
				groupName := parts[2]
				for _, gLine := range lines {
					gLine = strings.TrimSpace(gLine)
					if gLine == "" {
						continue
					}
					gParts := strings.Split(gLine, ",")
					if len(gParts) >= 3 {
						for i := range gParts {
							gParts[i] = strings.TrimSpace(gParts[i])
						}
						if gParts[1] == "G" && gParts[2] == groupName {
							fmt.Sscanf(gParts[0], "%d", &gid)
							break
						}
					}
				}

				return uid, gid, nil
			}
		}
	}

	return 0, 0, fmt.Errorf("usuario no encontrado")
}

// readFileContent lee el contenido de un archivo verificando permisos
func readFileContent(sb *structures.SuperBlock, partition *structures.Partition, path string, filePath string, userUID int32, userGID int32, username string) (string, error) {
	// Parsear la ruta del archivo
	filePath = strings.Trim(filePath, " ")

	// Normalizar la ruta - eliminar barras múltiples
	filePath = strings.ReplaceAll(filePath, "//", "/")

	// Separar la ruta en partes
	pathParts := strings.Split(strings.Trim(filePath, "/"), "/")

	// Filtrar partes vacías
	var validParts []string
	for _, part := range pathParts {
		if part != "" {
			validParts = append(validParts, part)
		}
	}

	fmt.Printf("DEBUG CAT -> PathParts válidas: %v\n", validParts)

	// Si no hay partes válidas, es inválida
	if len(validParts) == 0 {
		return "", fmt.Errorf("debe especificar un archivo válido")
	}

	// Buscar el archivo navegando por la estructura de directorios
	// Empezamos desde el inodo 0 (raíz)
	fileInode, err := findFileInode(sb, partition, path, validParts, 0)
	if err != nil {
		return "", err
	}

	fmt.Printf("DEBUG CAT -> Inodo encontrado: %d\n", fileInode)

	// Deserializar el inodo del archivo
	inode := &structures.Inode{}
	inodeOffset := int64(sb.S_inode_start + (fileInode * sb.S_inode_size))
	err = inode.Deserialize(path, inodeOffset)
	if err != nil {
		return "", fmt.Errorf("error al deserializar inodo: %w", err)
	}

	fmt.Printf("DEBUG CAT -> Tipo de inodo: %c (0=carpeta, 1=archivo)\n", inode.I_type[0])

	// Verificar que sea un archivo
	if inode.I_type[0] != '1' {
		return "", fmt.Errorf("%s es un directorio, no un archivo", filePath)
	}

	// Verificar permisos de lectura
	if !hasReadPermission(inode, userUID, userGID, username) {
		return "", fmt.Errorf("no tiene permisos de lectura sobre %s", filePath)
	}

	// Leer el contenido del archivo
	var content strings.Builder

	// Leer bloques directos (primeros 12)
	for i := 0; i < 12; i++ {
		blockIndex := inode.I_block[i]

		if blockIndex == -1 {
			break
		}

		// Deserializar el bloque de archivo
		block := &structures.FileBlock{}
		blockOffset := int64(sb.S_block_start + (blockIndex * sb.S_block_size))
		err := block.Deserialize(path, blockOffset)
		if err != nil {
			return "", fmt.Errorf("error al leer bloque: %w", err)
		}

		// Agregar contenido eliminando padding null
		blockContent := string(block.B_content[:])
		blockContent = strings.ReplaceAll(blockContent, "\x00", "")
		content.WriteString(blockContent)
	}

	return content.String(), nil
}

// findFileInode busca el inodo de un archivo navegando por la estructura de directorios
func findFileInode(sb *structures.SuperBlock, partition *structures.Partition, path string, pathParts []string, currentInodeIndex int32) (int32, error) {
	fmt.Printf("DEBUG findFileInode -> Buscando en inodo %d, pathParts restantes: %v\n", currentInodeIndex, pathParts)

	// Si no hay más partes de la ruta, retornar el inodo actual
	if len(pathParts) == 0 {
		return currentInodeIndex, nil
	}

	// Deserializar el inodo actual con el offset correcto
	inode := &structures.Inode{}
	inodeOffset := int64(sb.S_inode_start + (currentInodeIndex * sb.S_inode_size))
	err := inode.Deserialize(path, inodeOffset)
	if err != nil {
		return -1, fmt.Errorf("error al deserializar inodo %d: %w", currentInodeIndex, err)
	}

	fmt.Printf("DEBUG findFileInode -> Inodo %d tipo: %c\n", currentInodeIndex, inode.I_type[0])

	// Si quedan más partes de la ruta, el inodo actual debe ser un directorio
	if len(pathParts) > 1 && inode.I_type[0] != '0' {
		return -1, fmt.Errorf("la ruta contiene un archivo en lugar de un directorio")
	}

	// Buscar en los bloques del directorio
	targetName := pathParts[0]
	fmt.Printf("DEBUG findFileInode -> Buscando nombre: '%s'\n", targetName)

	for i := 0; i < 12; i++ {
		blockIndex := inode.I_block[i]

		if blockIndex == -1 {
			break
		}

		fmt.Printf("DEBUG findFileInode -> Revisando bloque %d\n", blockIndex)

		// Deserializar el bloque de carpeta con el offset correcto
		block := &structures.FolderBlock{}
		blockOffset := int64(sb.S_block_start + (blockIndex * sb.S_block_size))
		err := block.Deserialize(path, blockOffset)
		if err != nil {
			fmt.Printf("DEBUG findFileInode -> Error al deserializar bloque: %v\n", err)
			continue
		}

		// Buscar el archivo/carpeta en el bloque
		for j, content := range block.B_content {
			name := strings.Trim(string(content.B_name[:]), "\x00 ")

			// Debug: mostrar cada entrada
			fmt.Printf("DEBUG findFileInode -> Entrada %d: nombre='%s' (bytes: %v), inodo=%d\n",
				j, name, content.B_name[:], content.B_inodo)

			// Ignorar entradas . y ..
			if name == "." || name == ".." {
				continue
			}

			// Comparación case-insensitive
			if strings.EqualFold(name, targetName) && content.B_inodo != -1 {
				fmt.Printf("DEBUG findFileInode -> ¡Coincidencia encontrada! inodo=%d\n", content.B_inodo)

				// Si es el último elemento de la ruta, retornar este inodo
				if len(pathParts) == 1 {
					return content.B_inodo, nil
				}

				// Si no, seguir navegando
				return findFileInode(sb, partition, path, pathParts[1:], content.B_inodo)
			}
		}
	}

	return -1, fmt.Errorf("archivo o directorio no encontrado: %s", targetName)
}

// hasReadPermission verifica si el usuario tiene permiso de lectura
func hasReadPermission(inode *structures.Inode, userUID int32, userGID int32, username string) bool {
	// El usuario root siempre tiene todos los permisos
	if username == "root" {
		return true
	}

	// Obtener los permisos del archivo
	perms := string(inode.I_perm[:])

	// Determinar qué conjunto de permisos aplicar
	var permBit byte

	if inode.I_uid == userUID {
		// Es el propietario (User)
		permBit = perms[0]
	} else if inode.I_gid == userGID {
		// Pertenece al mismo grupo (Group)
		permBit = perms[1]
	} else {
		// Otros usuarios (Others)
		permBit = perms[2]
	}

	// Convertir el permiso a número
	perm := int(permBit - '0')

	// Verificar permiso de lectura (bit 2 en octal)
	return (perm & 4) != 0
}

// readUsersFileCat lee el archivo users.txt (inodo 1)
func readUsersFileCat(sb *structures.SuperBlock, partition *structures.Partition, path string) (string, error) {
	fmt.Println("DEBUG readUsersFileCat -> Iniciando lectura de users.txt")

	inode := &structures.Inode{}

	// El superbloque ya contiene los offsets absolutos
	inodeOffset := int64(sb.S_inode_start + (1 * sb.S_inode_size))
	fmt.Printf("DEBUG readUsersFileCat -> Offset inodo: %d\n", inodeOffset)

	err := inode.Deserialize(path, inodeOffset)
	if err != nil {
		return "", fmt.Errorf("error al deserializar inodo users.txt: %w", err)
	}

	fmt.Printf("DEBUG readUsersFileCat -> Tipo inodo: %c\n", inode.I_type[0])

	if inode.I_type[0] != '1' {
		return "", errors.New("el inodo 1 no es un archivo")
	}

	var content strings.Builder

	for i := 0; i < 12; i++ {
		blockIndex := inode.I_block[i]

		if blockIndex == -1 {
			break
		}

		block := &structures.FileBlock{}
		blockOffset := int64(sb.S_block_start + (blockIndex * sb.S_block_size))
		fmt.Printf("DEBUG readUsersFileCat -> Leyendo bloque %d en offset %d\n", blockIndex, blockOffset)

		err := block.Deserialize(path, blockOffset)
		if err != nil {
			return "", fmt.Errorf("error al deserializar bloque %d: %w", blockIndex, err)
		}

		blockContent := string(block.B_content[:])
		blockContent = strings.ReplaceAll(blockContent, "\x00", "")
		content.WriteString(blockContent)
	}

	fmt.Printf("DEBUG readUsersFileCat -> Contenido leído: '%s'\n", content.String())
	return content.String(), nil
}
