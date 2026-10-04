package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ubinote-cli/internal/api"
)

func TestLoginAndAuthHeader(t *testing.T) {
	var sawAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "tok123"})
		case "/api/v1/notes":
			sawAuth = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode([]any{})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := api.New(srv.URL, "")
	tok, err := c.Login(context.Background(), "a@b.c", "secret")
	if err != nil || tok != "tok123" {
		t.Fatalf("login: %q %v", tok, err)
	}
	c.SetToken(tok)
	if _, err := c.ListNotes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sawAuth != "Bearer tok123" {
		t.Fatalf("auth %q", sawAuth)
	}
}

func TestUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := api.New(srv.URL, "bad")
	_, err := c.ListNotes(context.Background())
	if err == nil || !errors.Is(err, api.ErrUnauthorized) {
		t.Fatalf("err=%v", err)
	}
}

func TestCreateNoteAndNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/notes":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         "n1",
				"user_id":    "u1",
				"title":      "T",
				"body":       "B",
				"created_at": "2026-10-04T00:00:00Z",
				"updated_at": "2026-10-04T00:00:00Z",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/notes/missing":
			http.Error(w, "note not found", http.StatusNotFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := api.New(srv.URL, "tok")
	note, err := c.CreateNote(context.Background(), "T", "B")
	if err != nil {
		t.Fatal(err)
	}
	if note.ID != "n1" || note.Title != "T" {
		t.Fatalf("note %+v", note)
	}
	_, err = c.GetNote(context.Background(), "missing")
	if err == nil || !errors.Is(err, api.ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}
