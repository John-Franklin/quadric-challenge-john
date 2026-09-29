package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"backend/db"
	"backend/sessions"
)

const testRunnerToken = "test-runner-token"

// newTestServer runs the API against a throwaway Postgres schema, so it never touches real data.
// Requires TEST_DATABASE_URL (e.g. the compose.yml DSN) and Redis at REDIS_ADDR or localhost:6379.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	baseDSN := os.Getenv("TEST_DATABASE_URL")
	if baseDSN == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()

	admin, err := sql.Open("pgx", baseDSN)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("auth_test_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
		admin.Close()
	})

	u, err := url.Parse(baseDSN)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	t.Setenv("DATABASE_URL", u.String())

	conn, err := db.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(ctx, conn); err != nil {
		t.Fatal(err)
	}
	store, err := sessions.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	h := &Handlers{DB: conn, Sessions: store, RunnerToken: testRunnerToken, AdminEmail: "admin@example.com"}
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)
	return srv
}

type apiClient struct {
	t    *testing.T
	base string
	http *http.Client
}

func newClient(t *testing.T, srv *httptest.Server) *apiClient {
	jar, _ := cookiejar.New(nil)
	return &apiClient{t: t, base: srv.URL, http: &http.Client{Jar: jar}}
}

// do sends a request and returns the status code and decoded JSON body (if any).
func (c *apiClient) do(method, path string, body any, headers ...string) (int, map[string]any) {
	c.t.Helper()
	var buf []byte
	if body != nil {
		buf, _ = json.Marshal(body)
	}
	req, err := http.NewRequestWithContext(c.t.Context(), method, c.base+path, bytes.NewReader(buf))
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	// 204 responses have no body to decode.
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func (c *apiClient) expect(want int, method, path string, body any, headers ...string) map[string]any {
	c.t.Helper()
	got, out := c.do(method, path, body, headers...)
	if got != want {
		c.t.Fatalf("%s %s: status %d, want %d (body %v)", method, path, got, want, out)
	}
	return out
}

func register(t *testing.T, srv *httptest.Server, email string) (*apiClient, map[string]any) {
	c := newClient(t, srv)
	user := c.expect(http.StatusCreated, "POST", "/api/auth/register",
		map[string]string{"email": email, "password": "correct-horse"})
	return c, user
}

func TestAuthAndRoles(t *testing.T) {
	srv := newTestServer(t)
	anon := newClient(t, srv)
	job := map[string]string{"action": "calculate_pi"}

	anon.expect(http.StatusUnauthorized, "GET", "/api/jobs", nil)
	anon.expect(http.StatusUnauthorized, "POST", "/api/jobs", job)
	anon.expect(http.StatusBadRequest, "POST", "/api/auth/register", map[string]string{"email": "x@example.com", "password": "short"})
	anon.expect(http.StatusBadRequest, "POST", "/api/auth/register", map[string]string{"email": "not-an-email", "password": "correct-horse"})

	admin, adminUser := register(t, srv, "Admin@Example.com")
	if adminUser["is_admin"] != true {
		t.Fatalf("admin user = %v", adminUser)
	}
	alice, aliceUser := register(t, srv, "alice@example.com")
	if aliceUser["is_admin"] != false {
		t.Fatalf("alice = %v", aliceUser)
	}
	bob, _ := register(t, srv, "bob@example.com")
	anon.expect(http.StatusConflict, "POST", "/api/auth/register", map[string]string{"email": "alice@example.com", "password": "correct-horse"})
	anon.expect(http.StatusUnauthorized, "POST", "/api/auth/login", map[string]string{"email": "alice@example.com", "password": "wrong-password"})
	anon.expect(http.StatusUnauthorized, "POST", "/api/auth/login", map[string]string{"email": "nobody@example.com", "password": "correct-horse"})

	// Users have no active job limit.
	var aliceJobID float64
	for range 4 {
		aliceJobID = alice.expect(http.StatusCreated, "POST", "/api/jobs", job)["id"].(float64)
	}

	// Everyone sees every job, but only the owner (or an admin) can cancel it.
	listed := bob.expect(http.StatusOK, "GET", "/api/jobs", nil)
	if listed["total"] != float64(4) || listed["jobs"].([]any)[0].(map[string]any)["owner_email"] != "alice@example.com" {
		t.Fatalf("bob list = %v", listed)
	}
	jobPath := fmt.Sprintf("/api/jobs/%d", int(aliceJobID))
	bob.expect(http.StatusOK, "GET", jobPath, nil)
	bob.expect(http.StatusForbidden, "POST", jobPath+"/cancel", nil)
	bob.expect(http.StatusNotFound, "POST", "/api/jobs/999999/cancel", nil)

	alice.expect(http.StatusOK, "POST", jobPath+"/cancel", nil)
	alice.expect(http.StatusConflict, "POST", jobPath+"/cancel", nil)
	firstJobPath := fmt.Sprintf("/api/jobs/%d", int(aliceJobID)-3)
	admin.expect(http.StatusOK, "POST", firstJobPath+"/cancel", nil)

	// Admin endpoints.
	alice.expect(http.StatusForbidden, "GET", "/api/admin/users", nil)
	aliceID := int(aliceUser["id"].(float64))
	admin.expect(http.StatusBadRequest, "PATCH", fmt.Sprintf("/api/admin/users/%d", aliceID), map[string]string{})
	admin.expect(http.StatusOK, "PATCH", fmt.Sprintf("/api/admin/users/%d", aliceID), map[string]bool{"is_admin": true})
	if me := alice.expect(http.StatusOK, "GET", "/api/auth/me", nil); me["is_admin"] != true {
		t.Fatalf("alice after promotion = %v", me)
	}
	adminID := int(adminUser["id"].(float64))
	admin.expect(http.StatusBadRequest, "PATCH", fmt.Sprintf("/api/admin/users/%d", adminID), map[string]bool{"is_admin": false})

	// Logout ends the session.
	alice.expect(http.StatusNoContent, "POST", "/api/auth/logout", nil)
	alice.expect(http.StatusUnauthorized, "GET", "/api/auth/me", nil)
}

func TestRunnerToken(t *testing.T) {
	srv := newTestServer(t)
	c := newClient(t, srv)

	c.expect(http.StatusUnauthorized, "POST", "/api/runner", map[string]string{"id": "r1"})
	c.expect(http.StatusUnauthorized, "GET", "/api/jobs/acquire", nil, "X-Runner-ID", "r1", "X-Runner-Token", "wrong")
	c.expect(http.StatusOK, "POST", "/api/runner", map[string]string{"id": "r1"}, "X-Runner-Token", testRunnerToken)
	c.expect(http.StatusNoContent, "GET", "/api/jobs/acquire", nil, "X-Runner-ID", "r1", "X-Runner-Token", testRunnerToken)
}
