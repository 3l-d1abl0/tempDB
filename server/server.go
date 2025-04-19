package server

import (
	"bytes"
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

		// fmt.Println(connection)
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

	fmt.Printf("Processing new connection :\n")

	var received int

	buffer := bytes.NewBuffer(nil)

	for {

		chunk := make([]byte, 4096) //Read 4MB bytes
		read, err := connection.Read(chunk)
		if err != nil {
			//EOF
			fmt.Println(received, buffer.Bytes(), err)
			return
		}

		received += read
		buffer.Write(chunk[:read])

		if read == 0 || read < 4096 {
			//fmt.Println(received, buffer.Bytes())
			//Command
			//fmt.Println((buffer))

			//check for command Validity
			request, err := utils.ParseCommands(buffer.String())
			buffer.Reset()

			if err != nil {
				connection.Write([]byte(err.Error()))
				continue
			}

			response, dbError := server.Db.CommandHandler(request)

			if dbError != nil {
				fmt.Println("ERR: ", dbError)
				connection.Write([]byte("Failed"))
			} else {
				fmt.Println("RES: ", string(response))
				connection.Write([]byte(response))
			}

		}
	} //for

	/*
		strBytes := []byte(request.Command)
		strSliceBytes := make([][]byte, len(request.Params))
		for i, s := range request.Params {
			strSliceBytes[i] = []byte(s)
		}

		// Join the byte slices into a single byte slice
		byteSlice := bytes.Join(append([][]byte{strBytes}, strSliceBytes...), []byte{})

		connection.Write(byteSlice)
	*/
}
