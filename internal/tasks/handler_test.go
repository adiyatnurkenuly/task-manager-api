package tasks

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseTaskID(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"1", true},
		{"42", true},
		{"0", false},
		{"-1", false},
		{"abc", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
			req.SetPathValue("id", tt.input)

			_, err := parseTaskID(req)

			if (err == nil) != tt.valid {
				t.Errorf("unexpected result for ID %q", tt.input)
			}
		})
	}
}

func TestTasksRequireAuth(t *testing.T) {
	h := &Handler{}

	tests := []struct {
		name   string
		method string
		handle http.HandlerFunc
	}{
		{"List", http.MethodGet, h.List},
		{"Create", http.MethodPost, h.Create},
		{"Get", http.MethodGet, h.Get},
		{"Update", http.MethodPut, h.Update},
		{"Delete", http.MethodDelete, h.Delete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/tasks/1", nil)
			rec := httptest.NewRecorder()

			tt.handle(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", rec.Code)
			}
		})
	}
}
