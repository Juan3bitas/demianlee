package main

import "os"

// Función getHostname: Obtiene el nombre del host directamente del sistema
// operativo donde se está ejecutando la aplicación. En contenedores
// también se respeta la variable de entorno HOSTNAME si el sistema operativo
// no la expone igual.
func getHostname() string {
	if h := os.Getenv("HOSTNAME"); h != "" {
		return h
	}
	h, err := os.Hostname()
	if err != nil {
		return "desconocido"
	}
	return h
}

// Función getServerName: Es el nombre "amigable" del servidor/entorno donde se
// desplegó la aplicación (por ejemplo "Servidor-Produccion-1" o el
// nombre de la máquina virtual). Se configura con la variable de
// entorno SERVER_NAME; si no está definida, se usa el hostname como
// valor por defecto.
func getServerName() string {
	if s := os.Getenv("SERVER_NAME"); s != "" {
		return s
	}
	return getHostname()
}
