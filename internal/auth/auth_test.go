package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestRequireAuth(t *testing.T) {
	secret := []byte("test-secret-with-at-least-32-bytes")

	signToken := func(key []byte, expires time.Time) string {
		t.Helper()

		claims := jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   "42",
			ExpiresAt: jwt.NewNumericDate(expires),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		signed, err := token.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}

		return signed
	}

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"No token", "", http.StatusUnauthorized},
		{"Invalid token", "Bearer invalid", http.StatusUnauthorized},
		{
			"Expired token",
			"Bearer " + signToken(secret, time.Now().Add(-time.Hour)),
			http.StatusUnauthorized,
		},
		{
			"Valid token",
			"Bearer " + signToken(secret, time.Now().Add(time.Hour)),
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Secret: secret}

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, ok := UserID(r.Context())
				if !ok || id != 42 {
					t.Error("Incorrect user ID")
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
			req.Header.Set("Authorization", tt.authHeader)

			rec := httptest.NewRecorder()

			h.RequireAuth(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}
