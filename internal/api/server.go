package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	db         *sql.DB
	jwtSecret  []byte
	uploadsDir string
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

type HTTPError struct {
	Status  int
	Message string
}

func (e HTTPError) Error() string { return e.Message }

func NewError(status int, msg string) HTTPError {
	return HTTPError{Status: status, Message: msg}
}

func NewServer(db *sql.DB, jwtSecret, uploadsDir string) *Server {
	if uploadsDir == "" {
		uploadsDir = "uploads"
	}
	return &Server{
		db:         db,
		jwtSecret:  []byte(jwtSecret),
		uploadsDir: filepath.Join("uploads", uploadsDir),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Endpoints users
	mux.HandleFunc("POST /users", s.handle(s.createUser))
	mux.HandleFunc("DELETE /users/{id}", s.handle(s.deleteUser))
	mux.HandleFunc("POST /login", s.handle(s.loginUser))
	mux.HandleFunc("GET /users", s.handle(s.requireAuth(s.getUsers)))

	// Endpoints meetups
	mux.HandleFunc("POST /meetups", s.handle(s.requireAuth(s.createMeetup)))
	mux.HandleFunc("DELETE /meetup/{id}", s.handle(s.deleteMeetup))
	mux.HandleFunc("GET /meetups", s.handle(s.getMeetups))
	mux.HandleFunc("GET /meetup/{id}", s.handle(s.getMeetupById))

	// Endpoints inscriptions
	mux.HandleFunc("POST /inscription/{id}", s.handle(s.requireAuth(s.createInscription)))
	mux.HandleFunc("DELETE /inscription/{id}", s.handle(s.requireAuth(s.dropInscription)))

	// Servir imágenes subidas bajo /uploads/
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// Fallback: sirve imágenes de uploads/<UPLOADS_DIR> y ./public en la raíz (como pide el frontend); si no, 404 JSON
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
			if clean != "." && !strings.HasPrefix(clean, "..") {
				for _, dir := range []string{s.uploadsDir, "public"} {
					full := filepath.Join(dir, clean)
					if info, err := os.Stat(full); err == nil && !info.IsDir() {
						http.ServeFile(w, r, full)
						return
					}
				}
				switch strings.ToLower(filepath.Ext(clean)) {
				case ".jpeg", ".jpg", ".png", ".gif", ".webp":
					http.ServeFile(w, r, filepath.Join("public", "default-meetup-image.png"))
					return
				}
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]any{"status": "error", "message": "Not found"})
	})

	return corsMiddleware(mux)
}

// handle envuelve nuestros controladores y centraliza los errores igual que handleError.js.
func (s *Server) handle(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			var httpErr HTTPError
			status := http.StatusInternalServerError
			if errors.As(err, &httpErr) && httpErr.Status != 0 {
				status = httpErr.Status
			}
			log.Printf("ERROR %s %s: %v", r.Method, r.URL.Path, err)
			writeJSON(w, status, map[string]any{
				"status":  "error",
				"message": err.Error(),
			})
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
