package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSignAndVerifyJWT(t *testing.T) {
	secret := []byte("super-secret")
	tok, err := SignJWT(42, secret)
	if err != nil {
		t.Fatalf("SignJWT failed: %v", err)
	}
	uid, err := VerifyJWT(tok, secret)
	if err != nil || uid != 42 {
		t.Fatalf("VerifyJWT got (%d, %v), want (42, nil)", uid, err)
	}
	if _, err := VerifyJWT(tok, []byte("wrong-secret")); err == nil {
		t.Fatal("expected error with wrong secret")
	}
}

func TestRoutesAndValidation(t *testing.T) {
	srv := NewServer(nil, "secret", "uploads")
	h := srv.Handler()

	// 1. Ruta inexistente -> 404 JSON
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/no-existe", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"Not found"`) {
		t.Fatalf("404 handler got %d %s", rec.Code, rec.Body.String())
	}

	// 2. Ruta protegida sin token -> 401 Token not found
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/users", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), `"Token not found"`) {
		t.Fatalf("auth middleware got %d %s", rec.Code, rec.Body.String())
	}

	// 3. Validación de POST /meetups con token válido pero categoría inválida -> 400
	tok, _ := SignJWT(1, []byte("secret"))
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/meetups", strings.NewReader(`{"title":"Hola","description":"Descripcion valida","category":"Invalida","city":"Madrid","date":"2026-10-01T15:30:00Z"}`))
	req.Header.Set("Authorization", tok)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "field:category") {
		t.Fatalf("validation got %d %s", rec.Code, rec.Body.String())
	}
}
