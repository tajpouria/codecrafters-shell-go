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
	Cd   = "cd"
)

var builtinCommands = map[string]string{
	Exit: Exit,
	Echo: Echo,
	Type: Type,
	Pwd:  Pwd,
	Cd:   Cd,
}

func lookupExecPath(command string) (string, error) {
	execPathList := os.Getenv("PATH")
	// TODO: use SplitSeq
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

func changeDirectory(dir string) string {
	if strings.HasPrefix(dir, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Sprintf("error getting user home directory: %v\n", err)
		}
		dir = strings.Replace(dir, "~", homeDir, 1)
	}
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return fmt.Sprintf("cd: %s: No such file or directory\n", dir)
	} else if err != nil {
		return fmt.Sprintf("error getting dir stats: %v\n", err)
	}
	// TODO: check for info.isDir() ?

	if err := os.Chdir(dir); err != nil {
		return fmt.Sprintf("error changing directory %v\n", err)
	}

	return ""
}

func main() {
loop:
	for {
		fmt.Print("$ ")
		statement, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			panic(err)
		}
		statement = statement[:len(statement)-1]

		statement = strings.ReplaceAll(statement, "''", "")

		parts := strings.SplitN(statement, " ", 2)
		command := strings.TrimSpace(parts[0])

		var argsSlice []string
		if len(parts) >= 2 {
			qParts := strings.Split(parts[1], "'")
			for pi, p := range qParts {
				if p = strings.TrimSpace(p); p != "" && p != "\n" {
					if pi-1 >= 0 &&
						pi+1 <= len(qParts)-1 &&
						strings.TrimSpace(qParts[pi-1]) == "" &&
						strings.TrimSpace(qParts[pi+1]) == "" {
						// part was quoted
						argsSlice = append(argsSlice, p)
					} else {
						// part was note quoted
						var ppSlice []string
						for _, pp := range strings.Split(p, " ") {
							if pp = strings.TrimSpace(pp); pp != "" {
								ppSlice = append(ppSlice, strings.TrimSpace(pp))
							}
						}
						argsSlice = append(argsSlice, strings.Join(ppSlice, " "))
					}
				}
			}
		}

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
			wd, err := os.Getwd()
			if err != nil {
				fmt.Printf("error getting working directory: %v\n", err)
				continue loop
			}
			fmt.Println(wd)
			continue loop
		case builtinCommands[Cd]:
			dir := ""
			if len(argsSlice) > 0 {
				dir = argsSlice[0]
			}
			fmt.Print(changeDirectory(dir))
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
