package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	Exit = "exit"
	Echo = "echo"
	Type = "type"
)

func getType(command string) string {
	var cmd string
	switch command {
	case Exit, Echo, Type:
		cmd = "is a shell builtin"
	default:
		cmd = "not found"
	}
	return cmd
}

func main() {
	for {
		fmt.Print("$ ")
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			panic("can not read the input")
		}
		command = strings.TrimSpace(command[:len(command)-1])
		if command == "" {
			continue
		} else if command == Exit {
			break
		} else if strings.HasPrefix(command, fmt.Sprintf("%s ", Echo)) {
			fmt.Println(command[5:])
		} else if strings.HasPrefix(command, fmt.Sprintf("%s ", Type)) {
			fmt.Printf("type %s: %s\n", command, getType(command[5:]))
		} else {
			fmt.Printf("%s: command not found\n", command)
		}
	}
}
