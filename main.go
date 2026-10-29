package main

import (
	"log"
	"meetups-backend-go/internal/api"
	"meetups-backend-go/internal/db"
	"net/http"
	"os"
	"time"
)

func main() {
	db.LoadEnv(".env")

	pool, err := db.Open(true)
	if err != nil {
		log.Fatalf("Error conectando a la base de datos: %v", err)
	}
	defer pool.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	srv := api.NewServer(pool, os.Getenv("JWT_SECRET"), os.Getenv("UPLOADS_DIR"))

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Server listening on http://localhost:%s", port)
	log.Fatal(httpServer.ListenAndServe())
}
