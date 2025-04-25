package utils

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Request struct {
	Command string
	Params  []string
}

/*
  - Validate Commands - basically check if the incoming commands
    are as per the command structure supported
*/
func ValidCommand(cmd []string) bool {

	switch cmd[0] {

	case "PING":
		if len(cmd) != 1 {
			return false
		}
		return true
	case "GET":
		if len(cmd) != 2 {
			return false
		}
		return true
	case "SET":
		if len(cmd) < 3 || len(cmd) > 5 {
			return false
		}
		return true
	case "DEL":
		if len(cmd) != 2 {
			return false
		}
		return true
	case "FLUSHDB":
		if len(cmd) != 1 {
			return false
		}
		return true
	case "EXPIRE":
		if len(cmd) != 3 {
			return false
		}
		return true
	case "TTL":
		if len(cmd) != 2 {
			return false
		}
		return true
	default:
		return false
	}
}

// func ParseCommands(input string) (Request, error) {

// 	cmd := strings.Trim(input, "\r\n")
// 	args := strings.Split(cmd, " ")

// 	//fmt.Println("[", args, "]")

// 	//If empty
// 	if len(args) == 0 {
// 		return Request{}, errors.New("empty command")
// 	}

// 	commandStatus := validCommand(args[0])

// 	if !commandStatus {
// 		return Request{}, errors.New("invalid command")
// 	}

// 	return Request{Command: args[0], Params: args[1:]}, nil
// }

// ParseRESP reads a RESP array command from the connection and returns a slice of strings.
func ParseRESP(r *bufio.Reader) ([]string, error) {
	//remove delimeter and read command Length
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}

	fmt.Println("LINE: ", line)
	//check resp Format
	if len(line) == 0 || line[0] != '*' {
		return nil, fmt.Errorf("expected array")
	}
	numElements, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	fmt.Println("ELEMENTS: ", numElements)
	if err != nil {
		return nil, fmt.Errorf("invalid array length")
	}

	//read the rest of the commands
	result := make([]string, 0, numElements) //len, cap
	for i := 0; i < numElements; i++ {
		bulkHeader, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if len(bulkHeader) == 0 || bulkHeader[0] != '$' {
			return nil, fmt.Errorf("expected bulk string")
		}
		bulkLen, err := strconv.Atoi(strings.TrimSpace(bulkHeader[1:]))
		if err != nil {
			return nil, fmt.Errorf("invalid bulk string length")
		}
		bulkData := make([]byte, bulkLen+2) // +2 for \r\n
		if _, err := io.ReadFull(r, bulkData); err != nil {
			return nil, err
		}
		result = append(result, string(bulkData[:bulkLen]))
	}

	fmt.Println("RESULT: ", result)
	return result, nil
}
