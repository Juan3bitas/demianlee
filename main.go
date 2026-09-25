package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
)

var tmpl *template.Template

// Clase PageData: Es lo que recibe la plantilla index.html para el primer
// renderizado. El resto de la navegación la maneja el JavaScript del
// frontend consumiendo la API /api/*.
type PageData struct {
	AppName    string
	Host       string
	ServerName string
	InitialJS  template.JS
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	initial := map[string]interface{}{
		"hostname":   getHostname(),
		"serverName": getServerName(),
		"isAdmin":    isAdminRequest(r),
	}
	initialJSON, err := json.Marshal(initial)
	if err != nil {
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	data := PageData{
		AppName:    "Debian Lee",
		Host:       getHostname(),
		ServerName: getServerName(),
		InitialJS:  template.JS(initialJSON),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("error ejecutando plantilla: %v", err)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"host":   getHostname(),
	})
}

func main() {
	var err error
	tmpl, err = template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatalf("error cargando plantillas: %v", err)
	}

	// Asegura que exista la carpeta de portadas subidas por el administrador.
	if err := os.MkdirAll("static/images/uploads", 0755); err != nil {
		log.Fatalf("error creando carpeta de subidas: %v", err)
	}

	// Asegura que exista la carpeta de archivos de texto subidos por el administrador.
	if err := os.MkdirAll("static/texts/uploads", 0755); err != nil {
		log.Fatalf("error creando carpeta de subidas de texto: %v", err)
	}

	mux := http.NewServeMux()

	// Páginas
	mux.HandleFunc("GET /{$}", indexHandler)
	mux.HandleFunc("GET /healthz", healthHandler)

	// API de sesión / autenticación de administrador
	mux.HandleFunc("GET /api/session", handleSession)
	mux.HandleFunc("POST /api/login", handleLogin)
	mux.HandleFunc("POST /api/logout", handleLogout)

	// API de información del servidor
	mux.HandleFunc("GET /api/info", handleInfo)

	// API de libros
	mux.HandleFunc("GET /api/books", handleListBooks)
	mux.HandleFunc("GET /api/books/{id}", handleGetBook)
	mux.HandleFunc("POST /api/books", requireAdmin(handleCreateBook))
	mux.HandleFunc("PUT /api/books/{id}", requireAdmin(handleUpdateBook))
	mux.HandleFunc("DELETE /api/books/{id}", requireAdmin(handleDeleteBook))

	// Recursos estáticos (CSS, JS, portadas) como archivos independientes del HTML.
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("Debian Lee escuchando en %s (host: %s, servidor: %s)", addr, getHostname(), getServerName())
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("error iniciando el servidor: %v", err)
	}
}
