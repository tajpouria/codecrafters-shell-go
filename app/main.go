package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	Exit = "exit"
	Echo = "echo"
	Type = "type"
)

func lookupExecPath(command string) (string, error) {
	execPathList := os.Getenv("PATH")
	for _, execPath := range strings.Split(execPathList, string(os.PathListSeparator)) {
		expectedPath := filepath.Join(execPath, command)
		info, err := os.Stat(expectedPath)
		if err != nil {
			continue
		}
		if info.Mode().Perm()&0111 != 0 {
			return execPath, nil
		}
	}
	return "", errors.New("exec path not found")
}

func getType(command string) string {
	if command == Exit || command == Echo || command == Type {
		return fmt.Sprintf("%s is a shell builtin", command)
	}
	path, err := lookupExecPath(command)
	if err == nil {
		return fmt.Sprintf("%s is %s/%s", command, path, command)
	}
	return fmt.Sprintf("%s: not found", command)
}

func main() {
loop:
	for {
		fmt.Print("$ ")
		statement, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			panic("can not read the input")
		}
		statement = strings.TrimSpace(statement[:len(statement)-1])

		statementSlice := strings.Split(statement, " ")
		command, argsSlice := statementSlice[0], statementSlice[1:]

		switch command {
		case "":
			continue loop
		case Exit:
			break loop
		case Echo:
			fmt.Println(strings.Join(argsSlice, " "))
			continue loop
		case Type:
			fmt.Println(getType(strings.Join(argsSlice, " ")))
			continue loop
		}

		execPath, err := lookupExecPath(command)
		if err == nil {
			cmd := exec.Command(
				filepath.Join(
					execPath,
					command,
				),
				argsSlice...,
			)
			output, _ := cmd.Output()
			fmt.Print(string(output))
			continue loop
		}

		fmt.Printf("%s: command not found\n", statement)
	}
}
