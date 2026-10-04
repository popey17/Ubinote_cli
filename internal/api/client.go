package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) Token() string {
	return c.token
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) Register(ctx context.Context, email, password string) (id, emailOut string, err error) {
	var resp registerResponse
	err = c.do(ctx, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	}, &resp, false)
	if err != nil {
		return "", "", err
	}
	return resp.ID, resp.Email, nil
}

func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	var resp loginResponse
	err := c.do(ctx, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, &resp, false)
	if err != nil {
		return "", err
	}
	return resp.Token, nil
}

func (c *Client) Logout(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/v1/auth/logout", nil, nil, true)
}

func (c *Client) Me(ctx context.Context) (string, error) {
	var resp meResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/me", nil, &resp, true); err != nil {
		return "", err
	}
	return resp.UserID, nil
}

func (c *Client) ListNotes(ctx context.Context) ([]Note, error) {
	var notes []Note
	if err := c.do(ctx, http.MethodGet, "/api/v1/notes", nil, &notes, true); err != nil {
		return nil, err
	}
	if notes == nil {
		notes = []Note{}
	}
	return notes, nil
}

func (c *Client) GetNote(ctx context.Context, id string) (Note, error) {
	var note Note
	err := c.do(ctx, http.MethodGet, "/api/v1/notes/"+id, nil, &note, true)
	return note, err
}

func (c *Client) CreateNote(ctx context.Context, title, body string) (Note, error) {
	var note Note
	err := c.do(ctx, http.MethodPost, "/api/v1/notes", noteRequest{Title: title, Body: body}, &note, true)
	return note, err
}

func (c *Client) UpdateNote(ctx context.Context, id, title, body string) (Note, error) {
	var note Note
	err := c.do(ctx, http.MethodPut, "/api/v1/notes/"+id, noteRequest{Title: title, Body: body}, &note, true)
	return note, err
}

func (c *Client) DeleteNote(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/notes/"+id, nil, nil, true)
}

func (c *Client) do(ctx context.Context, method, path string, body any, dest any, auth bool) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth && c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%w: %s", ErrUnauthorized, strings.TrimSpace(string(raw)))
	}
	if res.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s", ErrNotFound, strings.TrimSpace(string(raw)))
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = res.Status
		}
		return fmt.Errorf("api %s: %s", res.Status, msg)
	}

	if dest == nil || res.StatusCode == http.StatusNoContent || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, dest)
}
