package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Клиент отправляет 100 запросов с лимитом 5 запросов в секунду
func sendRequests(clientID int, wg *sync.WaitGroup) {
	defer wg.Done() // Уменьшаем счетчик после завершения

	limiter := rate.NewLimiter(rate.Every(time.Second/5), 1) // Ограничение: 5 запросов в секунду
	url := "http://localhost:8080/post"
	statusCount := make(map[int]int) // Статистика по кодам ответа

	var clientWg sync.WaitGroup
	clientWg.Add(2) // Два воркера

	// Воркеры (каждый отправляет по 50 запросов)
	for i := 0; i < 2; i++ {
		go func(workerID int) {
			defer clientWg.Done()
			for j := 0; j < 50; j++ {
				// Ждем разрешения на отправку (ограничение скорости)
				if err := limiter.Wait(context.Background()); err != nil {
					log.Printf("Клиент %d, воркер %d: ошибка лимитера: %v", clientID, workerID, err)
					continue
				}

				resp, err := http.Post(url, "application/json", nil)
				if err != nil {
					log.Printf("Клиент %d, воркер %d: ошибка запроса: %v", clientID, workerID, err)
					continue
				}

				statusCount[resp.StatusCode]++ // Запоминаем код ответа
				resp.Body.Close()
			}
		}(i)
	}

	clientWg.Wait() // Ждем завершения всех воркеров

	// Вывод статистики
	fmt.Printf("Клиент %d завершил отправку запросов\n", clientID)
	fmt.Printf("Отправлено запросов: %d\n", 100)
	for status, count := range statusCount {
		fmt.Printf("Статус %d: %d раз\n", status, count)
	}
}

// Запуск клиентов 1 и 2
func RunClients12() {
	var wg sync.WaitGroup
	wg.Add(2) // Два клиента

	go sendRequests(1, &wg)
	go sendRequests(2, &wg)

	wg.Wait() // Ждем завершения всех клиентов
}
