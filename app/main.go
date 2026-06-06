package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"
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

func changeDirectory(dir string) error {
	if strings.HasPrefix(dir, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error getting user home directory: %v\n", err)
		}
		dir = strings.Replace(dir, "~", homeDir, 1)
	}
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cd: %s: No such file or directory\n", dir)
	} else if err != nil {
		return fmt.Errorf("error getting dir stats: %v\n", err)
	}
	// TODO: check for info.isDir() ?

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("error changing directory %v\n", err)
	}

	return nil
}

type AutoCompleteNode struct {
	children map[rune]*AutoCompleteNode
	isEnd    bool
}

type AutoComplete struct {
	root *AutoCompleteNode
}

func NewAutoComplete() *AutoComplete {
	return &AutoComplete{
		root: &AutoCompleteNode{
			children: make(map[rune]*AutoCompleteNode),
		},
	}
}

func (ac *AutoComplete) Insert(cmd []rune) {
	cur := ac.root
	for _, c := range cmd {
		if _, ok := cur.children[c]; !ok {
			cur.children[c] = &AutoCompleteNode{
				children: make(map[rune]*AutoCompleteNode),
			}
		}
		cur = cur.children[c]
	}
	cur.isEnd = true
}

func (ac *AutoComplete) Search(cmd []rune) [][]rune {
	cur := ac.root
	var resCmd []rune
	for _, c := range cmd {
		if _, ok := cur.children[c]; !ok {
			return [][]rune{cmd}
		}
		resCmd = append(resCmd, c)
		cur = cur.children[c]
	}

	// DFS
	v := map[*AutoCompleteNode]struct{}{}
	s := []*AutoCompleteNode{cur}
	for len(s) > 0 {
		cur = s[len(s)-1]
		s = s[:len(s)-1]
		if _, ok := v[cur]; ok {
			continue
		}
		v[cur] = struct{}{}
		for c, child := range cur.children {
			s = append(s, child)
			resCmd = append(resCmd, c)
		}
	}

	if len(resCmd) > 0 {
		// TODO: Handle isEnd?
		resCmd = append(resCmd, ' ')
		return [][]rune{resCmd}
	}

	return [][]rune{cmd}
}

func main() {
	// NOTE: Cooked Mode, Raw Mode, Restoring, File Descriptor
	autoComplete := NewAutoComplete()
	autoComplete.Insert([]rune(Echo))
	autoComplete.Insert([]rune(Exit))

	stdinFd := int(os.Stdin.Fd())
terminal:
	for {
		oldState, err := term.MakeRaw(stdinFd)
		if err != nil {
			panic(err)
		}

		var inputLine []rune
		buf := make([]byte, 1)
		fmt.Print("$ ")
	readline:
		for {
			_, err := os.Stdin.Read(buf)
			if err != nil {
				_ = term.Restore(stdinFd, oldState)
				panic(err)
			}
			char := buf[0]
			switch char {
			case 3: // ASCII code for Ctrl+C
				fmt.Print("^C\r\n") // NOTE: Carriage Return
				inputLine = nil
				fmt.Print("$ ")
			case 4: // ASCII code for Ctrl+D
				_ = term.Restore(stdinFd, oldState)
				return
			case 9: // ASCII code for Tab
				cmdRes := autoComplete.Search(inputLine)
				if len(cmdRes) > 0 {
					inputLine = cmdRes[0] // TODO: Iterate
				}
				fmt.Printf("\r$ %s", string(inputLine))
			case 10, 13: // ASCII code for Enter
				fmt.Printf("\r\n")
				_ = term.Restore(stdinFd, oldState)
				break readline
			case 127, 8: // ASCII code for backspace
				if len(inputLine) > 0 {
					inputLine = inputLine[:len(inputLine)-1]
					fmt.Print("\b \b") // Move cursor back (\b), overwrite with a blank space (" "), move cursor back again (\b)
				}
			default:
				if char >= 32 && char <= 126 {
					inputLine = append(inputLine, rune(char))
					fmt.Printf("%c", char)
				}
			}
		}

		statement := string(inputLine)

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

		var redirectfd uint = 3 // 1: stdout, 2: stderr, 3: none
		redirectAppend := false
		var outRedirectArgSlice []string
		var errRedirectArgSlice []string
		for iarg, arg := range argsSlice {
			switch arg {
			case ">", "1>", ">>", "1>>":
				outRedirectArgSlice = argsSlice[iarg+1:]
				argsSlice = argsSlice[:iarg]
				redirectfd = 1
				redirectAppend = strings.HasSuffix(arg, ">>")
			case "2>", "2>>":
				errRedirectArgSlice = argsSlice[iarg+1:]
				argsSlice = argsSlice[:iarg]
				redirectfd = 2
				redirectAppend = strings.HasSuffix(arg, ">>")
			}
		}

		var outRes string
		var errRes string
		switch command {
		case "":
			continue terminal
		case builtinCommands[Exit]:
			break terminal
		case builtinCommands[Echo]:
			outRes = strings.Join(argsSlice, " ") + "\n"
		case builtinCommands[Type]:
			outRes = getType(strings.Join(argsSlice, " ")) + "\n"
		case builtinCommands[Pwd]:
			wd, err := os.Getwd()
			if err != nil {
				fmt.Printf("error getting working directory: %v\n", err)
				continue terminal
			}
			outRes = wd + "\n"
		case builtinCommands[Cd]:
			dir := ""
			if len(argsSlice) > 0 {
				dir = argsSlice[0]
			}
			err = changeDirectory(dir)
			if err != nil {
				fmt.Print(err)
			}
			continue terminal
		default:
			_, err = lookupExecPath(command)
			if err == nil {
				cmd := exec.Command(command, argsSlice...)
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				coutput, err := cmd.Output()
				if err != nil {
					errRes = stderr.String()
				}
				outRes = string(coutput)
			} else {
				outRes = fmt.Sprintf("%s: command not found\n", statement)
			}
		}

		switch redirectfd {
		case 1, 2:
			var res string
			var redirectdepath string
			switch redirectfd {
			case 1:
				if errRes != "" {
					fmt.Print(errRes)
				}
				res, redirectdepath = outRes, outRedirectArgSlice[0]
			case 2:
				if outRes != "" {
					fmt.Print(outRes)
				}
				res, redirectdepath = errRes, errRedirectArgSlice[0]
			}
			func(redirectdepath string, res string) {
				redirectAbsPath, err := filepath.Abs(redirectdepath)
				if err != nil {
					fmt.Printf("error getting the absolute path of the redirect file: %v", err)
				}
				redirectDir := filepath.Dir(redirectAbsPath)
				err = os.MkdirAll(redirectDir, 0755) // read/write/exec to owner, read/exec to others
				if err != nil {
					fmt.Printf("error making redirect directory: %v\n", err)
				}
				flag := os.O_CREATE | os.O_WRONLY
				if redirectAppend {
					flag = flag | os.O_APPEND
				} else {
					flag = flag | os.O_TRUNC
				}
				outFile, err := os.OpenFile(redirectAbsPath, flag, 0644)
				if err != nil {
					fmt.Printf("error opening the redirect file: %v\n", err)
				}
				defer outFile.Close()
				_, err = outFile.WriteString(res)
				if err != nil {
					fmt.Printf("error wrting to the redirect file: %v\n", err)
				}
			}(redirectdepath, res)
		default:
			fmt.Print(outRes)
		}
	}
}
