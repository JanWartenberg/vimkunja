package vikunja

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

const apiBasePath = "/api/v2"

type paginated[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
}

type Project struct {
	ID          int64  `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"name"`
	Description string `json:"description"`
}

type ProjectView struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	ViewKind string `json:"view_kind"`
}

type Task struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Done     bool   `json:"done"`
	DueDate  string `json:"due_date"`
	Priority int64  `json:"priority"`
	Index    int64  `json:"index"`
	Description string `json:"description"`
	// Identifier is present on some responses; we still can fall back to "#<index>".
	Identifier string `json:"identifier"`
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	full := c.BaseURL + apiBasePath + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}

	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		r = strings.NewReader(string(b))
	}

	req, err := http.NewRequestWithContext(ctx, method, full, r)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Best-effort: include raw body for easier debugging.
		return fmt.Errorf("vikunja api %s %s failed: status=%d body=%s", method, path, resp.StatusCode, string(respBody))
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) ListProjects(ctx context.Context, page, perPage int) ([]Project, int, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", fmt.Sprint(page))
	}
	if perPage > 0 {
		q.Set("per_page", fmt.Sprint(perPage))
	}

	var env paginated[Project]
	if err := c.do(ctx, http.MethodGet, "/projects", q, nil, &env); err != nil {
		return nil, 0, err
	}
	return env.Items, env.TotalPages, nil
}

func (c *Client) ListViews(ctx context.Context, projectID int64, page, perPage int) ([]ProjectView, int, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", fmt.Sprint(page))
	}
	if perPage > 0 {
		q.Set("per_page", fmt.Sprint(perPage))
	}

	var env paginated[ProjectView]
	path := fmt.Sprintf("/projects/%d/views", projectID)
	if err := c.do(ctx, http.MethodGet, path, q, nil, &env); err != nil {
		return nil, 0, err
	}
	return env.Items, env.TotalPages, nil
}

func (c *Client) ListTasksInView(ctx context.Context, projectID, viewID int64, page, perPage int) ([]Task, int, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", fmt.Sprint(page))
	}
	if perPage > 0 {
		q.Set("per_page", fmt.Sprint(perPage))
	}

	var env paginated[Task]
	path := fmt.Sprintf("/projects/%d/views/%d/tasks", projectID, viewID)
	if err := c.do(ctx, http.MethodGet, path, q, nil, &env); err != nil {
		return nil, 0, err
	}
	return env.Items, env.TotalPages, nil
}

func (c *Client) ToggleTaskDone(ctx context.Context, taskID int64, done bool) error {
	return fmt.Errorf("ToggleTaskDone not implemented for REST mode (MVP uses SQLite db file by default)")
}
