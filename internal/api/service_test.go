package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

func TestHealth(t *testing.T) {
	service := NewService(rate.Inf, 1, 42)
	recorder := httptest.NewRecorder()
	service.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestRequestsAreRecorded(t *testing.T) {
	service := NewService(rate.Inf, 1, 42)
	handler := service.Handler()
	for range 20 {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/requests", nil))
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/stats", nil))
	var stats Stats
	if err := json.NewDecoder(recorder.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats.Total != 20 {
		t.Fatalf("expected 20 requests, got %d", stats.Total)
	}
	if stats.Successful+stats.Failed+stats.RateLimited != stats.Total {
		t.Fatal("request counters do not add up")
	}
}

func TestRateLimit(t *testing.T) {
	service := NewService(0.001, 1, 42)
	handler := service.Handler()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/requests", nil))
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/requests", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", second.Code)
	}
}
