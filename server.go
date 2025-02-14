package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

// Ответ сервера (70% - положительные, 30% - отрицательные)
var responses = []int{
	http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK,
	http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK,
	http.StatusAccepted, http.StatusAccepted, http.StatusAccepted, http.StatusAccepted, http.StatusAccepted,
	http.StatusAccepted, http.StatusAccepted, http.StatusAccepted, http.StatusAccepted, http.StatusAccepted,
	http.StatusBadRequest, http.StatusBadRequest, http.StatusBadRequest,
	http.StatusInternalServerError, http.StatusInternalServerError, http.StatusInternalServerError,
}

// Лимитер запросов (5 запросов в секунду)
var limiter = rate.NewLimiter(5, 1)

// Структура для хранения статистики
type ServerStats struct {
	mu          sync.Mutex
	Total       int         `json:"total_requests"`
	StatusCodes map[int]int `json:"status_codes"`
}

var stats = ServerStats{StatusCodes: make(map[int]int)}

// Обработчик POST-запросов
func handlePost(w http.ResponseWriter, r *http.Request) {
	// Ограничение запросов
	if !limiter.Allow() {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Выбираем случайный статус
	status := responses[rand.Intn(len(responses))]

	// Логируем и обновляем статистику
	stats.mu.Lock()
	stats.Total++
	stats.StatusCodes[status]++
	stats.mu.Unlock()

	w.WriteHeader(status)
	fmt.Fprintf(w, "Response: %d", status)
}

// Обработчик GET-запросов (возвращает статистику)
func handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Функция запуска сервера
func RunServer() {
	// Загружаем переменные окружения
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env")
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT не найден в .env")
	}

	rand.Seed(time.Now().UnixNano()) // Инициализация генератора случайных чисел

	// Регистрируем обработчики
	http.HandleFunc("/post", handlePost)
	http.HandleFunc("/stats", handleGetStats)

	log.Printf("Сервер запущен на порту %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
