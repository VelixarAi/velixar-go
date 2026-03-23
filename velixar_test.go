package velixar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := New("test-key", WithBaseURL(srv.URL))
	return c, srv
}

func TestStore(t *testing.T) {
	c, srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/memory" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing auth header")
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["content"] != "hello world" {
			t.Errorf("unexpected content: %v", body["content"])
		}
		json.NewEncoder(w).Encode(map[string]string{"id": "mem-123"})
	})
	defer srv.Close()

	id, err := c.Store(context.Background(), "hello world", WithTags("test"))
	if err != nil {
		t.Fatal(err)
	}
	if id != "mem-123" {
		t.Errorf("expected mem-123, got %s", id)
	}
}

func TestSearch(t *testing.T) {
	c, srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "hello" {
			t.Errorf("unexpected query: %s", r.URL.Query().Get("q"))
		}
		json.NewEncoder(w).Encode(SearchResult{
			Memories: []Memory{{ID: "m1", Content: "hello world", Score: 0.95}},
			Count:    1,
		})
	})
	defer srv.Close()

	res, err := c.Search(context.Background(), "hello", WithLimit(5))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Memories) != 1 || res.Memories[0].ID != "m1" {
		t.Errorf("unexpected results: %+v", res)
	}
}

func TestHealth(t *testing.T) {
	c, srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(HealthStatus{Status: "healthy", Qdrant: true, Redis: true, Search: true, Version: 2})
	})
	defer srv.Close()

	h, err := c.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != "healthy" || !h.Qdrant {
		t.Errorf("unexpected health: %+v", h)
	}
}

func TestAPIError(t *testing.T) {
	c, srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"unauthorized"}`))
	})
	defer srv.Close()

	_, err := c.Store(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected 401, got %d", apiErr.StatusCode)
	}
}

func TestDelete(t *testing.T) {
	c, srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/memory/mem-456" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	defer srv.Close()

	err := c.Delete(context.Background(), "mem-456")
	if err != nil {
		t.Fatal(err)
	}
}
