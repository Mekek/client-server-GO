package api

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"sync"

	"golang.org/x/time/rate"
)

type Stats struct {
	Total       int            `json:"total_requests"`
	Successful  int            `json:"successful_requests"`
	Failed      int            `json:"failed_requests"`
	RateLimited int            `json:"rate_limited_requests"`
	StatusCodes map[string]int `json:"status_codes"`
}

type Service struct {
	limiter *rate.Limiter
	rng     *rand.Rand
	mu      sync.Mutex
	stats   Stats
}

func NewService(requestsPerSecond rate.Limit, burst int, seed int64) *Service {
	return &Service{
		limiter: rate.NewLimiter(requestsPerSecond, burst),
		rng:     rand.New(rand.NewSource(seed)),
		stats: Stats{
			StatusCodes: make(map[string]int),
		},
	}
}

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/requests", s.handleRequest)
	mux.HandleFunc("/stats", s.handleStats)
	return mux
}

func (s *Service) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Service) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !s.limiter.Allow() {
		s.record(http.StatusTooManyRequests)
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		return
	}

	status := s.nextStatus()
	s.record(status)
	writeJSON(w, status, map[string]int{"status": status})
}

func (s *Service) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.snapshot())
}

func (s *Service) nextStatus() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rng.Float64() < 0.7 {
		if s.rng.Intn(2) == 0 {
			return http.StatusOK
		}
		return http.StatusAccepted
	}
	if s.rng.Intn(2) == 0 {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func (s *Service) record(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.Total++
	s.stats.StatusCodes[strconv.Itoa(status)]++
	switch {
	case status == http.StatusTooManyRequests:
		s.stats.RateLimited++
	case status >= 200 && status < 300:
		s.stats.Successful++
	default:
		s.stats.Failed++
	}
}

func (s *Service) snapshot() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	codes := make(map[string]int, len(s.stats.StatusCodes))
	for status, count := range s.stats.StatusCodes {
		codes[status] = count
	}
	result := s.stats
	result.StatusCodes = codes
	return result
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
