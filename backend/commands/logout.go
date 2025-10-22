package commands

import (
	stores "backend/stores"
	"errors"
	"fmt"
)

// LOGOUT estructura que representa el comando logout
type LOGOUT struct{}

/*
	logout
*/

func ParseLogout(tokens []string) (string, error) {
	// Verificar que no haya parámetros adicionales
	if len(tokens) > 0 {
		return "", fmt.Errorf("el comando logout no acepta parámetros")
	}

	cmd := &LOGOUT{} // Crea una nueva instancia de LOGOUT

	// Ejecutar el comando logout
	err := commandLogout(cmd)
	if err != nil {
		return "", err
	}

	return "LOGOUT: Se ha cerrado la sesión correctamente", nil
}

func commandLogout(logout *LOGOUT) error {
	// Verificar si hay una sesión iniciada
	if !stores.Auth.IsAuthenticated() {
		return errors.New("no hay ninguna sesión iniciada")
	}

	// Obtener la información de la sesión actual antes de cerrarla
	username, _, partitionID := stores.Auth.GetCurrentUser()

	// Cerrar la sesión
	stores.Auth.Logout()

	fmt.Printf("Se ha cerrado la sesión de %s en la partición %s\n", username, partitionID)
	return nil
}
