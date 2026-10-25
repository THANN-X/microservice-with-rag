package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func get(t *testing.T, app *fiber.App, path string) (int, map[string]any) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", path, nil))
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return resp.StatusCode, body
}

func TestReadyz(t *testing.T) {
	healthy := func(context.Context) error { return nil }
	broken := func(context.Context) error { return errors.New("dial tcp db:5432: connection refused") }

	tests := []struct {
		name     string
		checks   map[string]Check
		shutdown bool
		wantCode int
		wantBody string
	}{
		{"all dependencies up", map[string]Check{"postgres": healthy}, false, 200, "ok"},
		{"one dependency down", map[string]Check{"postgres": healthy, "mongo": broken}, false, 503, "unavailable"},
		{"shutting down", map[string]Check{"postgres": healthy}, true, 503, "shutting_down"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New("test")
			for name, check := range tt.checks {
				c.Add(name, check)
			}
			if tt.shutdown {
				c.MarkShuttingDown()
			}
			app := fiber.New()
			c.Register(app)

			code, body := get(t, app, "/readyz")
			if code != tt.wantCode || body["status"] != tt.wantBody {
				t.Errorf("GET /readyz = %d %v, want %d status=%s", code, body, tt.wantCode, tt.wantBody)
			}
		})
	}
}

func TestReadyz_DoesNotLeakErrorDetails(t *testing.T) {
	c := New("test").Add("postgres", func(context.Context) error {
		return errors.New("password authentication failed for user admin")
	})
	app := fiber.New()
	c.Register(app)

	_, body := get(t, app, "/readyz")
	checks, _ := body["checks"].(map[string]any)
	if checks["postgres"] != "unavailable" {
		t.Errorf("checks.postgres = %v, want generic \"unavailable\"", checks["postgres"])
	}
}

func TestHealthz_IgnoresDependencies(t *testing.T) {
	c := New("test").Add("postgres", func(context.Context) error { return errors.New("down") })
	app := fiber.New()
	c.Register(app)

	if code, _ := get(t, app, "/healthz"); code != 200 {
		t.Errorf("GET /healthz = %d, want 200 even when a dependency is down", code)
	}
}
