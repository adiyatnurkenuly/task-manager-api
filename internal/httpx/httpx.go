package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func Error(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{
		Error: message,
	})
}

func ReadJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		jsonDecodeError(w, err)
		return false
	}

	var extra any

	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			Error(w, "Only one JSON value is allowed", http.StatusBadRequest)
		} else {
			jsonDecodeError(w, err)
		}
		return false
	}

	return true
}

func jsonDecodeError(w http.ResponseWriter, err error) {
	var sizeErr *http.MaxBytesError

	if errors.As(err, &sizeErr) {
		Error(w, "Request body is too large", http.StatusRequestEntityTooLarge)
		return
	}

	Error(w, "Invalid JSON", http.StatusBadRequest)
}
