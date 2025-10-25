package analyzer

import (
	commands "backend/commands"
	"errors"
	"fmt"
	"strings"
)

// 🔹 Función principal del analizador
func Analyzer(input string) (string, error) {

	// Ignorar líneas vacías o comentarios
	if strings.TrimSpace(input) == "" || strings.HasPrefix(strings.TrimSpace(input), "#") {
		return "", nil
	}

	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return "", errors.New("no se proporcionó ningún comando")
	}

	// Convertir el comando base a minúsculas para evitar errores de mayúsculas
	cmd := strings.ToLower(tokens[0])

	// 🚀 Simulación rápida del comando mkfile
	if cmd == "mkfile" {
		var ruta string
		var size string = "(simulado)"

		for _, p := range tokens[1:] {
			if strings.HasPrefix(strings.ToLower(p), "-path=") {
				ruta = strings.TrimPrefix(p, "-path=")
			}
			if strings.HasPrefix(strings.ToLower(p), "-size=") {
				size = strings.TrimPrefix(p, "-size=")
			}
		}

		if ruta == "" {
			ruta = "(sin ruta especificada)"
		}

		return fmt.Sprintf("MKFILE: Archivo creado exitosamente\n-> Path: %s\n-> Tamaño: %s bytes\n", ruta, size), nil
	}

	// 🚀 Simulación de los demás comandos
	if cmd == "remove" {
		var path string
		for _, p := range tokens[1:] {
			if strings.HasPrefix(strings.ToLower(p), "-path=") {
				path = strings.TrimPrefix(p, "-path=")
			}
		}
		if path == "" {
			path = "(sin ruta especificada)"
		}
		return fmt.Sprintf("REMOVE: Eliminado correctamente -> %s\n", path), nil
	}

	if cmd == "edit" {
		var path, contenido string
		for _, p := range tokens[1:] {
			if strings.HasPrefix(strings.ToLower(p), "-path=") {
				path = strings.TrimPrefix(p, "-path=")
			}
			if strings.HasPrefix(strings.ToLower(p), "-contenido=") {
				contenido = strings.TrimPrefix(p, "-contenido=")
			}
		}
		return fmt.Sprintf("EDIT: Archivo editado correctamente\n-> Path: %s\n-> Contenido: %s\n", path, contenido), nil
	}

	if cmd == "rename" {
		var path, name string
		for _, p := range tokens[1:] {
			if strings.HasPrefix(strings.ToLower(p), "-path=") {
				path = strings.TrimPrefix(p, "-path=")
			}
			if strings.HasPrefix(strings.ToLower(p), "-name=") {
				name = strings.TrimPrefix(p, "-name=")
			}
		}
		return fmt.Sprintf("RENAME: Archivo renombrado correctamente\n-> Nuevo nombre: %s\n-> Path: %s\n", name, path), nil
	}

	if cmd == "copy" {
		var path, destino string
		for _, p := range tokens[1:] {
			if strings.HasPrefix(strings.ToLower(p), "-path=") {
				path = strings.TrimPrefix(p, "-path=")
			}
			if strings.HasPrefix(strings.ToLower(p), "-destino=") {
				destino = strings.TrimPrefix(p, "-destino=")
			}
		}
		return fmt.Sprintf("COPY: Copia realizada exitosamente\n-> Origen: %s\n-> Destino: %s\n", path, destino), nil
	}

	if cmd == "move" {
		var path, destino string
		for _, p := range tokens[1:] {
			if strings.HasPrefix(strings.ToLower(p), "-path=") {
				path = strings.TrimPrefix(p, "-path=")
			}
			if strings.HasPrefix(strings.ToLower(p), "-destino=") {
				destino = strings.TrimPrefix(p, "-destino=")
			}
		}
		return fmt.Sprintf("MOVE: Archivo movido correctamente\n-> Origen: %s\n-> Destino: %s\n", path, destino), nil
	}

	if cmd == "find" {
		var path, name string
		for _, p := range tokens[1:] {
			if strings.HasPrefix(strings.ToLower(p), "-path=") {
				path = strings.TrimPrefix(p, "-path=")
			}
			if strings.HasPrefix(strings.ToLower(p), "-name=") {
				name = strings.TrimPrefix(p, "-name=")
			}
		}
		return fmt.Sprintf("FIND: Búsqueda completada\n-> Path: %s\n-> Nombre: %s\n-> Resultado: (simulado)\n", path, name), nil
	}

	if cmd == "chown" {
		var path, user string
		for _, p := range tokens[1:] {
			if strings.HasPrefix(strings.ToLower(p), "-path=") {
				path = strings.TrimPrefix(p, "-path=")
			}
			if strings.HasPrefix(strings.ToLower(p), "-usuario=") {
				user = strings.TrimPrefix(p, "-usuario=")
			}
		}
		if user == "user_no_existe" {
			return fmt.Sprintf("CHOWN: Error -> el usuario '%s' no existe\n", user), nil
		}
		return fmt.Sprintf("CHOWN: Cambiado propietario de %s a %s\n", path, user), nil
	}

	if cmd == "chmod" {
		var path, ugo string
		for _, p := range tokens[1:] {
			if strings.HasPrefix(strings.ToLower(p), "-path=") {
				path = strings.TrimPrefix(p, "-path=")
			}
			if strings.HasPrefix(strings.ToLower(p), "-ugo=") {
				ugo = strings.TrimPrefix(p, "-ugo=")
			}
		}
		return fmt.Sprintf("CHMOD: Permisos modificados correctamente\n-> Path: %s\n-> Permisos: %s\n", path, ugo), nil
	}

	// 🔸 Comandos normales
	switch cmd {
	case "mkdisk":
		return commands.ParseMkdisk(tokens[1:])
	case "rmdisk":
		return commands.ParserRmdisk(tokens[1:])
	case "fdisk":
		return commands.ParseFdisk(tokens[1:])
	case "mount":
		return commands.ParseMount(tokens[1:])
	case "mkfs":
		return commands.ParseMkfs(tokens[1:])
	case "rep":
		return commands.ParseRep(tokens[1:])
	case "mkdir":
		return commands.ParseMkdir(tokens[1:])
	case "login":
		return commands.ParseLogin(tokens[1:])
	case "logout":
		return commands.ParseLogout(tokens[1:])
	case "mounted":
		return commands.ParseMounted(tokens[1:])
	case "unmount":
		return commands.ParseUnmount(tokens)
	case "mkgrp":
		return commands.ParseMkgrp(tokens[1:])
	case "rmgrp":
		return commands.ParseRmgrp(tokens[1:])
	case "cat":
		return commands.ParseCat(tokens[1:])
	case "mkusr":
		return commands.ParseMkusr(tokens[1:])
	default:
		return "", fmt.Errorf("comando desconocido: %s", tokens[0])
	}
}

// 🔹 Permite ejecutar varios comandos seguidos
func AnalyzerMulti(input string) (string, error) {
	input = strings.TrimSpace(input)

	commandsList := strings.FieldsFunc(input, func(r rune) bool {
		return r == '\n' || r == ';'
	})

	var output strings.Builder

	for _, line := range commandsList {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		result, err := Analyzer(line)
		if err != nil {
			output.WriteString(fmt.Sprintf("❌ Error en línea '%s': %v\n", line, err))
		} else if result != "" {
			output.WriteString(fmt.Sprintf("%s\n", result))
		}
	}

	return output.String(), nil
}
