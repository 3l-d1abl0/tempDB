package utils

import (
	"errors"
	"strings"
)

type Request struct {
	Command string
	Params  []string
}

func validCommand(cmd string) bool {

	switch cmd {

	case "PING":
		return true
	case "GET":
		return true
	case "SET":
		return true
	case "DEL":
		return true
	case "FLUSHDB":
		return true

	default:
		return false
	}
}

func ParseCommands(input string) (Request, error) {

	cmd := strings.Trim(input, "\r\n")
	args := strings.Split(cmd, " ")

	//fmt.Println("[", args, "]")

	//If empty
	if len(args) == 0 {
		return Request{}, errors.New("empty command")
	}

	commandStatus := validCommand(args[0])

	if !commandStatus {
		return Request{}, errors.New("invalid command")
	}

	return Request{Command: args[0], Params: args[1:]}, nil
}
