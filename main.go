package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	mu         sync.RWMutex
	data       map[string]string
	requests   int
	shutdownCh chan struct{}
}

func NewServer() *Server {
	return &Server{
		data:       make(map[string]string),
		shutdownCh: make(chan struct{}),
	}
}

// Handle POST /data
func (s *Server) postDataHandler(w http.ResponseWriter, r *http.Request) {
	var data map[string]string
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range data {
		s.data[k] = v
	}
	s.requests++
}

// Handle GET /data/{key}
func (s *Server) getDataHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/data/"):]

	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[key]
	if !ok {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "%s", value)
}

// Handle GET /stats
func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fmt.Fprintf(w, "Requests: %d, Data size: %d\n", s.requests, len(s.data))
}

// Handle DELETE /data/{key}
func (s *Server) deleteDataHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/data/"):]

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *Server) backgroundWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			fmt.Printf("Requests: %d, Data size: %d\n", s.requests, len(s.data))
			s.mu.RUnlock()
		case <-s.shutdownCh:
			return
		}
	}
}

func main() {
	server := NewServer()
	go server.backgroundWorker()

	http.HandleFunc("/data", server.postDataHandler)
	http.HandleFunc("/data/", server.getDataHandler) // Обработка по конкретному ключу
	http.HandleFunc("/stats", server.statsHandler)
	// http.HandleFunc("/data/", server.deleteDataHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
