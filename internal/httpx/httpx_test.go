package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestError(t *testing.T) {
	rec := httptest.NewRecorder()

	Error(rec, "Invalid input", http.StatusBadRequest)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatal("expected application/json")
	}

	var result map[string]string

	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	if result["error"] != "Invalid input" {
		t.Fatalf("unexpected body: %v", result)
	}
}

func TestReadJSON(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		limit      int64
		wantOK     bool
		wantStatus int
	}{
		{"Valid JSON", `{"title":"Go"}`, 1024, true, 200},
		{"Unknown field", `{"title":"Go","unknown":true}`, 1024, false, 400},
		{"Two objects", `{"title":"Go"}{"title":"Other"}`, 1024, false, 400},
		{"Invalid JSON", `{"title":`, 1024, false, 400},
		{"Too large", `{"title":"Long description"}`, 8, false, 413},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/tasks",
				strings.NewReader(tt.body),
			)

			rec := httptest.NewRecorder()

			var input struct {
				Title string `json:"title"`
			}

			ok := ReadJSON(rec, req, &input, tt.limit)

			if ok != tt.wantOK || rec.Code != tt.wantStatus {
				t.Errorf(
					"got ok=%v status=%d, want ok=%v status=%d",
					ok, rec.Code, tt.wantOK, tt.wantStatus,
				)
			}
		})
	}
}
