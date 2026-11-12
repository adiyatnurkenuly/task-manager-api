package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/adiyatnurkenuly/task-manager-api/internal/auth"
	"github.com/adiyatnurkenuly/task-manager-api/internal/tasks"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must contain at least 32 bytes")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatal("Database connection failed: ", err)
	}
	log.Println("Connected to PostgreSQL")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	authHandler := &auth.Handler{DB: db, Secret: []byte(jwtSecret)}
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)

	taskHandler := &tasks.Handler{
		Repo: &tasks.PostgresRepository{DB: db},
	}
	taskMux := http.NewServeMux()
	taskHandler.Register(taskMux)
	mux.Handle("/tasks", authHandler.RequireAuth(taskMux))
	mux.Handle("/tasks/", authHandler.RequireAuth(taskMux))

	log.Println("Server running on port 8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
