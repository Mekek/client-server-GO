package main

func main() {
	go RunServer()    // Запуск сервера
	go RunClients12() // Запуск клиентов 1 и 2
	go RunClient3()   // запуск клиента 3

	select {} // Бесконечное ожидание, чтобы программа не завершалась
}
