package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func getGitHubConfig() (token, repo string, err error) {
	baseDir, err := getAppBaseDir()
	if err != nil {
		return "", "", err
	}

	configPath := filepath.Join(baseDir, "github_token.txt")
	contentBytes, err := os.ReadFile(configPath)
	if err != nil {
		return "", "", err // Fallback triggered: file doesn't exist or can't be read
	}

	content := strings.ReplaceAll(string(contentBytes), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if token == "" {
			token = line
			continue
		}
		if repo == "" {
			repo = line
			break
		}
	}

	if token == "" || repo == "" {
		return "", "", errors.New("github_token.txt is invalid (missing token or repo)")
	}

	return token, repo, nil
}

func PullFromGitHub(now time.Time) {
	token, repo, err := getGitHubConfig()
	if err != nil {
		LogToConsole("GitHub Sync skipped (Local only mode). Reason: %v", err)
		return
	}

	LogToConsole("Checking for updates from GitHub...")

	fileName := now.Format("200601") + "_daily.txt"
	apiPath := "logs/" + fileName
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", repo, apiPath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		LogToConsole("Error creating pull request: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		LogToConsole("Network error while pulling from GitHub: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		LogToConsole("No remote backup found for %s yet.", fileName)
		return
	}

	if resp.StatusCode != 200 {
		LogToConsole("GitHub returned status %d when pulling file.", resp.StatusCode)
		return
	}

	var result struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		LogToConsole("Failed to decode GitHub response: %v", err)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(result.Content, "\n", ""))
	if err != nil {
		LogToConsole("Failed to decode base64 file content: %v", err)
		return
	}

	logDir, err := getLogDir()
	if err != nil {
		LogToConsole("Failed to get local log dir: %v", err)
		return
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		LogToConsole("Failed to create log dir: %v", err)
		return
	}

	filePath := filepath.Join(logDir, fileName)
	
	// Size check to avoid overwriting local with smaller remote
	localInfo, err := os.Stat(filePath)
	if err == nil {
		if int64(len(decoded)) <= localInfo.Size() {
			LogToConsole("Local file is equal or larger than remote. Skipping download.")
			return
		}
	}

	if err := os.WriteFile(filePath, decoded, 0644); err != nil {
		LogToConsole("Failed to write downloaded file: %v", err)
		return
	}

	LogToConsole("Successfully synchronized %s from GitHub.", fileName)
}

func PushToGitHub(now time.Time) {
	token, repo, err := getGitHubConfig()
	if err != nil {
		return // Silent fallback, user knows they are local-only
	}

	fileName := now.Format("200601") + "_daily.txt"
	logDir, err := getLogDir()
	if err != nil {
		return
	}
	filePath := filepath.Join(logDir, fileName)

	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		LogToConsole("Failed to read local file for pushing: %v", err)
		return
	}

	LogToConsole("Pushing %s to GitHub...", fileName)

	apiPath := "logs/" + fileName
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", repo, apiPath)

	// Get file SHA first
	sha := ""
	getReq, _ := http.NewRequest("GET", url, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getReq.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	getResp, err := client.Do(getReq)
	if err == nil {
		defer getResp.Body.Close()
		if getResp.StatusCode == 200 {
			var getRes struct {
				Sha string `json:"sha"`
			}
			json.NewDecoder(getResp.Body).Decode(&getRes)
			sha = getRes.Sha
		}
	}

	encodedContent := base64.StdEncoding.EncodeToString(contentBytes)
	
	payload := map[string]string{
		"message": "Update worklog " + fileName,
		"content": encodedContent,
	}
	if sha != "" {
		payload["sha"] = sha
	}

	payloadBytes, _ := json.Marshal(payload)
	putReq, _ := http.NewRequest("PUT", url, bytes.NewBuffer(payloadBytes))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("Accept", "application/vnd.github.v3+json")
	putReq.Header.Set("Content-Type", "application/json")

	putResp, err := client.Do(putReq)
	if err != nil {
		LogToConsole("Network error while pushing to GitHub: %v", err)
		return
	}
	defer putResp.Body.Close()

	if putResp.StatusCode == 200 || putResp.StatusCode == 201 {
		LogToConsole("Successfully pushed %s to GitHub.", fileName)
	} else {
		bodyBytes, _ := io.ReadAll(putResp.Body)
		LogToConsole("Failed to push to GitHub (Status %d): %s", putResp.StatusCode, string(bodyBytes))
	}
}
