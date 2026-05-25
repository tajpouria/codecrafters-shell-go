package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	Exit = "exit"
	Echo = "echo"
	Type = "type"
)

func lookupExecPath(command string) (string, error) {
	execPath := os.Getenv("PATH")
	for _, path := range strings.Split(execPath, string(os.PathListSeparator)) {
		expectedPath := filepath.Join(path, command)
		info, err := os.Stat(expectedPath)
		if err != nil {
			continue
		}
		if info.Mode().Perm()&0111 != 0 {
			return path, nil
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
			fmt.Println(getType(command[5:]))
		} else {
			fmt.Printf("%s: command not found\n", command)
		}
	}
}
