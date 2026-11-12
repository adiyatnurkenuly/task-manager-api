package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adiyatnurkenuly/task-manager-api/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

type fakeRepository struct {
	listFunc func(context.Context, int64, listOptions) ([]Task, int64, error)
	getFunc  func(context.Context, int64, int64) (Task, error)
}

func (f *fakeRepository) List(ctx context.Context, userID int64, opts listOptions) ([]Task, int64, error) {
	if f.listFunc == nil {
		return nil, 0, errors.New("unexpected List call")
	}
	return f.listFunc(ctx, userID, opts)
}

func (f *fakeRepository) Get(ctx context.Context, userID, taskID int64) (Task, error) {
	if f.getFunc == nil {
		return Task{}, errors.New("unexpected Get call")
	}
	return f.getFunc(ctx, userID, taskID)
}

func (f *fakeRepository) Create(context.Context, int64, string, string) (Task, error) {
	return Task{}, errors.New("unexpected Create call")
}

func (f *fakeRepository) Update(context.Context, int64, int64, string, string, string) (Task, error) {
	return Task{}, errors.New("unexpected Update call")
}

func (f *fakeRepository) Delete(context.Context, int64, int64) (bool, error) {
	return false, errors.New("unexpected Delete call")
}

func authorizedRequest(t *testing.T, method, url string) *http.Request {
	t.Helper()

	secret := []byte("test-secret-with-at-least-32-bytes")
	claims := jwt.RegisteredClaims{
		Issuer:    "task-manager-api",
		Subject:   "42",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(method, url, nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	return req
}

func TestListWithFakeRepository(t *testing.T) {
	fake := &fakeRepository{
		listFunc: func(_ context.Context, userID int64, opts listOptions) ([]Task, int64, error) {
			if userID != 42 {
				t.Errorf("userID = %d, want 42", userID)
			}
			if opts.Status != "done" || opts.Sort != "oldest" || opts.Page != 2 || opts.Limit != 5 {
				t.Errorf("unexpected options: %+v", opts)
			}
			return []Task{{ID: 6, Title: "Finished", Status: "done"}}, 6, nil
		},
	}

	mux := http.NewServeMux()
	(&Handler{Repo: fake}).Register(mux)
	guarded := (&auth.Handler{Secret: []byte("test-secret-with-at-least-32-bytes")}).RequireAuth(mux)

	req := authorizedRequest(t, http.MethodGet, "/tasks?status=done&sort=oldest&page=2&limit=5")
	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var result struct {
		Tasks      []Task `json:"tasks"`
		Page       int    `json:"page"`
		Limit      int    `json:"limit"`
		Total      int64  `json:"total"`
		TotalPages int64  `json:"total_pages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Page != 2 || result.Limit != 5 || result.Total != 6 || result.TotalPages != 2 || len(result.Tasks) != 1 || result.Tasks[0].ID != 6 {
		t.Fatalf("unexpected response: %+v", result)
	}
}

func TestGetNotFoundWithFakeRepository(t *testing.T) {
	fake := &fakeRepository{
		getFunc: func(_ context.Context, userID, taskID int64) (Task, error) {
			if userID != 42 || taskID != 999 {
				t.Errorf("unexpected ids: user=%d task=%d", userID, taskID)
			}
			return Task{}, sql.ErrNoRows
		},
	}

	mux := http.NewServeMux()
	(&Handler{Repo: fake}).Register(mux)
	guarded := (&auth.Handler{Secret: []byte("test-secret-with-at-least-32-bytes")}).RequireAuth(mux)

	req := authorizedRequest(t, http.MethodGet, "/tasks/999")
	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
}
