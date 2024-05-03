package server

import (
	"bytes"
	"fmt"
	"net"
	"strings"
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

		go server.handleConnection(connection)
	} //for

} //Start

func (server *Server) handleConnection(connection net.Conn) {

	defer connection.Close()

	fmt.Printf("Processing new connection :\n")

	var received int

	buffer := bytes.NewBuffer(nil)

	for {

		chunk := make([]byte, 4096)
		read, err := connection.Read(chunk)
		if err != nil {
			//EOF
			fmt.Println(received, buffer.Bytes(), err)
			return
		}

		received += read
		buffer.Write(chunk[:read])

		if read == 0 || read < 4096 {
			fmt.Println(received, buffer.Bytes())
			//Command
			fmt.Println((buffer))
			break
		}
	} //for

	cmd := strings.Trim(buffer.String(), "\r\n")
	args := strings.Split(cmd, " ")

	fmt.Println("[", args, "]")
}
