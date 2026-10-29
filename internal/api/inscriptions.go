package api

import (
	"net/http"
)

func (s *Server) createInscription(w http.ResponseWriter, r *http.Request) error {
	userID := UserIDFromContext(r.Context())
	meetupID := r.PathValue("id")

	if _, err := s.db.ExecContext(r.Context(),
		"INSERT INTO inscriptions (id_user, id_meetup) VALUES (?, ?)",
		userID, meetupID,
	); err != nil {
		return NewError(http.StatusConflict, "User already inscribed")
	}

	return writeJSON(w, http.StatusCreated, map[string]any{
		"status":  "ok",
		"message": "user inscribed",
	})
}

func (s *Server) dropInscription(w http.ResponseWriter, r *http.Request) error {
	userID := UserIDFromContext(r.Context())
	meetupID := r.PathValue("id")

	res, err := s.db.ExecContext(r.Context(),
		"DELETE FROM inscriptions WHERE id_user = ? AND id_meetup = ?",
		userID, meetupID,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return NewError(http.StatusNotFound, "User is not inscribed in this meetup")
	}

	return writeJSON(w, http.StatusCreated, map[string]any{
		"status":  "ok",
		"message": "user unsuscribed",
	})
}
