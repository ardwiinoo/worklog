package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func saveWorkLog(rawInput string, now time.Time) error {
	entries := normalizeEntries(rawInput)
	if len(entries) == 0 {
		return nil
	}

	logDir, err := getLogDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	fileName := now.Format("200601") + "_daily.txt"
	filePath := filepath.Join(logDir, fileName)

	dateHeader := now.Format("02/01/2006")

	bullets := make([]string, 0, len(entries))
	for _, entry := range entries {
		bullets = append(bullets, "* "+entry)
	}

	contentBytes, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(contentBytes)

	if strings.TrimSpace(content) == "" {
		newContent := dateHeader + "\n" + strings.Join(bullets, "\n") + "\n"
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}

	newContent := insertBullets(content, dateHeader, bullets)
	return os.WriteFile(filePath, []byte(newContent), 0644)
}

func normalizeEntries(rawInput string) []string {
	normalized := strings.ReplaceAll(rawInput, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	entries := make([]string, 0, len(lines))

	for _, line := range lines {
		entry := strings.TrimSpace(line)
		if entry == "" {
			continue
		}

		entry = strings.TrimPrefix(entry, "*")
		entry = strings.TrimSpace(entry)

		if entry != "" {
			entries = append(entries, entry)
		}
	}

	return entries
}

func insertBullets(content, dateHeader string, bullets []string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	dateRegex := regexp.MustCompile(`^\d{2}/\d{2}/\d{4}\s*$`)

	dateIndex := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == dateHeader {
			dateIndex = i
			break
		}
	}

	if dateIndex == -1 {
		trimmed := strings.TrimRight(normalized, "\n")
		return trimmed + "\n\n" + dateHeader + "\n" + strings.Join(bullets, "\n") + "\n"
	}

	insertAt := len(lines)

	for i := dateIndex + 1; i < len(lines); i++ {
		if dateRegex.MatchString(strings.TrimSpace(lines[i])) {
			insertAt = i
			break
		}
	}

	for insertAt > dateIndex+1 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}

	newLines := make([]string, 0, len(lines)+len(bullets))
	newLines = append(newLines, lines[:insertAt]...)
	newLines = append(newLines, bullets...)
	newLines = append(newLines, lines[insertAt:]...)

	return strings.TrimRight(strings.Join(newLines, "\n"), "\n") + "\n"
}

func ensureMonthLog(month time.Time) (string, error) {
	logDir, err := getLogDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", err
	}

	fileName := month.Format("200601") + "_daily.txt"
	filePath := filepath.Join(logDir, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
			return "", err
		}
	}

	return filePath, nil
}

func ensureCurrentMonthLog(now time.Time) (string, error) {
	return ensureMonthLog(now)
}
