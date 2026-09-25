package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Clase Book: representa cada libro
type Book struct {
	ID     string `json:"id"`
	Titulo string `json:"titulo"`
	Autor  string `json:"autor"`
	Anio   int    `json:"anio"`
	Imagen string `json:"imagen"`
	Texto  string `json:"texto"` // Acá está almacenado el texto del libro
}

const booksFilePath = "data/books.json"
const pageSize = 12 // Tamaño máximo de la página que almacena los libros en la pantalla

// bookStore protege el acceso concurrente al archivo JSON de libros.
// Como el archivo también puede editarse a mano con un editor de texto,
// releemos de disco en cada operación de lectura en lugar de mantener
// todo únicamente en memoria.
var storeMu sync.Mutex

func loadBooks() ([]Book, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	return loadBooksLocked()
}

// Función loadBooksLocked: Asume que storeMu ya está tomado por el llamador.
func loadBooksLocked() ([]Book, error) {
	data, err := os.ReadFile(booksFilePath)
	if err != nil {
		return nil, err
	}
	var books []Book
	if err := json.Unmarshal(data, &books); err != nil {
		return nil, err
	}
	return books, nil
}

func saveBooksLocked(books []Book) error {
	data, err := json.MarshalIndent(books, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(booksFilePath, data, 0644)
}

// Función nextID: Calcula un id numérico incremental a partir del máximo existente.
func nextID(books []Book) string {
	max := 0
	for _, b := range books {
		if n, err := strconv.Atoi(b.ID); err == nil && n > max {
			max = n
		}
	}
	return strconv.Itoa(max + 1)
}

// Función matches: Indica si un libro coincide con el texto de búsqueda.
// Compara título y autor, sin distinguir mayúsculas/minúsculas.
func matches(b Book, query string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(strings.TrimSpace(query))
	return strings.Contains(strings.ToLower(b.Titulo), q) ||
		strings.Contains(strings.ToLower(b.Autor), q)
}

// Función searchBooks: Filtra y pagina el catálogo. page es 1-indexado.
func searchBooks(all []Book, query string, page int) (items []Book, totalPages int, total int) {
	var filtered []Book
	for _, b := range all {
		if matches(b, query) {
			filtered = append(filtered, b)
		}
	}

	total = len(filtered)
	totalPages = (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	items = filtered[start:end]
	if items == nil {
		items = []Book{}
	}
	return items, totalPages, total
}

func findBookLocked(books []Book, id string) (Book, int) {
	for i, b := range books {
		if b.ID == id {
			return b, i
		}
	}
	return Book{}, -1
}

var errNotFound = fmt.Errorf("libro no encontrado")
