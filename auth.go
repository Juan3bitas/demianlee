package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"sync"
	"time"
)

// Autenticación de administrador muy simple, pensada para un proyecto
// académico: una sola contraseña de administrador (configurable por
// variable de entorno) y un token de sesión guardado en memoria.

const sessionCookieName = "demian_lee_session"

var (
	sessionsMu sync.Mutex
	sessions   = map[string]time.Time{}
	sessionTTL = 12 * time.Hour
)

func adminPassword() string {
	if p := os.Getenv("ADMIN_PASSWORD"); p != "" {
		return p
	}
	return "admin123"
}

func newSessionToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func createSession(w http.ResponseWriter) error {
	token, err := newSessionToken()
	if err != nil {
		return err
	}

	sessionsMu.Lock()
	sessions[token] = time.Now().Add(sessionTTL)
	sessionsMu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(sessionTTL.Seconds()),
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func destroySession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		sessionsMu.Lock()
		delete(sessions, c.Value)
		sessionsMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}

func isAdminRequest(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}

	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	expiry, ok := sessions[c.Value]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		delete(sessions, c.Value)
		return false
	}
	return true
}

// Función requireAdmin: Es un middleware que corta la petición con 401 si
// quien llama no tiene una sesión de administrador activa.
func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdminRequest(r) {
			writeJSONError(w, http.StatusUnauthorized, "Se requiere sesión de administrador")
			return
		}
		next(w, r)
	}
}
