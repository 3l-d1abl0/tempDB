package server

import (
	"fmt"
	"net"
)

type Server struct {
	Addr     string
	Listener net.Listener
}

func Init(addr string) Server {
	return Server{
		Addr: addr,
	}
}

func (server *Server) Start() {

	fmt.Printf("port : %s\n", server.Addr)
	listener, err := net.Listen("tcp", server.Addr)
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
		tcpConn, ok := connection.(*net.TCPConn)
		if ok {
			fmt.Println("TCP info:")
			fmt.Println("   Local address:", tcpConn.LocalAddr())
			fmt.Println("   Remote address:", tcpConn.RemoteAddr())
			fmt.Println("   Set keepalive:", tcpConn.SetKeepAlive(true))
			fmt.Println("   Set keepalive period:", tcpConn.SetKeepAlivePeriod(30))
		} else {
			fmt.Println("Not a TCP connection !")
		}
	}

}
