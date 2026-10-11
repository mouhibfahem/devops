// Flashcards API: a tiny REST service to revise course topics with flashcards.
package main

import (
	"cmp"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Card is one flashcard: a question on the front, the answer on the back.
type Card struct {
	ID       int    `json:"id"`
	Topic    string `json:"topic"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// Store keeps the cards in memory; the mutex makes it safe for concurrent requests.
type Store struct {
	mu     sync.RWMutex
	cards  map[int]Card
	nextID int
}

func NewStore() *Store {
	s := &Store{cards: map[int]Card{}, nextID: 1}
	s.Add(Card{Topic: "devops", Question: "What does CI stand for?", Answer: "Continuous Integration"})
	s.Add(Card{Topic: "sre", Question: "What is an SLO?", Answer: "A target value for a service level indicator (SLI)"})
	s.Add(Card{Topic: "docker", Question: "Image vs container?", Answer: "A container is a running instance of an image"})
	return s
}

func (s *Store) Add(c Card) Card {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.ID, s.nextID = s.nextID, s.nextID+1
	s.cards[c.ID] = c
	return c
}

// List returns the cards in id order, optionally filtered by topic.
func (s *Store) List(topic string) []Card {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Card{}
	for id := 1; id < s.nextID; id++ {
		if c, ok := s.cards[id]; ok && (topic == "" || strings.EqualFold(c.Topic, topic)) {
			out = append(out, c)
		}
	}
	return out
}

func (s *Store) Get(id int) (Card, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.cards[id]
	return c, ok
}

func (s *Store) Delete(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cards, id)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error encoding JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// lookup finds the card named by the {id} path value, answering 400 or 404 itself when it cannot.
func lookup(s *Store, w http.ResponseWriter, r *http.Request) (Card, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return Card{}, false
	}
	c, ok := s.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "card not found")
	}
	return c, ok
}

func NewRouter(s *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /cards", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.List(r.URL.Query().Get("topic")))
	})
	mux.HandleFunc("POST /cards", func(w http.ResponseWriter, r *http.Request) {
		var c Card
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&c); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
		} else if strings.TrimSpace(c.Question) == "" || strings.TrimSpace(c.Answer) == "" {
			writeError(w, http.StatusBadRequest, "fields 'question' and 'answer' are required")
		} else {
			writeJSON(w, http.StatusCreated, s.Add(c))
		}
	})
	// Quiz mode: a random question, without its answer.
	mux.HandleFunc("GET /cards/random", func(w http.ResponseWriter, r *http.Request) {
		cards := s.List(r.URL.Query().Get("topic"))
		if len(cards) == 0 {
			writeError(w, http.StatusNotFound, "no cards available")
			return
		}
		c := cards[rand.IntN(len(cards))] // #nosec G404 -- picking a quiz card is not security-sensitive
		writeJSON(w, http.StatusOK, map[string]any{"id": c.ID, "topic": c.Topic, "question": c.Question})
	})
	mux.HandleFunc("GET /cards/{id}", func(w http.ResponseWriter, r *http.Request) {
		if c, ok := lookup(s, w, r); ok {
			writeJSON(w, http.StatusOK, c)
		}
	})
	mux.HandleFunc("DELETE /cards/{id}", func(w http.ResponseWriter, r *http.Request) {
		if c, ok := lookup(s, w, r); ok {
			s.Delete(c.ID)
			w.WriteHeader(http.StatusNoContent)
		}
	})
	return mux
}

func main() {
	port := cmp.Or(os.Getenv("PORT"), "8080")
	server := &http.Server{Addr: ":" + port, Handler: withObservability(NewRouter(NewStore())),
		ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("flashcards API listening on :%s", port)
	log.Fatal(server.ListenAndServe())
}
