package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// LoadEnv carga variables clave=valor desde un archivo .env usando solo la librería estándar.
func LoadEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok && os.Getenv(k) == "" {
			os.Setenv(strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"'`))
		}
	}
}

// Open crea y verifica un pool de conexiones a MySQL.
// Si includeDB es false, conecta al servidor sin seleccionar base de datos (útil para initDb).
func Open(includeDB bool) (*sql.DB, error) {
	host := envOr("DATABASE_HOST", "127.0.0.1")
	port := envOr("DATABASE_PORT", "3306")
	user := os.Getenv("DATABASE_USER")
	pass := os.Getenv("DATABASE_PASSWORD")
	name := ""
	if includeDB {
		name = os.Getenv("DATABASE_NAME")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local&multiStatements=true&interpolateParams=true",
		user, pass, host, port, name)

	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	pool.SetMaxOpenConns(25)
	pool.SetMaxIdleConns(10)
	pool.SetConnMaxLifetime(5 * time.Minute)

	if err := pool.Ping(); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
