package main

import "tempDB/server"

func main() {
	server_instance := server.Init()
	server_instance.Start()
}
