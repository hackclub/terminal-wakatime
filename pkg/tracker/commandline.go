package tracker

import "strings"

// maxCommandsPerLine bounds the work done for very long command lines.
const maxCommandsPerLine = 20

// splitCommandLine splits a shell command line into its simple commands,
// e.g. "cd api && vim main.go | tee log" becomes ["cd api", "vim main.go",
// "tee log"]. It splits on unquoted ; & && || | |& and newlines, drops
// subshell and group brackets, and drops shell keywords such as `do` and
// `then` so the command after them is seen. Loop headers like
// `for f in *` are skipped since they do not run a program.
func splitCommandLine(line string) []string {
	var commands []string
	var current strings.Builder
	quote := byte(0)

	flush := func() {
		if command := cleanCommand(current.String()); command != "" && len(commands) < maxCommandsPerLine {
			commands = append(commands, command)
		}
		current.Reset()
	}

	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0:
			current.WriteByte(c)
			if c == '\\' && quote == '"' && i+1 < len(line) {
				i++
				current.WriteByte(line[i])
			} else if c == quote {
				quote = 0
			}
		case c == '\\' && i+1 < len(line):
			current.WriteByte(c)
			i++
			current.WriteByte(line[i])
		case c == '\'' || c == '"':
			quote = c
			current.WriteByte(c)
		case c == '&' && isRedirection(line, i):
			current.WriteByte(c)
		case c == ';' || c == '&' || c == '|' || c == '\n':
			flush()
		default:
			current.WriteByte(c)
		}
	}
	flush()

	return commands
}

// isRedirection reports whether the & at index i belongs to a redirection
// such as 2>&1 or &> rather than being a control operator.
func isRedirection(line string, i int) bool {
	if i > 0 && (line[i-1] == '>' || line[i-1] == '<') {
		return true
	}
	return i+1 < len(line) && line[i+1] == '>'
}

// shellKeywords precede a command without being one.
var shellKeywords = map[string]bool{
	"do": true, "then": true, "else": true, "elif": true, "if": true,
	"while": true, "until": true, "!": true,
}

// shellHeaders start compound commands that do not run a program themselves.
var shellHeaders = map[string]bool{
	"for": true, "case": true, "select": true, "function": true,
	"done": true, "fi": true, "esac": true, "}": true, ")": true,
}

func cleanCommand(command string) string {
	command = strings.TrimSpace(command)
	for {
		trimmed := strings.TrimSpace(strings.TrimLeft(command, "({"))
		fields := strings.Fields(trimmed)
		if len(fields) > 0 && shellKeywords[fields[0]] {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, fields[0]))
		}
		if trimmed == command {
			break
		}
		command = trimmed
	}
	command = strings.TrimSpace(strings.TrimRight(command, ")} \t"))

	fields := strings.Fields(command)
	if len(fields) == 0 || shellHeaders[fields[0]] {
		return ""
	}
	return command
}
