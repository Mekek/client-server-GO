package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type result struct {
	clientID int
	counts   map[int]int
	errors   int
}

func main() {
	baseURL := flag.String("url", "http://localhost:8080", "server base URL")
	clients := flag.Int("clients", 2, "number of independent clients")
	workers := flag.Int("workers", 2, "workers per client")
	requests := flag.Int("requests", 100, "requests per client")
	requestsPerSecond := flag.Float64("rate", 5, "request rate per client")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go monitorHealth(ctx, *baseURL)

	results := make(chan result, *clients)
	var wg sync.WaitGroup
	for clientID := 1; clientID <= *clients; clientID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			results <- runClient(ctx, id, *baseURL, *workers, *requests, *requestsPerSecond)
		}(clientID)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	for current := range results {
		fmt.Printf("client %d: errors=%d", current.clientID, current.errors)
		statuses := make([]int, 0, len(current.counts))
		for status := range current.counts {
			statuses = append(statuses, status)
		}
		sort.Ints(statuses)
		for _, status := range statuses {
			fmt.Printf(" %d=%d", status, current.counts[status])
		}
		fmt.Println()
	}
}

func runClient(ctx context.Context, id int, baseURL string, workers, requests int, perSecond float64) result {
	jobs := make(chan struct{})
	statuses := make(chan int, workers)
	limiter := rate.NewLimiter(rate.Limit(perSecond), 1)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				if err := limiter.Wait(ctx); err != nil {
					statuses <- 0
					continue
				}
				request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/requests", nil)
				if err != nil {
					statuses <- 0
					continue
				}
				response, err := httpClient.Do(request)
				if err != nil {
					statuses <- 0
					continue
				}
				_, _ = io.Copy(io.Discard, response.Body)
				response.Body.Close()
				statuses <- response.StatusCode
			}
		}()
	}

	go func() {
		for range requests {
			jobs <- struct{}{}
		}
		close(jobs)
		wg.Wait()
		close(statuses)
	}()

	output := result{clientID: id, counts: make(map[int]int)}
	for status := range statuses {
		if status == 0 {
			output.errors++
		} else {
			output.counts[status]++
		}
	}
	return output
}

func monitorHealth(ctx context.Context, baseURL string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			request, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health", nil)
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				log.Printf("health check failed: %v", err)
				continue
			}
			response.Body.Close()
			log.Printf("health status: %d", response.StatusCode)
		}
	}
}
