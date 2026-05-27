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
	Pwd  = "pwd"
)

var builtinCommands = map[string]string{
	Exit: Exit,
	Echo: Echo,
	Type: Type,
	Pwd:  Pwd,
}

var wd string

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
	if _, ok := builtinCommands[command]; ok {
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
		path, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		wd = path

		fmt.Print("$ ")
		statement, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			panic(err)
		}
		statement = strings.TrimSpace(statement[:len(statement)-1])

		statementSlice := strings.Split(statement, " ")
		command, argsSlice := statementSlice[0], statementSlice[1:]

		switch command {
		case "":
			continue loop
		case builtinCommands[Exit]:
			break loop
		case builtinCommands[Echo]:
			fmt.Println(strings.Join(argsSlice, " "))
			continue loop
		case builtinCommands[Type]:
			fmt.Println(getType(strings.Join(argsSlice, " ")))
			continue loop
		case builtinCommands[Pwd]:
			fmt.Println(wd)
			continue loop
		}

		_, err = lookupExecPath(command)
		if err == nil {
			cmd := exec.Command(command, argsSlice...)
			output, _ := cmd.Output()
			fmt.Print(string(output))
			continue loop
		}

		fmt.Printf("%s: command not found\n", statement)
	}
}
