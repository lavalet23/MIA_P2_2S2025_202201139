package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	stores "backend/stores"
	structures "backend/structures"
)

// ParseMkfs procesa el comando MKFS
func ParseMkfs(tokens []string) (string, error) {
	id := ""
	ftype := "full" // valor por defecto
	fs := "2fs"     // valor por defecto EXT2

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		lowerToken := strings.ToLower(token)

		if strings.HasPrefix(lowerToken, "-id=") {
			id = token[len("-id="):]
		} else if strings.HasPrefix(lowerToken, "-type=") {
			ftype = strings.ToLower(token[len("-type="):])
			if ftype != "full" {
				return "", fmt.Errorf("MKFS ERROR: tipo inválido '%s' (solo 'full')", ftype)
			}
		} else if strings.HasPrefix(lowerToken, "-fs=") {
			fs = strings.ToLower(token[len("-fs="):])
			if fs != "2fs" && fs != "3fs" {
				return "", fmt.Errorf("MKFS ERROR: sistema de archivos inválido '%s' (solo '2fs' o '3fs')", fs)
			}
		} else if token != "" && token != "mkfs" {
			return "", fmt.Errorf("MKFS ERROR: parámetro no reconocido '%s'", token)
		}
	}

	if id == "" {
		return "", fmt.Errorf("MKFS ERROR: el parámetro -id es obligatorio")
	}

	if err := Mkfs(id, ftype, fs); err != nil {
		return "", err
	}

	fsType := "EXT2"
	if fs == "3fs" {
		fsType = "EXT3"
	}
	return fmt.Sprintf("MKFS: Formateo completado con éxito en %s", fsType), nil
}

// Mkfs formatea una partición con EXT2 o EXT3
func Mkfs(id string, ftype string, fs string) error {
	// 1) Resolver id -> path/offset/size
	pathEntry, ok := stores.MountedPartitions[id]
	if !ok {
		// Búsqueda case-insensitive
		for k, v := range stores.MountedPartitions {
			if strings.EqualFold(k, id) {
				pathEntry = v
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("partición %s no encontrada o no montada", id)
		}
	}

	// Parsear pathEntry: formato esperado "path|start|size"
	var diskPath string
	var partStart int64
	var partSize int64

	if strings.Contains(pathEntry, "|") {
		parts := strings.Split(pathEntry, "|")
		diskPath = parts[0]
		if len(parts) > 1 {
			fmt.Sscan(parts[1], &partStart)
		}
		if len(parts) > 2 {
			fmt.Sscan(parts[2], &partSize)
		}
	} else {
		diskPath = pathEntry
		// Necesitamos obtener start y size del MBR
		mbr := &structures.MBR{}
		if err := mbr.Deserialize(diskPath); err != nil {
			return fmt.Errorf("error al leer MBR: %v", err)
		}

		// Buscar la partición por ID
		partition, err := mbr.GetPartitionByID(id)
		if err != nil {
			return fmt.Errorf("partición no encontrada en MBR: %v", err)
		}

		partStart = int64(partition.Part_start)
		partSize = int64(partition.Part_size)
	}

	// 2) Abrir archivo disco
	f, err := os.OpenFile(diskPath, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("no se pudo abrir disco %s: %v", diskPath, err)
	}
	defer f.Close()

	// 3) Calcular número de estructuras según el sistema de archivos
	var n int64

	if fs == "3fs" {
		// EXT3: tamaño_particion = sizeof(superblock) + 50*sizeof(journal) + n + 3*n + n*sizeof(inodo) + 3*n*sizeof(block)
		// sizeof(superblock) = 68 bytes
		// sizeof(journal) = aprox 256 bytes (content + metadata)
		// sizeof(inodo) = 128 bytes
		// sizeof(block) = 64 bytes

		journalSize := int64(50 * 256) // 50 journals de 256 bytes cada uno
		superblockSize := int64(68)

		// tamaño_particion = 68 + 12800 + n + 3n + 128n + 192n = 12868 + 324n
		// n = (tamaño_particion - 12868) / 324
		n = (partSize - superblockSize - journalSize) / (1 + 3 + 128 + 192)

	} else {
		// EXT2: tamaño_particion = sizeof(superblock) + n + 3*n + n*sizeof(inodo) + 3*n*sizeof(block)
		// sizeof(superblock) = 68 bytes
		// tamaño_particion = 68 + n + 3n + 128n + 192n = 68 + 324n
		// n = (tamaño_particion - 68) / 324
		n = (partSize - 68) / (1 + 3 + 128 + 192)
	}

	if n < 3 {
		n = 3 // mínimo razonable
	}

	// 4) Crear SuperBlock
	sb := structures.SuperBlock{}

	if fs == "3fs" {
		sb.S_filesystem_type = 3 // EXT3
	} else {
		sb.S_filesystem_type = 2 // EXT2
	}

	sb.S_inodes_count = int32(n)
	sb.S_blocks_count = int32(3 * n)
	sb.S_free_inodes_count = int32(n - 2)   // Reservamos root y users.txt
	sb.S_free_blocks_count = int32(3*n - 2) // Reservamos 2 bloques
	sb.S_mtime = float32(time.Now().Unix())
	sb.S_umtime = 0
	sb.S_mnt_count = 0
	sb.S_magic = 0xEF53
	sb.S_inode_size = 128
	sb.S_block_size = 64

	// IMPORTANTE: Los offsets en el superbloque son ABSOLUTOS (incluyen partStart)
	// Calcular offsets absolutos desde el inicio del disco
	currentOffset := int32(partStart)

	// Superblock en el inicio de la partición
	currentOffset += 68

	// Si es EXT3, añadir espacio para journal
	if fs == "3fs" {
		currentOffset += int32(50 * 256) // 50 journals
	}

	sb.S_bm_inode_start = currentOffset
	currentOffset += int32(n)

	sb.S_bm_block_start = currentOffset
	currentOffset += int32(3 * n)

	sb.S_inode_start = currentOffset
	sb.S_first_ino = currentOffset // Primer inodo libre (al principio son iguales)
	currentOffset += int32(n * 128)

	sb.S_block_start = currentOffset
	sb.S_first_blo = currentOffset // Primer bloque libre (al principio son iguales)

	// 5) Escribir Superblock
	if err := sb.Serialize(diskPath, partStart); err != nil {
		return fmt.Errorf("error al escribir superblock: %v", err)
	}

	// 6) Inicializar Journal si es EXT3
	if fs == "3fs" {
		journalStart := partStart + 68
		// Inicializar 50 journals vacíos
		emptyJournal := make([]byte, 256)
		for i := 0; i < 50; i++ {
			if _, err := f.Seek(journalStart+int64(i*256), 0); err != nil {
				return fmt.Errorf("error al posicionar journal: %v", err)
			}
			if _, err := f.Write(emptyJournal); err != nil {
				return fmt.Errorf("error al escribir journal: %v", err)
			}
		}
	}

	// 7) Inicializar Bitmaps
	// Bitmap de inodos
	if _, err := f.Seek(int64(sb.S_bm_inode_start), 0); err != nil {
		return fmt.Errorf("error al posicionar bitmap inodos: %v", err)
	}
	bmInodes := make([]byte, n)
	bmInodes[0] = 1 // inodo root usado
	bmInodes[1] = 1 // inodo users.txt usado
	if _, err := f.Write(bmInodes); err != nil {
		return fmt.Errorf("error al escribir bitmap inodos: %v", err)
	}

	// Bitmap de bloques
	if _, err := f.Seek(int64(sb.S_bm_block_start), 0); err != nil {
		return fmt.Errorf("error al posicionar bitmap bloques: %v", err)
	}
	bmBlocks := make([]byte, 3*n)
	bmBlocks[0] = 1 // bloque carpeta root usado
	bmBlocks[1] = 1 // bloque archivo users.txt usado
	if _, err := f.Write(bmBlocks); err != nil {
		return fmt.Errorf("error al escribir bitmap bloques: %v", err)
	}

	// 8) Crear inodo root (inodo 0)
	rootInode := structures.Inode{
		I_uid:   1,
		I_gid:   1,
		I_size:  0,
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{0, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		I_type:  [1]byte{'0'}, // carpeta
		I_perm:  [3]byte{'7', '7', '7'},
	}

	if err := rootInode.Serialize(diskPath, int64(sb.S_inode_start)); err != nil {
		return fmt.Errorf("error al escribir inodo root: %v", err)
	}

	// 9) Crear inodo users.txt (inodo 1)
	usersContent := "1,G,root\n1,U,root,root,123\n"
	usersInode := structures.Inode{
		I_uid:   1,
		I_gid:   1,
		I_size:  int32(len(usersContent)),
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		I_type:  [1]byte{'1'}, // archivo
		I_perm:  [3]byte{'6', '6', '4'},
	}

	if err := usersInode.Serialize(diskPath, int64(sb.S_inode_start+128)); err != nil {
		return fmt.Errorf("error al escribir inodo users: %v", err)
	}

	// 10) Crear bloque carpeta root (bloque 0)
	rootBlock := structures.FolderBlock{
		B_content: [4]structures.FolderContent{
			{B_name: [12]byte{'.', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, B_inodo: 0},
			{B_name: [12]byte{'.', '.', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, B_inodo: 0},
			{B_name: [12]byte{'u', 's', 'e', 'r', 's', '.', 't', 'x', 't', 0, 0, 0}, B_inodo: 1},
			{B_name: [12]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, B_inodo: -1},
		},
	}

	if err := rootBlock.Serialize(diskPath, int64(sb.S_block_start)); err != nil {
		return fmt.Errorf("error al escribir bloque root: %v", err)
	}

	// 11) Crear bloque archivo users.txt (bloque 1)
	usersBlock := structures.FileBlock{}
	copy(usersBlock.B_content[:], usersContent)

	if err := usersBlock.Serialize(diskPath, int64(sb.S_block_start+64)); err != nil {
		return fmt.Errorf("error al escribir bloque users: %v", err)
	}

	// 12) Actualizar superblock con valores finales
	if err := sb.Serialize(diskPath, partStart); err != nil {
		return fmt.Errorf("error al actualizar superblock: %v", err)
	}

	fsType := "EXT2"
	if fs == "3fs" {
		fsType = "EXT3"
	}

	fmt.Printf("MKFS: Partición %s formateada en %s\n", id, fsType)
	fmt.Printf("  Disco: %s\n", diskPath)
	fmt.Printf("  Start: %d, Size: %d bytes\n", partStart, partSize)
	fmt.Printf("  Inodos: %d (libres: %d)\n", n, n-2)
	fmt.Printf("  Bloques: %d (libres: %d)\n", 3*n, 3*n-2)
	fmt.Printf("  S_inode_start: %d\n", sb.S_inode_start)
	fmt.Printf("  S_block_start: %d\n", sb.S_block_start)

	return nil
}
