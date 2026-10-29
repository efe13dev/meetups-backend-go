package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        uint64  `json:"id"`
	Email     string  `json:"email"`
	Password  string  `json:"password"`
	Name      string  `json:"name"`
	Biography *string `json:"biography"`
	Avatar    string  `json:"avatar"`
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Email     string  `json:"email"`
		Password  string  `json:"password"`
		Name      string  `json:"name"`
		Biography *string `json:"biography"`
	}

	avatar := "avatar-default.png"
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return NewError(http.StatusBadRequest, err.Error())
		}
		req.Email = r.FormValue("email")
		req.Password = r.FormValue("password")
		req.Name = r.FormValue("name")
		if bio := r.FormValue("biography"); bio != "" {
			req.Biography = &bio
		}
		if r.MultipartForm != nil && len(r.MultipartForm.File["avatar"]) > 0 {
			saved, err := s.processAndSaveImage(r.MultipartForm.File["avatar"][0])
			if err != nil {
				return NewError(http.StatusBadRequest, err.Error())
			}
			avatar = saved
		}
	} else if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return NewError(http.StatusBadRequest, "invalid JSON body")
	}

	if err := validateEmail(req.Email); err != nil {
		return NewError(http.StatusBadRequest, "email, "+err.Error())
	}
	if req.Password == "" {
		return NewError(http.StatusBadRequest, "password, Required")
	}
	if len(req.Name) < 3 || len(req.Name) > 50 {
		return NewError(http.StatusBadRequest, "name, String must contain between 3 and 50 characters")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return err
	}

	// Optimización: INSERT directo sin SELECT previo; capturamos error 1062 (Duplicate entry) de MySQL.
	res, err := s.db.ExecContext(r.Context(),
		"INSERT INTO users (email, password, name, biography, avatar) VALUES (?, ?, ?, ?, ?)",
		req.Email, string(hashed), req.Name, req.Biography, avatar,
	)
	if err != nil {
		var myErr *mysql.MySQLError
		if errors.As(err, &myErr) && myErr.Number == 1062 {
			return NewError(http.StatusConflict, "User already exists")
		}
		return err
	}

	insertID, _ := res.LastInsertId()
	return writeJSON(w, http.StatusCreated, map[string]any{
		"status":  "ok",
		"id":      insertID,
		"email":   req.Email,
		"message": "user created",
	})
}

func (s *Server) loginUser(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return NewError(http.StatusBadRequest, "invalid JSON body")
	}
	if err := validateEmail(req.Email); err != nil {
		return NewError(http.StatusBadRequest, "field:email, "+err.Error())
	}
	if req.Password == "" {
		return NewError(http.StatusBadRequest, "field:password, Required")
	}

	// Solo traemos las 2 columnas necesarias (id, password) en vez de SELECT *
	var uid uint64
	var hash string
	err := s.db.QueryRowContext(r.Context(),
		"SELECT id, password FROM users WHERE email = ?", req.Email,
	).Scan(&uid, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return NewError(http.StatusUnauthorized, "Invalid email or password")
	}
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		return NewError(http.StatusUnauthorized, "Invalid email or password")
	}

	token, err := SignJWT(uid, s.jwtSecret)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"data":   map[string]string{"token": token},
		"id":     uid,
	})
}

func (s *Server) getUsers(w http.ResponseWriter, r *http.Request) error {
	uid := UserIDFromContext(r.Context())
	var u User
	err := s.db.QueryRowContext(r.Context(),
		"SELECT id, email, password, name, biography, avatar FROM users WHERE id = ?", uid,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Name, &u.Biography, &u.Avatar)
	if errors.Is(err, sql.ErrNoRows) {
		return NewError(http.StatusNotFound, "User not found")
	}
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"user":   u,
	})
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	// Optimización: 1 sola query DELETE mirando RowsAffected en vez de SELECT + DELETE
	res, err := s.db.ExecContext(r.Context(), "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return NewError(http.StatusNotFound, fmt.Sprintf("User id:%s not found", id))
	}
	return writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"message": fmt.Sprintf("User id:%s deleted", id),
	})
}

func validateEmail(email string) error {
	if len(email) < 5 || len(email) > 50 {
		return errors.New("Invalid email length")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return errors.New("Invalid email")
	}
	return nil
}
