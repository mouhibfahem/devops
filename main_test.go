package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// do sends one request to a fresh router and returns the recorded response.
func do(t *testing.T, s *Store, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(w, req)
	return w
}

func TestHealth(t *testing.T) {
	w := do(t, NewStore(), http.MethodGet, "/health", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ok"`) {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestListCards(t *testing.T) {
	s := NewStore()
	var cards []Card
	w := do(t, s, http.MethodGet, "/cards", "")
	if err := json.NewDecoder(w.Body).Decode(&cards); err != nil || len(cards) != 3 {
		t.Fatalf("expected 3 seeded cards, got %d (err %v)", len(cards), err)
	}

	w = do(t, s, http.MethodGet, "/cards?topic=DOCKER", "")
	cards = nil
	if err := json.NewDecoder(w.Body).Decode(&cards); err != nil || len(cards) != 1 || cards[0].Topic != "docker" {
		t.Fatalf("topic filter failed: %+v (err %v)", cards, err)
	}
}

func TestCreateCard(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		status int
	}{
		{"valid card", `{"topic":"k8s","question":"What is a Pod?","answer":"The smallest deployable unit"}`, http.StatusCreated},
		{"invalid json", `{not json`, http.StatusBadRequest},
		{"missing answer", `{"question":"Q?"}`, http.StatusBadRequest},
		{"blank question", `{"question":"   ","answer":"A"}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := do(t, NewStore(), http.MethodPost, "/cards", tt.body)
			if w.Code != tt.status {
				t.Fatalf("expected %d, got %d: %s", tt.status, w.Code, w.Body.String())
			}
		})
	}
}

func TestCreatedCardGetsNextID(t *testing.T) {
	s := NewStore()
	var c Card
	w := do(t, s, http.MethodPost, "/cards", `{"question":"Q?","answer":"A"}`)
	if err := json.NewDecoder(w.Body).Decode(&c); err != nil || c.ID != 4 {
		t.Fatalf("expected id 4, got %+v (err %v)", c, err)
	}
	if w := do(t, s, http.MethodGet, "/cards/4", ""); w.Code != http.StatusOK {
		t.Fatalf("created card not retrievable: %d", w.Code)
	}
}

func TestGetCard(t *testing.T) {
	tests := []struct {
		path   string
		status int
	}{
		{"/cards/1", http.StatusOK},
		{"/cards/999", http.StatusNotFound},
		{"/cards/abc", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if w := do(t, NewStore(), http.MethodGet, tt.path, ""); w.Code != tt.status {
				t.Fatalf("expected %d, got %d", tt.status, w.Code)
			}
		})
	}
}

func TestDeleteCard(t *testing.T) {
	s := NewStore()
	if w := do(t, s, http.MethodDelete, "/cards/1", ""); w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if w := do(t, s, http.MethodDelete, "/cards/1", ""); w.Code != http.StatusNotFound {
		t.Fatalf("second delete: expected 404, got %d", w.Code)
	}
	if w := do(t, s, http.MethodDelete, "/cards/x", ""); w.Code != http.StatusBadRequest {
		t.Fatalf("bad id: expected 400, got %d", w.Code)
	}
}

func TestRandomCardHidesAnswer(t *testing.T) {
	w := do(t, NewStore(), http.MethodGet, "/cards/random", "")
	var got map[string]any
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil || w.Code != http.StatusOK {
		t.Fatalf("got %d (err %v)", w.Code, err)
	}
	if _, has := got["answer"]; has || got["question"] == "" {
		t.Fatalf("quiz card must show the question but not the answer: %v", got)
	}
}

func TestRandomCardEmpty(t *testing.T) {
	if w := do(t, NewStore(), http.MethodGet, "/cards/random?topic=none", ""); w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	if w := do(t, NewStore(), http.MethodPut, "/cards", ""); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
