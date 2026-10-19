package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// keyValue is a single setting to write into an INI section.
type keyValue struct {
	key   string
	value string
}

// updateINISection rewrites only the given keys in one section of an INI
// file, leaving every other line (comments, unknown keys, other sections)
// byte-for-byte intact. Missing keys are appended to the end of the section
// and a missing section is appended to the file.
func updateINISection(path, section string, values []keyValue) error {
	if len(values) == 0 {
		return nil
	}

	var lines []string
	perm := os.FileMode(0600)
	if data, err := os.ReadFile(path); err == nil {
		content := strings.TrimSuffix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
		if content != "" {
			lines = strings.Split(content, "\n")
		}
		if info, err := os.Stat(path); err == nil {
			perm = info.Mode().Perm()
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	lines = updateSection(lines, section, values)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to a temp file and rename, so a crash never leaves a half-written config.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".wakatime.cfg.*")
	if err != nil {
		return fmt.Errorf("failed to create temp config file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write config file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return os.Rename(tmp.Name(), path)
}

func updateSection(lines []string, section string, values []keyValue) []string {
	start, end := findSection(lines, section)
	if start < 0 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "["+section+"]")
		for _, kv := range values {
			lines = append(lines, formatKeyValue(kv))
		}
		return lines
	}

	for _, kv := range values {
		keyStart, keyEnd := findKey(lines, start+1, end, kv.key)
		if keyStart < 0 {
			insertAt := lastContentLine(lines, start, end) + 1
			lines = splice(lines, insertAt, insertAt, formatKeyValue(kv))
			end++
			continue
		}
		lines = splice(lines, keyStart, keyEnd, formatKeyValue(kv))
		end -= keyEnd - keyStart - 1
	}

	return lines
}

// findSection returns the index of the section header and the index one past
// the last line belonging to it, or -1 if the section does not exist.
func findSection(lines []string, section string) (int, int) {
	start := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !isSectionHeader(trimmed) {
			continue
		}
		if start >= 0 {
			return start, i
		}
		if strings.EqualFold(strings.TrimSpace(trimmed[1:len(trimmed)-1]), section) {
			start = i
		}
	}
	if start < 0 {
		return -1, -1
	}
	return start, len(lines)
}

// findKey returns the line range [start, end) covering a key and any indented
// continuation lines of its value, or -1 if the key is not found.
func findKey(lines []string, from, to int, key string) (int, int) {
	for i := from; i < to; i++ {
		if lineKey(lines[i]) != key {
			continue
		}
		end := i + 1
		if value := lineValue(lines[i]); strings.HasPrefix(value, `"""`) && (len(value) < 6 || !strings.HasSuffix(value, `"""`)) {
			// Triple-quoted value continues until the closing quotes.
			for end < to && !strings.Contains(lines[end], `"""`) {
				end++
			}
			if end < to {
				end++
			}
			return i, end
		}
		for end < to && isContinuation(lines[end]) {
			end++
		}
		return i, end
	}
	return -1, -1
}

func lastContentLine(lines []string, start, end int) int {
	last := start
	for i := start + 1; i < end; i++ {
		if strings.TrimSpace(lines[i]) != "" {
			last = i
		}
	}
	return last
}

func lineKey(line string) string {
	if line == "" || line[0] == ' ' || line[0] == '\t' {
		return ""
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] == '#' || trimmed[0] == ';' || isSectionHeader(trimmed) {
		return ""
	}
	idx := strings.IndexAny(trimmed, "=:")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(trimmed[:idx])
}

func lineValue(line string) string {
	idx := strings.IndexAny(line, "=:")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(line[idx+1:])
}

func isContinuation(line string) bool {
	return line != "" && (line[0] == ' ' || line[0] == '\t') && strings.TrimSpace(line) != ""
}

func isSectionHeader(trimmed string) bool {
	return len(trimmed) >= 2 && trimmed[0] == '[' && trimmed[len(trimmed)-1] == ']'
}

// formatKeyValue writes multi-line values in triple quotes, the format
// gopkg.in/ini.v1 reads back.
func formatKeyValue(kv keyValue) string {
	if strings.Contains(kv.value, "\n") {
		return kv.key + " = \"\"\"" + kv.value + "\"\"\""
	}
	return kv.key + " = " + kv.value
}

func splice(lines []string, from, to int, replacement string) []string {
	out := make([]string, 0, len(lines)-(to-from)+1)
	out = append(out, lines[:from]...)
	out = append(out, replacement)
	return append(out, lines[to:]...)
}
