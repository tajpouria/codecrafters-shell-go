package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
loop:
	for {
		fmt.Print("$ ")
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			panic("can not read the input")
		}
		command = strings.TrimSpace(command[:len(command)-1])
		switch command {
		case "exit":
			break loop
		default:
			fmt.Println(command + ": command not found")
		}
	}
}
