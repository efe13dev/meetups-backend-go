package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type ctxKey struct{}

var jwtHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

type jwtClaims struct {
	ID  uint64 `json:"id"`
	IAT int64  `json:"iat"`
	EXP int64  `json:"exp"`
}

// SignJWT firma un token HS256 idéntico al que genera jsonwebtoken en Node.js (40 días de validez).
func SignJWT(userID uint64, secret []byte) (string, error) {
	now := time.Now().Unix()
	claimsJSON, err := json.Marshal(jwtClaims{
		ID:  userID,
		IAT: now,
		EXP: now + 40*24*3600,
	})
	if err != nil {
		return "", err
	}
	unsigned := jwtHeader + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := signHMAC(unsigned, secret)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// VerifyJWT valida la firma HS256 y la fecha de expiración y devuelve el id del usuario.
func VerifyJWT(token string, secret []byte) (uint64, error) {
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	if token == "" {
		return 0, errors.New("Token not found")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, errors.New("jwt malformed")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, signHMAC(parts[0]+"."+parts[1], secret)) {
		return 0, errors.New("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, errors.New("invalid token payload")
	}
	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, err
	}
	if claims.EXP > 0 && time.Now().Unix() > claims.EXP {
		return 0, errors.New("jwt expired")
	}
	return claims.ID, nil
}

func signHMAC(unsigned string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(unsigned))
	return mac.Sum(nil)
}

func (s *Server) requireAuth(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		uid, err := VerifyJWT(r.Header.Get("Authorization"), s.jwtSecret)
		if err != nil {
			return NewError(http.StatusUnauthorized, err.Error())
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, uid)
		return next(w, r.WithContext(ctx))
	}
}

func UserIDFromContext(ctx context.Context) uint64 {
	uid, _ := ctx.Value(ctxKey{}).(uint64)
	return uid
}
