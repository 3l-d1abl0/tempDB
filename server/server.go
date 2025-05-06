package server

import (
	"bufio"
	"fmt"
	"net"
	"tempDB/config"
	"tempDB/engine"
	"tempDB/utils"
)

type Server struct {
	Listener net.Listener
	Db       engine.Store
}

func Init() Server {
	//cfg := config.GetServerConfig()
	return Server{
		Db: engine.NewStore(),
	}
}

func (server *Server) Start() {

	//Read the configs
	cfg := config.GetServerConfig()
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	fmt.Printf("Starting server on: %s\n", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Error While Starting server: ", err)
		panic(err)
	}

	defer listener.Close()
	server.Listener = listener

	fmt.Println("Listening for connections ...")
	for {
		connection, err := server.Listener.Accept()
		if err != nil {
			fmt.Printf("Connection Refused: %s\n", err)
			continue
		}

		fmt.Println(connection)
		// tcpConn, ok := connection.(*net.TCPConn)
		// if ok {
		// 	fmt.Println("TCP info:")
		// 	fmt.Println("   Local address:", tcpConn.LocalAddr())
		// 	fmt.Println("   Remote address:", tcpConn.RemoteAddr())
		// 	fmt.Println("   Set keepalive:", tcpConn.SetKeepAlive(true))
		// 	fmt.Println("   Set keepalive period:", tcpConn.SetKeepAlivePeriod(30))
		// } else {
		// 	fmt.Println("Not a TCP connection !")
		// }

		//Run seperate Goroutine to handle connections
		go server.handleConnection(connection)
	}
}

func (server *Server) handleConnection(connection net.Conn) {

	//defer connection.Close()
	reader := bufio.NewReader(connection)

	for {

		//parse the incoming Bytes
		cmd, err := utils.ParseRESP(reader)
		if err != nil {
			connection.Write([]byte(err.Error()))
			continue
		}
		fmt.Println("Parsed: ", cmd)

		//check the validity of the commands
		if len(cmd) == 0 || !utils.ValidCommand(cmd) {
			connection.Write([]byte("+Invalid command"))
			continue
		}

		//the commnds are valid
		command := utils.Request{
			Command: cmd[0],
			Params:  cmd[1:],
		}
		fmt.Println("Computing ...: ", command)
		response, dbError := server.Db.CommandHandler(command)

		if dbError != nil {
			fmt.Println("ERR: ", dbError)
			connection.Write([]byte("+Failed"))
		} else {
			fmt.Println("RES: ", string(response))
			connection.Write(response)
			fmt.Println("Sent: ", string(response))
		}

	} //for

}
