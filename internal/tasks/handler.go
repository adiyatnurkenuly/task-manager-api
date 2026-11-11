package tasks

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/adiyatnurkenuly/task-manager-api/internal/auth"
	"github.com/adiyatnurkenuly/task-manager-api/internal/httpx"
)

type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Handler struct {
	DB *sql.DB
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /tasks", h.List)
	mux.HandleFunc("POST /tasks", h.Create)
	mux.HandleFunc("GET /tasks/{id}", h.Get)
	mux.HandleFunc("PUT /tasks/{id}", h.Update)
	mux.HandleFunc("DELETE /tasks/{id}", h.Delete)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	rows, err := h.DB.QueryContext(
		r.Context(),
		`SELECT id, title, description, status, created_at, updated_at
		 FROM tasks
		 WHERE user_id = $1
		 ORDER BY id DESC`,
		userID,
	)
	if err != nil {
		log.Println(err)
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make([]Task, 0)

	for rows.Next() {
		var task Task

		err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			log.Println(err)
			httpx.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		result = append(result, task)
	}

	if err := rows.Err(); err != nil {
		log.Println(err)
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	id, err := parseTaskID(r)
	if err != nil {
		httpx.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var task Task

	err = h.DB.QueryRowContext(
		r.Context(),
		`SELECT id, title, description, status, created_at, updated_at
         FROM tasks
         WHERE id = $1 AND user_id = $2`,
		id,
		userID,
	).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		httpx.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Println(err)
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if !httpx.ReadJSON(w, r, &input, 1<<20) {
		return
	}

	input.Title = strings.TrimSpace(input.Title)

	if input.Title == "" || utf8.RuneCountInString(input.Title) > 255 {
		httpx.Error(w, "Title must contain 1 to 255 characters", http.StatusBadRequest)
		return
	}

	if len(input.Description) > 5000 {
		httpx.Error(
			w,
			"Description must not exceed 5000 bytes",
			http.StatusBadRequest,
		)
		return
	}

	var task Task

	err := h.DB.QueryRowContext(
		r.Context(),
		`INSERT INTO tasks (user_id, title, description)
		 VALUES ($1, $2, $3)
		 RETURNING id, title, description, status, created_at, updated_at`,
		userID,
		input.Title,
		input.Description,
	).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err != nil {
		log.Println(err)
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	id, err := parseTaskID(r)
	if err != nil {
		httpx.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}

	if !httpx.ReadJSON(w, r, &input, 1<<20) {
		return
	}

	input.Title = strings.TrimSpace(input.Title)

	if input.Title == "" || utf8.RuneCountInString(input.Title) > 255 {
		httpx.Error(w, "Title must contain 1 to 255 characters", http.StatusBadRequest)
		return
	}

	if len(input.Description) > 5000 {
		httpx.Error(
			w,
			"Description must not exceed 5000 bytes",
			http.StatusBadRequest,
		)
		return
	}

	if input.Status != "todo" &&
		input.Status != "in_progress" &&
		input.Status != "done" {
		httpx.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	var task Task

	err = h.DB.QueryRowContext(
		r.Context(),
		`UPDATE tasks
         SET title = $1,
             description = $2,
             status = $3,
             updated_at = NOW()
         WHERE id = $4 AND user_id = $5
         RETURNING id, title, description, status, created_at, updated_at`,
		input.Title,
		input.Description,
		input.Status,
		id,
		userID,
	).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		httpx.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Println(err)
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	id, err := parseTaskID(r)
	if err != nil {
		httpx.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	result, err := h.DB.ExecContext(
		r.Context(),
		"DELETE FROM tasks WHERE id = $1 AND user_id = $2",
		id,
		userID,
	)

	if err != nil {
		log.Println(err)
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		log.Println(err)
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if affected == 0 {
		httpx.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseTaskID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)

	if err != nil || id <= 0 {
		return 0, errors.New("invalid task ID")
	}

	return id, nil
}

func currentUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := auth.UserID(r.Context())

	if !ok {
		httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
		return 0, false
	}

	return userID, true
}
