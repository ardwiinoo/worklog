package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func main() {
	entry := strings.Join(os.Args[1:], " ")

	if strings.TrimSpace(entry) == "" {
		fmt.Print("Apa yang lo kerjain? ")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		entry = strings.TrimSpace(text)
	}

	if entry == "" {
		fmt.Println("Entry kosong, gak disimpan.")
		return
	}

	if err := saveWorkLog(entry, time.Now()); err != nil {
		fmt.Println("Gagal save:", err)
		return
	}

	fmt.Println("Saved:", entry)
}

func saveWorkLog(entry string, now time.Time) error {
	logDir := "logs"

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	fileName := now.Format("200601") + "_daily.txt"
	filePath := filepath.Join(logDir, fileName)

	dateHeader := now.Format("02/01/2006")
	bullet := "* " + strings.TrimSpace(entry)

	contentBytes, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(contentBytes)

	if strings.TrimSpace(content) == "" {
		newContent := fmt.Sprintf("%s\n%s\n", dateHeader, bullet)
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}

	newContent := insertEntry(content, dateHeader, bullet)

	return os.WriteFile(filePath, []byte(newContent), 0644)
}

func insertEntry(content, dateHeader, bullet string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

	dateRegex := regexp.MustCompile(`^\d{2}/\d{2}/\d{4}\s*$`)

	dateIndex := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == dateHeader {
			dateIndex = i
			break
		}
	}

	// Kalau tanggal belum ada, append section baru di bawah.
	if dateIndex == -1 {
		content = strings.TrimRight(content, "\n")
		return content + "\n\n" + dateHeader + "\n" + bullet + "\n"
	}

	// Cari posisi sebelum tanggal berikutnya.
	insertAt := len(lines)
	for i := dateIndex + 1; i < len(lines); i++ {
		if dateRegex.MatchString(strings.TrimSpace(lines[i])) {
			insertAt = i
			break
		}
	}

	// Rapikan: kalau ada blank line sebelum tanggal berikutnya, insert sebelum blank line itu.
	for insertAt > dateIndex+1 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}

	newLines := append(lines[:insertAt], append([]string{bullet}, lines[insertAt:]...)...)

	return strings.TrimRight(strings.Join(newLines, "\n"), "\n") + "\n"
}
