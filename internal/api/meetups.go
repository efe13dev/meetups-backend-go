package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var validCategories = map[string]bool{
	"Fotografia": true, "Ciencia": true, "Tecnologia": true, "Gastronomia": true,
	"Naturaleza": true, "Literatura": true, "Musica": true, "Salud": true,
	"Arte": true, "Cultura": true, "Otras": true,
}

var validCities = map[string]bool{
	"Andalucia": true, "Aragon": true, "Asturias": true, "Cantabria": true,
	"Castilla_León": true, "Castilla_La_Mancha": true, "Cataluña": true, "Madrid": true,
	"Valencia": true, "Extremadura": true, "Galicia": true, "Baleares": true,
	"Canarias": true, "Rioja": true, "Murcia": true, "Navarra": true, "Pais_Vasco": true,
}

type Meetup struct {
	ID                uint64        `json:"id"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	Photo             string        `json:"photo"`
	Category          string        `json:"category"`
	City              string        `json:"city"`
	Date              time.Time     `json:"date"`
	UserID            uint64        `json:"user_id"`
	TotalInscriptions int            `json:"total_inscriptions"`
	Inscriptions      *[]Inscription `json:"inscriptions,omitempty"`
}

type Inscription struct {
	IDUser    uint64  `json:"id_user"`
	IDMeetup  uint64  `json:"id_meetup"`
	ID        uint64  `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	Biography *string `json:"biography"`
	Avatar    string  `json:"avatar"`
}

func (s *Server) getMeetups(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT m.id, m.title, m.description, m.photo, m.category, m.city, m.date, m.user_id, COUNT(i.id_user) AS total_inscriptions
		FROM meetups AS m
		LEFT JOIN inscriptions AS i ON m.id = i.id_meetup
		WHERE m.date >= CURDATE()
		GROUP BY m.id
		ORDER BY m.date ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	meetups := []Meetup{}
	for rows.Next() {
		var m Meetup
		if err := rows.Scan(&m.ID, &m.Title, &m.Description, &m.Photo, &m.Category, &m.City, &m.Date, &m.UserID, &m.TotalInscriptions); err != nil {
			return err
		}
		meetups = append(meetups, m)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"meetups": meetups,
	})
}

func (s *Server) getMeetupById(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	ctx := r.Context()

	var (
		m            Meetup
		inscriptions = []Inscription{}
		errMeetup    error
		errInsc      error
		wg           sync.WaitGroup
	)

	// Ejecutamos las 2 consultas a MySQL en paralelo con goroutines
	wg.Add(2)
	go func() {
		defer wg.Done()
		errMeetup = s.db.QueryRowContext(ctx, `
			SELECT m.id, m.title, m.description, m.photo, m.category, m.city, m.date, m.user_id, COUNT(i.id_user) AS total_inscriptions
			FROM meetups AS m
			LEFT JOIN inscriptions AS i ON m.id = i.id_meetup
			WHERE m.id = ?
			GROUP BY m.id`, id,
		).Scan(&m.ID, &m.Title, &m.Description, &m.Photo, &m.Category, &m.City, &m.Date, &m.UserID, &m.TotalInscriptions)
	}()

	go func() {
		defer wg.Done()
		rows, err := s.db.QueryContext(ctx, `
			SELECT i.id_user, i.id_meetup, u.id, u.email, u.name, u.biography, u.avatar
			FROM inscriptions AS i
			INNER JOIN users AS u ON i.id_user = u.id
			WHERE i.id_meetup = ?`, id)
		if err != nil {
			errInsc = err
			return
		}
		defer rows.Close()
		for rows.Next() {
			var ins Inscription
			if err := rows.Scan(&ins.IDUser, &ins.IDMeetup, &ins.ID, &ins.Email, &ins.Name, &ins.Biography, &ins.Avatar); err != nil {
				errInsc = err
				return
			}
			inscriptions = append(inscriptions, ins)
		}
		errInsc = rows.Err()
	}()
	wg.Wait()

	if errors.Is(errMeetup, sql.ErrNoRows) {
		return NewError(http.StatusNotFound, fmt.Sprintf("Meetup id:%s not found", id))
	}
	if errMeetup != nil {
		return errMeetup
	}
	if errInsc != nil {
		return errInsc
	}

	m.Inscriptions = &inscriptions
	return writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"meetup": m,
	})
}

func (s *Server) createMeetup(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Category    string `json:"category"`
		City        string `json:"city"`
		Date        string `json:"date"`
	}

	photo := "default-meetup-image.png"
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return NewError(http.StatusBadRequest, err.Error())
		}
		req.Title = r.FormValue("title")
		req.Description = r.FormValue("description")
		req.Category = r.FormValue("category")
		req.City = r.FormValue("city")
		req.Date = r.FormValue("date")
		if r.MultipartForm != nil && len(r.MultipartForm.File["photo"]) > 0 {
			saved, err := s.processAndSaveImage(r.MultipartForm.File["photo"][0])
			if err != nil {
				return NewError(http.StatusBadRequest, err.Error())
			}
			photo = saved
		}
	} else if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return NewError(http.StatusBadRequest, "invalid JSON body")
	}

	if len(req.Title) < 3 || len(req.Title) > 100 {
		return NewError(http.StatusBadRequest, "field:title, String must contain between 3 and 100 characters")
	}
	if len(req.Description) < 5 || len(req.Description) > 2000 {
		return NewError(http.StatusBadRequest, "field:description, String must contain between 5 and 2000 characters")
	}
	if !validCategories[req.Category] {
		return NewError(http.StatusBadRequest, "field:category, Invalid enum value")
	}
	if !validCities[req.City] {
		return NewError(http.StatusBadRequest, "field:city, Invalid enum value")
	}
	parsedDate, err := parseMeetupDate(req.Date)
	if err != nil {
		return NewError(http.StatusBadRequest, "field:date, Invalid datetime")
	}

	userID := UserIDFromContext(r.Context())
	res, err := s.db.ExecContext(r.Context(),
		"INSERT INTO meetups (title, description, photo, category, city, date, user_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		req.Title, req.Description, photo, req.Category, req.City, parsedDate, userID,
	)
	if err != nil {
		return err
	}
	meetupID, _ := res.LastInsertId()

	return writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"data": map[string]any{
			"id":          meetupID,
			"title":       req.Title,
			"description": req.Description,
			"photo":       photo,
			"category":    req.Category,
			"city":        req.City,
			"date":        req.Date,
			"userId":      userID,
		},
	})
}

func (s *Server) deleteMeetup(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	res, err := s.db.ExecContext(r.Context(), "DELETE FROM meetups WHERE id = ?", id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return NewError(http.StatusNotFound, fmt.Sprintf("Meetup id:%s not found", id))
	}
	return writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"message": fmt.Sprintf("Meetup id:%s deleted", id),
	})
}

func parseMeetupDate(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date")
}
