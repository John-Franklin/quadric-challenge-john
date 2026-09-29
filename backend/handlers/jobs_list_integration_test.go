package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"backend/models"
)

func jobIDs(t *testing.T, body map[string]any) []int {
	t.Helper()
	raw, ok := body["jobs"].([]any)
	if !ok {
		t.Fatalf("jobs missing from %v", body)
	}
	ids := make([]int, len(raw))
	for i, j := range raw {
		ids[i] = int(j.(map[string]any)["id"].(float64))
	}
	return ids
}

func TestListJobsPagingAndIDs(t *testing.T) {
	srv := newTestServer(t)
	c, _ := register(t, srv, "pager@example.com")

	const n = models.JobsPageSize + 5
	var created []int
	for range n {
		job := c.expect(http.StatusCreated, "POST", "/api/jobs", map[string]string{"action": "calculate_pi"})
		created = append(created, int(job["id"].(float64)))
	}

	first := c.expect(http.StatusOK, "GET", "/api/jobs", nil)
	firstIDs := jobIDs(t, first)
	if len(firstIDs) != models.JobsPageSize || firstIDs[0] != created[n-1] {
		t.Fatalf("first page ids = %v", firstIDs)
	}
	if first["has_more"] != true || first["total"] != float64(n) {
		t.Fatalf("first page = has_more %v total %v", first["has_more"], first["total"])
	}
	cursor := int(first["next_before"].(float64))
	if cursor != firstIDs[len(firstIDs)-1] {
		t.Fatalf("next_before = %d, want %d", cursor, firstIDs[len(firstIDs)-1])
	}

	// A job created between pages must not shift the next page (as OFFSET would).
	c.expect(http.StatusCreated, "POST", "/api/jobs", map[string]string{"action": "lorem_ipsum"})

	second := c.expect(http.StatusOK, "GET", fmt.Sprintf("/api/jobs?before=%d", cursor), nil)
	secondIDs := jobIDs(t, second)
	if len(secondIDs) != 5 || secondIDs[0] != cursor-1 || secondIDs[4] != created[0] {
		t.Fatalf("second page ids = %v", secondIDs)
	}
	if second["has_more"] != false {
		t.Fatalf("second page has_more = %v", second["has_more"])
	}
	if _, ok := second["next_before"]; ok {
		t.Fatalf("last page has next_before: %v", second["next_before"])
	}

	byIDs := c.expect(http.StatusOK, "GET", fmt.Sprintf("/api/jobs?ids=%d,%d,999999", created[0], created[3]), nil)
	if got := jobIDs(t, byIDs); len(got) != 2 || got[0] != created[3] || got[1] != created[0] {
		t.Fatalf("ids lookup = %v", got)
	}

	c.expect(http.StatusBadRequest, "GET", "/api/jobs?before=0", nil)
	c.expect(http.StatusBadRequest, "GET", "/api/jobs?before=abc", nil)
	c.expect(http.StatusBadRequest, "GET", "/api/jobs?ids=1,x", nil)
	c.expect(http.StatusBadRequest, "GET", "/api/jobs?ids=1&before=5", nil)
}
