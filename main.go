package main

import "tempDB/server"

func main() {

	server_instance := server.Init(":8090")

	server_instance.Start()
}
