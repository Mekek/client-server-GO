package main

import (
	"fmt"
	"net/http"
	"time"
)

func checkServer() {
	url := "http://localhost:8080/stats"
	for {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Сервер недоступен")
		} else {
			fmt.Println("Сервер доступен, статус:", resp.StatusCode)
			resp.Body.Close()
		}

		time.Sleep(5 * time.Second) // Ожидание 5 секунд перед следующей проверкой
	}
}

func RunClient3() {
	go checkServer()
}
