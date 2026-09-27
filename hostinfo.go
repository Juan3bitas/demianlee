package main

import (
	"net"
	"net/http"
	"os"
	"strings"
)

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

// Función getClientIP: Identifica la IP del visitante que hace la petición.
// Cuando la app corre detrás de nginx (balanceo de carga), la conexión TCP
// real le llega desde el propio nginx, no desde el navegador del usuario;
// por eso primero se revisan las cabeceras que un proxy reenvía con la IP
// original (X-Real-IP / X-Forwarded-For, esta última puede traer varias IPs
// separadas por coma si hay más de un proxy en la cadena). Si no existen esas
// cabeceras (ej. accediendo directo al .exe o a un único contenedor sin
// nginx delante), se usa la IP de la conexión TCP (r.RemoteAddr).
func getClientIP(r *http.Request) string {
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
