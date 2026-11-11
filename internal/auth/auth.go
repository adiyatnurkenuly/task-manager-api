package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/adiyatnurkenuly/task-manager-api/internal/httpx"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const tokenIssuer = "task-manager-api"

type Handler struct {
	DB     *sql.DB
	Secret []byte
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type contextKey struct{}

func UserID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(contextKey{}).(int64)
	return id, ok && id > 0
}

func readCredentials(w http.ResponseWriter, r *http.Request) (credentials, error) {
	var input credentials

	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		return input, err
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return input, errors.New("invalid JSON")
	}

	return input, nil
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	input, err := readCredentials(w, r)
	if err != nil {
		httpx.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	address, err := mail.ParseAddress(email)

	if err != nil || address.Address != email || len(email) > 255 {
		httpx.Error(w, "Invalid email", http.StatusBadRequest)
		return
	}

	if utf8.RuneCountInString(input.Password) < 8 || len(input.Password) > 72 {
		httpx.Error(w, "Password must be at least 8 characters and at most 72 bytes",
			http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		log.Println(err)
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var id int64

	err = h.DB.QueryRowContext(
		r.Context(),
		`INSERT INTO users (email, password_hash)
		 VALUES ($1, $2)
		 ON CONFLICT (email) DO NOTHING
		 RETURNING id`,
		email,
		string(hash),
	).Scan(&id)

	if errors.Is(err, sql.ErrNoRows) {
		httpx.Error(w, "Email already registered", http.StatusConflict)
		return
	}

	if err != nil {
		log.Println(err)
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]any{
		"id":    id,
		"email": email,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	input, err := readCredentials(w, r)
	if err != nil {
		httpx.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))

	var id int64
	var passwordHash string

	err = h.DB.QueryRowContext(
		r.Context(),
		"SELECT id, password_hash FROM users WHERE email = $1",
		email,
	).Scan(&id, &passwordHash)

	if errors.Is(err, sql.ErrNoRows) {
		httpx.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if err != nil {
		log.Println(err)
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(input.Password) > 72 ||
		bcrypt.CompareHashAndPassword(
			[]byte(passwordHash),
			[]byte(input.Password),
		) != nil {
		httpx.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	now := time.Now()

	claims := jwt.RegisteredClaims{
		Issuer:    tokenIssuer,
		Subject:   strconv.FormatInt(id, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(h.Secret)
	if err != nil {
		log.Println(err)
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]any{
		"token":      tokenString,
		"token_type": "Bearer",
		"expires_in": 86400,
	})
}

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))

		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims := &jwt.RegisteredClaims{}

		token, err := jwt.ParseWithClaims(
			parts[1],
			claims,
			func(token *jwt.Token) (any, error) {
				return h.Secret, nil
			},
			jwt.WithValidMethods([]string{"HS256"}),
			jwt.WithIssuer(tokenIssuer),
			jwt.WithExpirationRequired(),
		)

		if err != nil || !token.Valid {
			httpx.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil || userID <= 0 {
			httpx.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
