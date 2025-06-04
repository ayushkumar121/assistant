package main

import (
	"os"
	"strings"
)

// saveMemory appends new memory line if it's not already present,
// and trims to last memoryLimit lines before saving to disk.
func saveMemory(newMemory string) error {
	if strings.TrimSpace(newMemory) == "" {
		return nil
	}

	existing := loadMemory()
	lines := strings.Split(existing, "\n")

	// Add new memory only if it's not already in the list
	if !containsLine(lines, newMemory) {
		// only append newline if existing memory isn't empty
		if len(existing) > 0 {
			lines = append(lines, newMemory)
		} else {
			lines = []string{newMemory}
		}
	}

	// Trim to last memoryLimit lines
	if len(lines) > memoryLimit {
		lines = lines[len(lines)-memoryLimit:]
	}

	combined := strings.Join(lines, "\n")
	return os.WriteFile(memoryFile, []byte(combined), 0644)
}

func containsLine(lines []string, line string) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) == strings.TrimSpace(line) {
			return true
		}
	}
	return false
}

func loadMemory() string {
	data, err := os.ReadFile(memoryFile)
	if err != nil {
		return ""
	}
	return string(data)
}
