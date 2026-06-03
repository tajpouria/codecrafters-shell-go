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

		var statementSlice []string
		argsStr := strings.TrimSpace(statement)
		argsStrLen := len(argsStr)
		ic, iarg := 0, 0
	argsLoop:
		for ic < argsStrLen {
			c := string(argsStr[ic])
			if c == "'" {
				for ic2, c2 := range argsStr[ic+1:] {
					if string(c2) == "'" {
						newic := ic + 1 + ic2
						if len(statementSlice) > iarg {
							statementSlice[iarg] = statementSlice[iarg] + argsStr[ic+1:newic]
						} else {
							statementSlice = append(statementSlice, argsStr[ic+1:newic])
						}
						ic = newic + 1
						continue argsLoop
					}
				}
			} else if c == "\"" {
				quotedArg := ""
				ic2 := ic + 1
				for ic2 < argsStrLen {
					c2 := string(argsStr[ic2])
					if string(c2) == "\\" {
						quotedArg += string(argsStr[ic2+1])
						ic2 += 1
					} else if string(c2) == "\"" {
						if len(statementSlice) > iarg {
							statementSlice[iarg] = statementSlice[iarg] + quotedArg
						} else {
							statementSlice = append(statementSlice, quotedArg)
						}
						ic = ic2 + 1
						continue argsLoop
					} else {
						quotedArg += string(c2)
					}
					ic2 += 1
				}
			} else if c == "\\" {
				if argsStrLen > ic+1 {
					escChar := string(argsStr[ic+1])
					if len(statementSlice) > iarg {
						statementSlice[iarg] = statementSlice[iarg] + escChar
					} else {
						statementSlice = append(statementSlice, escChar)
					}
					ic += 1
				}
			} else if c == " " {
				if len(statementSlice) > iarg && statementSlice[iarg] != " " {
					iarg = iarg + 1
				}
			} else {
				if len(statementSlice) > iarg {
					statementSlice[iarg] = statementSlice[iarg] + c
				} else {
					statementSlice = append(statementSlice, c)
				}
			}

			ic += 1
		}

		command := statementSlice[0]
		var argsSlice []string
		if len(statementSlice) > 1 {
			argsSlice = statementSlice[1:]
		}

		var redirectArgSlice []string
		for iarg, arg := range argsSlice {
			if arg == ">" || arg == "1>" {
				redirectArgSlice = argsSlice[iarg+1:]
				argsSlice = argsSlice[iarg+1:]
			}
		}

		var output string
		switch command {
		case "":
			continue loop
		case builtinCommands[Exit]:
			break loop
		case builtinCommands[Echo]:
			output = strings.Join(argsSlice, " ")
		case builtinCommands[Type]:
			output = getType(strings.Join(argsSlice, " ")) + "\n"
		case builtinCommands[Pwd]:
			wd, err := os.Getwd()
			if err != nil {
				fmt.Printf("error getting working directory: %v\n", err)
				continue loop
			}
			output = wd + "\n"
		case builtinCommands[Cd]:
			dir := ""
			if len(argsSlice) > 0 {
				dir = argsSlice[0]
			}
			output = changeDirectory(dir)
		}

		_, err = lookupExecPath(command)
		if err == nil {
			cmd := exec.Command(command, argsSlice...)
			coutput, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Print(string(coutput))
				continue loop
			}
			output = string(coutput)
		}

		if output != "" {
			if len(redirectArgSlice) > 0 {
				redirectAbsPath, err := filepath.Abs(redirectArgSlice[0])
				if err != nil {
					fmt.Printf("error getting the absolute path of the output file: %v", err)
					continue loop
				}
				redirectDir := filepath.Dir(redirectAbsPath)
				err = os.MkdirAll(redirectDir, 0755) // read/write/exec to owner, read/exec to others
				if err != nil {
					fmt.Printf("error making output directory: %v\n", err)
					continue loop
				}
				outFile, err := os.OpenFile(redirectAbsPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Printf("error opening the output file: %v\n", err)
					continue loop
				}
				defer outFile.Close()
				_, err = outFile.WriteString(output)
				if err != nil {
					fmt.Printf("error wrting to the output file: %v\n", err)
				}
			} else {
				fmt.Print(output)
			}
			continue loop
		}

		fmt.Printf("%s: command not found\n", statement)
	}
}
