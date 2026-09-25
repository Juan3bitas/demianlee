package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// handler para iniciar sesión y/o autenticación

func handleSession(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"isAdmin": isAdminRequest(r)})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Solicitud inválida")
		return
	}

	if body.Password != adminPassword() {
		writeJSONError(w, http.StatusUnauthorized, "Contraseña incorrecta")
		return
	}

	if err := createSession(w); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "No se pudo iniciar sesión")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"isAdmin": true})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	destroySession(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"isAdmin": false})
}

// Info del servidor

func handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"hostname":   getHostname(),
		"serverName": getServerName(),
	})
}

// Listado y búsqueda

func handleListBooks(w http.ResponseWriter, r *http.Request) {
	all, err := loadBooks()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "No se pudo leer el catálogo")
		return
	}

	query := r.URL.Query().Get("query")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	items, totalPages, total := searchBooks(all, query, page)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"books":      items,
		"page":       page,
		"totalPages": totalPages,
		"total":      total,
		"pageSize":   pageSize,
	})
}

func handleGetBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	all, err := loadBooks()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "No se pudo leer el catálogo")
		return
	}

	book, idx := findBookLocked(all, id)
	if idx == -1 {
		writeJSONError(w, http.StatusNotFound, "Libro no encontrado")
		return
	}

	writeJSON(w, http.StatusOK, book)
}

// Handler para subida de portada

var allowedImageExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true,
}

var allowedTextExt = map[string]bool{
	".txt": true,
}

// Función saveUploadedCover: Guarda el archivo del campo "imagen" en static/images/uploads
// y devuelve la ruta pública para guardar en books.json. provided=false si el formulario no
// traía archivo.
func saveUploadedCover(r *http.Request) (path string, provided bool, err error) {
	file, header, err := r.FormFile("imagen")
	if err != nil {
		// No se adjuntó archivo: no es un error, simplemente no se reemplaza.
		return "", false, nil
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedImageExt[ext] {
		return "", true, fmt.Errorf("formato de imagen no soportado: %s", ext)
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	destPath := filepath.Join("static", "images", "uploads", filename)

	dest, err := os.Create(destPath)
	if err != nil {
		return "", true, err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		return "", true, err
	}

	return "/static/images/uploads/" + filename, true, nil
}

// Función saveUploadedText: Guarda el archivo del campo "texto" en static/texts/uploads
// y devuelve la ruta pública para guardar en books.json. provided=false si el formulario no
// traía archivo. El texto se guarda como recurso independiente, igual que la portada.
func saveUploadedText(r *http.Request) (path string, provided bool, err error) {
	file, header, err := r.FormFile("texto")
	if err != nil {
		return "", false, nil
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedTextExt[ext] {
		return "", true, fmt.Errorf("formato de archivo no soportado: %s (solo .txt)", ext)
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	destPath := filepath.Join("static", "texts", "uploads", filename)

	dest, err := os.Create(destPath)
	if err != nil {
		return "", true, err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		return "", true, err
	}

	return "/static/texts/uploads/" + filename, true, nil
}

// Handler para crear libro

func handleCreateBook(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "No se pudo procesar el formulario")
		return
	}

	fieldErrors := map[string]string{}

	titulo := strings.TrimSpace(r.FormValue("titulo"))
	autor := strings.TrimSpace(r.FormValue("autor"))
	anioStr := strings.TrimSpace(r.FormValue("anio"))

	if titulo == "" {
		fieldErrors["titulo"] = "El nombre del libro es obligatorio"
	}
	if autor == "" {
		fieldErrors["autor"] = "El autor del libro es obligatorio"
	}

	anio := 0
	if anioStr == "" {
		fieldErrors["anio"] = "La fecha de publicación es obligatoria"
	} else if n, err := strconv.Atoi(anioStr); err != nil {
		fieldErrors["anio"] = "La fecha de publicación debe ser un número de año válido"
	} else {
		anio = n
	}

	imagePath, provided, err := saveUploadedCover(r)
	if err != nil {
		fieldErrors["imagen"] = err.Error()
	} else if !provided {
		fieldErrors["imagen"] = "Debes seleccionar la portada del libro"
	}

	textPath, textProvided, err := saveUploadedText(r)
	if err != nil {
		fieldErrors["texto"] = err.Error()
	} else if !textProvided {
		fieldErrors["texto"] = "Debes seleccionar el archivo de texto del libro"
	}

	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"fieldErrors": fieldErrors})
		return
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	books, err := loadBooksLocked()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "No se pudo leer el catálogo")
		return
	}

	newBook := Book{
		ID:     nextID(books),
		Titulo: titulo,
		Autor:  autor,
		Anio:   anio,
		Imagen: imagePath,
		Texto:  textPath,
	}
	books = append(books, newBook)

	if err := saveBooksLocked(books); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "No se pudo guardar el libro")
		return
	}

	writeJSON(w, http.StatusCreated, newBook)
}

// Handler para editar libro

// Un campo de texto solo se valida/actualiza si el frontend indica
// explícitamente que el usuario lo "tocó" (touched_<campo>=1). Así se
// puede distinguir "no lo modifiqué" de "lo dejé vacío a propósito".
func handleUpdateBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "No se pudo procesar el formulario")
		return
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	books, err := loadBooksLocked()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "No se pudo leer el catálogo")
		return
	}

	existing, idx := findBookLocked(books, id)
	if idx == -1 {
		writeJSONError(w, http.StatusNotFound, "Libro no encontrado")
		return
	}

	fieldErrors := map[string]string{}
	updated := existing

	if r.FormValue("touched_titulo") == "1" {
		titulo := strings.TrimSpace(r.FormValue("titulo"))
		if titulo == "" {
			fieldErrors["titulo"] = "El nombre del libro es obligatorio"
		} else {
			updated.Titulo = titulo
		}
	}

	if r.FormValue("touched_autor") == "1" {
		autor := strings.TrimSpace(r.FormValue("autor"))
		if autor == "" {
			fieldErrors["autor"] = "El autor del libro es obligatorio"
		} else {
			updated.Autor = autor
		}
	}

	if r.FormValue("touched_anio") == "1" {
		anioStr := strings.TrimSpace(r.FormValue("anio"))
		if anioStr == "" {
			fieldErrors["anio"] = "La fecha de publicación es obligatoria"
		} else if n, err := strconv.Atoi(anioStr); err != nil {
			fieldErrors["anio"] = "La fecha de publicación debe ser un número de año válido"
		} else {
			updated.Anio = n
		}
	}

	imagePath, provided, err := saveUploadedCover(r)
	if err != nil {
		fieldErrors["imagen"] = err.Error()
	} else if provided {
		updated.Imagen = imagePath
	}
	// Si no se proporcionó una nueva imagen, se conserva la actual.

	textPath, textProvided, err := saveUploadedText(r)
	if err != nil {
		fieldErrors["texto"] = err.Error()
	} else if textProvided {
		updated.Texto = textPath
	}
	// Si no se proporcionó un nuevo archivo de texto, se conserva el actual.

	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"fieldErrors": fieldErrors})
		return
	}

	books[idx] = updated

	if err := saveBooksLocked(books); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "No se pudo guardar el libro")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// Handler para eliminar libro

func handleDeleteBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	storeMu.Lock()
	defer storeMu.Unlock()

	books, err := loadBooksLocked()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "No se pudo leer el catálogo")
		return
	}

	_, idx := findBookLocked(books, id)
	if idx == -1 {
		writeJSONError(w, http.StatusNotFound, "Libro no encontrado")
		return
	}

	books = append(books[:idx], books[idx+1:]...)

	if err := saveBooksLocked(books); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "No se pudo guardar el catálogo")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
