package main

func main() {
	go RunServer()
	go RunClients12()
	go RunClient3()

	select {}
}
