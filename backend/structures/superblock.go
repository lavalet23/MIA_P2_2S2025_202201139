package structures

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"time"
)

type SuperBlock struct {
	S_filesystem_type   int32
	S_inodes_count      int32
	S_blocks_count      int32
	S_free_inodes_count int32
	S_free_blocks_count int32
	S_mtime             float32
	S_umtime            float32
	S_mnt_count         int32
	S_magic             int32
	S_inode_size        int32
	S_block_size        int32
	S_first_ino         int32
	S_first_blo         int32
	S_bm_inode_start    int32
	S_bm_block_start    int32
	S_inode_start       int32
	S_block_start       int32
	// Total: 68 bytes
}

// Serialize escribe la estructura SuperBlock en un archivo binario en la posición especificada
func (sb *SuperBlock) Serialize(path string, offset int64) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Mover el puntero del archivo a la posición especificada
	_, err = file.Seek(offset, 0)
	if err != nil {
		return err
	}

	// Serializar la estructura SuperBlock directamente en el archivo
	err = binary.Write(file, binary.LittleEndian, sb)
	if err != nil {
		return err
	}

	return nil
}

// Deserialize lee la estructura SuperBlock desde un archivo binario en la posición especificada
func (sb *SuperBlock) Deserialize(path string, offset int64) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Mover el puntero del archivo a la posición especificada
	_, err = file.Seek(offset, 0)
	if err != nil {
		return err
	}

	// Obtener el tamaño de la estructura SuperBlock
	sbSize := binary.Size(sb)
	if sbSize <= 0 {
		return fmt.Errorf("invalid SuperBlock size: %d", sbSize)
	}

	// Leer solo la cantidad de bytes que corresponden al tamaño de la estructura SuperBlock
	buffer := make([]byte, sbSize)
	_, err = file.Read(buffer)
	if err != nil {
		return err
	}

	// Deserializar los bytes leídos en la estructura SuperBlock
	reader := bytes.NewReader(buffer)
	err = binary.Read(reader, binary.LittleEndian, sb)
	if err != nil {
		return err
	}

	return nil
}

// PrintSuperBlock imprime los valores de la estructura SuperBlock
func (sb *SuperBlock) Print() {
	// Convertir el tiempo de montaje a una fecha
	mountTime := time.Unix(int64(sb.S_mtime), 0)
	// Convertir el tiempo de desmontaje a una fecha
	unmountTime := time.Unix(int64(sb.S_umtime), 0)

	fmt.Printf("Filesystem Type: %d\n", sb.S_filesystem_type)
	fmt.Printf("Inodes Count: %d\n", sb.S_inodes_count)
	fmt.Printf("Blocks Count: %d\n", sb.S_blocks_count)
	fmt.Printf("Free Inodes Count: %d\n", sb.S_free_inodes_count)
	fmt.Printf("Free Blocks Count: %d\n", sb.S_free_blocks_count)
	fmt.Printf("Mount Time: %s\n", mountTime.Format(time.RFC3339))
	fmt.Printf("Unmount Time: %s\n", unmountTime.Format(time.RFC3339))
	fmt.Printf("Mount Count: %d\n", sb.S_mnt_count)
	fmt.Printf("Magic: %d\n", sb.S_magic)
	fmt.Printf("Inode Size: %d\n", sb.S_inode_size)
	fmt.Printf("Block Size: %d\n", sb.S_block_size)
	fmt.Printf("First Inode: %d\n", sb.S_first_ino)
	fmt.Printf("First Block: %d\n", sb.S_first_blo)
	fmt.Printf("Bitmap Inode Start: %d\n", sb.S_bm_inode_start)
	fmt.Printf("Bitmap Block Start: %d\n", sb.S_bm_block_start)
	fmt.Printf("Inode Start: %d\n", sb.S_inode_start)
	fmt.Printf("Block Start: %d\n", sb.S_block_start)
}

// Imprimir inodos
func (sb *SuperBlock) PrintInodes(path string) error {
	// Imprimir inodos
	fmt.Println("\nInodos\n----------------")
	// Iterar sobre cada inodo
	for i := int32(0); i < sb.S_inodes_count; i++ {
		inode := &Inode{}
		// Deserializar el inodo
		err := inode.Deserialize(path, int64(sb.S_inode_start+(i*sb.S_inode_size)))
		if err != nil {
			return err
		}
		// Imprimir el inodo
		fmt.Printf("\nInodo %d:\n", i)
		inode.Print()
	}

	return nil
}

// Impriir bloques
func (sb *SuperBlock) PrintBlocks(path string) error {
	// Imprimir bloques
	fmt.Println("\nBloques\n----------------")
	// Iterar sobre cada inodo
	for i := int32(0); i < sb.S_inodes_count; i++ {
		inode := &Inode{}
		// Deserializar el inodo
		err := inode.Deserialize(path, int64(sb.S_inode_start+(i*sb.S_inode_size)))
		if err != nil {
			return err
		}
		// Iterar sobre cada bloque del inodo (apuntadores)
		for _, blockIndex := range inode.I_block {
			// Si el bloque no existe, salir
			if blockIndex == -1 {
				break
			}
			// Si el inodo es de tipo carpeta
			if inode.I_type[0] == '0' {
				block := &FolderBlock{}
				// Deserializar el bloque
				err := block.Deserialize(path, int64(sb.S_block_start+(blockIndex*sb.S_block_size))) // 64 porque es el tamaño de un bloque
				if err != nil {
					return err
				}
				// Imprimir el bloque
				fmt.Printf("\nBloque %d:\n", blockIndex)
				block.Print()
				continue

				// Si el inodo es de tipo archivo
			} else if inode.I_type[0] == '1' {
				block := &FileBlock{}
				// Deserializar el bloque
				err := block.Deserialize(path, int64(sb.S_block_start+(blockIndex*sb.S_block_size))) // 64 porque es el tamaño de un bloque
				if err != nil {
					return err
				}
				// Imprimir el bloque
				fmt.Printf("\nBloque %d:\n", blockIndex)
				block.Print()
				continue
			}

		}
	}

	return nil
}

// Get users.txt block
func (sb *SuperBlock) GetUsersBlock(path string) (*FileBlock, error) {
	// Ir al inodo 1
	inode := &Inode{}

	// Deserializar el inodo
	err := inode.Deserialize(path, int64(sb.S_inode_start+(1*sb.S_inode_size))) // 1 porque es el inodo 1
	if err != nil {
		return nil, err
	}

	// Iterar sobre cada bloque del inodo (apuntadores)
	for _, blockIndex := range inode.I_block {
		// Si el bloque no existe, salir
		if blockIndex == -1 {
			break
		}
		// Si el inodo es de tipo archivo
		if inode.I_type[0] == '1' {
			block := &FileBlock{}
			// Deserializar el bloque
			err := block.Deserialize(path, int64(sb.S_block_start+(blockIndex*sb.S_block_size))) // 64 porque es el tamaño de un bloque
			if err != nil {
				return nil, err
			}
			// Deben ir guardando todo el contenido de los bloques en una variable

			// Retornar el bloque por temas explicativos
			return block, nil
		}
	}
	return nil, fmt.Errorf("users.txt block not found")
}

// CreateFolder crea una carpeta en el sistema de archivos
func (sb *SuperBlock) CreateFolder(path string, parentsDir []string, destDir string) error {

	// Validar el sistema de archivos
	if sb.S_filesystem_type == 3 {
		// Si parentsDir está vacío, solo trabajar con el primer inodo que sería el raíz "/"
		if len(parentsDir) == 0 {
			return sb.createFolderInInodeExt3(path, 0, parentsDir, destDir)
		}

		// Iterar sobre cada inodo ya que se necesita buscar el inodo padre
		for i := int32(0); i < sb.S_inodes_count; i++ {
			err := sb.createFolderInInodeExt3(path, i, parentsDir, destDir)
			if err != nil {
				return err
			}
		}
	} else {
		// Si parentsDir está vacío, solo trabajar con el primer inodo que sería el raíz "/"
		if len(parentsDir) == 0 {
			return sb.createFolderInInodeExt2(path, 0, parentsDir, destDir)
		}

		// Iterar sobre cada inodo ya que se necesita buscar el inodo padre
		for i := int32(0); i < sb.S_inodes_count; i++ {
			err := sb.createFolderInInodeExt2(path, i, parentsDir, destDir)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (sb *SuperBlock) CreateFile(path string, parentsDir []string, destFile string, size int, cont []string) error {
	// Preparar el contenido del archivo
	var content string
	if len(cont) > 0 {
		content = strings.Join(cont, "\n")
	} else {
		// Si no se proporciona contenido, generar contenido aleatorio o vacío
		content = strings.Repeat("0", size)
	}

	// Si parentsDir está vacío, significa que se creará en el directorio raíz
	if len(parentsDir) == 0 {
		return sb.createFileInInodeExt2(path, 0, parentsDir, destFile, size, content)
	}

	// Iterar sobre cada inodo para encontrar el directorio padre
	for i := int32(0); i < sb.S_inodes_count; i++ {
		// Deserializar el inodo actual
		currentInode := &Inode{}
		err := currentInode.Deserialize(path, int64(sb.S_inode_start+(i*sb.S_inode_size)))
		if err != nil {
			continue
		}

		// Verificar si es un inodo de directorio
		if currentInode.I_type[0] != '0' {
			continue
		}

		// Verificar si este inodo corresponde al directorio padre
		if sb.isParentDirectory(path, i, parentsDir) {
			// Intentar crear el archivo en este inodo
			return sb.createFileInInodeExt2(path, i, parentsDir, destFile, size, content)
		}
	}

	return fmt.Errorf("directorio padre no encontrado")
}

// Método auxiliar para verificar si un inodo es el directorio padre
func (sb *SuperBlock) isParentDirectory(path string, inodeIndex int32, parentsDir []string) bool {
	// Deserializar el inodo
	inode := &Inode{}
	err := inode.Deserialize(path, int64(sb.S_inode_start+(inodeIndex*sb.S_inode_size)))
	if err != nil {
		return false
	}

	// Verificar que sea un inodo de directorio
	if inode.I_type[0] != '0' {
		return false
	}

	// Iterar sobre los bloques del inodo
	for _, blockIndex := range inode.I_block {
		if blockIndex == -1 {
			break
		}

		// Deserializar el bloque de directorio
		block := &FolderBlock{}
		err := block.Deserialize(path, int64(sb.S_block_start+(blockIndex*sb.S_block_size)))
		if err != nil {
			continue
		}

		// Buscar coincidencia con el primer directorio padre
		if len(parentsDir) > 0 {
			firstParent := parentsDir[0]
			for _, content := range block.B_content {
				contentName := strings.Trim(string(content.B_name[:]), "\x00 ")
				if strings.EqualFold(contentName, firstParent) {
					return true
				}
			}
		}
	}

	return false
}

// Método para crear el archivo en un inodo específico
func (sb *SuperBlock) createFileInInodeExt2(path string, inodeIndex int32, parentsDir []string, destFile string, size int, content string) error {
	// Deserializar el inodo padre
	parentInode := &Inode{}
	err := parentInode.Deserialize(path, int64(sb.S_inode_start+(inodeIndex*sb.S_inode_size)))
	if err != nil {
		return err
	}

	// Deserializar el bloque padre
	parentBlock := &FolderBlock{}
	err = parentBlock.Deserialize(path, int64(sb.S_block_start+(parentInode.I_block[0]*sb.S_block_size)))
	if err != nil {
		return err
	}

	// Crear nuevo inodo para el archivo
	fileInode := &Inode{
		I_uid:   1,
		I_gid:   1,
		I_size:  int32(len(content)),
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{sb.S_blocks_count, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		I_type:  [1]byte{'1'}, // Tipo archivo
		I_perm:  [3]byte{'7', '7', '7'},
	}

	// Actualizar bitmap de inodos
	err = sb.UpdateBitmapInode(path)
	if err != nil {
		return err
	}

	// Serializar el inodo del archivo
	err = fileInode.Serialize(path, int64(sb.S_first_ino))
	if err != nil {
		return err
	}

	// Actualizar contadores de inodos
	sb.S_inodes_count++
	sb.S_free_inodes_count--
	sb.S_first_ino += sb.S_inode_size

	// Crear bloque de archivo
	fileBlock := &FileBlock{
		B_content: [64]byte{},
	}
	copy(fileBlock.B_content[:], content)

	// Serializar bloque de archivo
	err = fileBlock.Serialize(path, int64(sb.S_first_blo))
	if err != nil {
		return err
	}

	// Actualizar bitmap de bloques
	err = sb.UpdateBitmapBlock(path)
	if err != nil {
		return err
	}

	// Actualizar contadores de bloques
	sb.S_blocks_count++
	sb.S_free_blocks_count--
	sb.S_first_blo += sb.S_block_size

	// Actualizar bloque padre para añadir referencia al nuevo archivo
	for i := 2; i < len(parentBlock.B_content); i++ {
		if parentBlock.B_content[i].B_inodo == -1 {
			copy(parentBlock.B_content[i].B_name[:], destFile)
			parentBlock.B_content[i].B_inodo = sb.S_inodes_count - 1
			break
		}
	}

	// Serializar bloque padre actualizado
	err = parentBlock.Serialize(path, int64(sb.S_block_start+(parentInode.I_block[0]*sb.S_block_size)))
	if err != nil {
		return err
	}

	return nil
}
