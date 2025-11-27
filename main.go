/*
 * Project: GitUp 	Command-line tool to upload files directly to GitHub
 * File: main.go 	Main application entry point
 * Version: 		v1.0.0
 * Author: 			Kolja Nolte
 * Author URL: 		https://www.kolja-nolte.com
 * License: 		MIT
 * Repository: 		https://github.com/thaikolja/gitup
 */

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	// configDir is the directory under the user's home where the configuration is stored.
	configDir = ".gitup"
	// configFile is the filename for the JSON configuration file.
	configFile = "config.json"

	maxUploadSizeBytes = 25 * 1024 * 1024 // 25 MB practical limit for GitHub API
)

// Config holds the user's GitUp configuration.
type Config struct {
	// Token is the GitHub Personal Access Token used to authenticate API requests.
	Token string `json:"token"`
	// Repository is the target repository in the format owner/repo.
	Repository string `json:"repository"`
}

func main() {
	var (
		configCmd = flag.Bool("config", false, "Configure GitUp")
		verbose   = flag.Bool("v", false, "Enable verbose logging")
		branch    = flag.String("branch", "main", "Git branch for uploaded files")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] <file-path>\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *configCmd {
		configureGitUp()
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	filePath := args[0]
	if err := validateInputFile(filePath); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid file: %v\n", err)
		os.Exit(1)
	}

	config, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run 'gitup -config' first")
		os.Exit(1)
	}

	if err := validateRepository(config.Repository); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid repository: %v\n", err)
		os.Exit(1)
	}

	uploader := &Uploader{
		Client:     &http.Client{Timeout: 15 * time.Second},
		Branch:     *branch,
		Verbose:    *verbose,
		Repository: config.Repository,
		Token:      config.Token,
	}

	if err := uploader.Upload(filePath); err != nil {
		fmt.Fprintf(os.Stderr, "Upload failed: %v\n", err)
		os.Exit(1)
	}
}

// configureGitUp runs an interactive configuration flow.
// It prompts the user for a GitHub token and repository, and attempts to save the token
// to the macOS Keychain, and writes the remainder of the config to disk.
func configureGitUp() {
	fmt.Println("=== GitUp Configuration ===")

	var config Config

	// Get GitHub token
	fmt.Print("Enter your GitHub Personal Access Token: ")
	if _, err := fmt.Scanln(&config.Token); err != nil {
		fmt.Printf("Error reading token: %v\n", err)
		os.Exit(1)
	}

	// Get repository
	fmt.Print("Enter repository (owner/repo): ")
	if _, err := fmt.Scanln(&config.Repository); err != nil {
		fmt.Printf("Error reading repository: %v\n", err)
		os.Exit(1)
	}

	// Save to keychain
	err := saveToKeychain(config.Token)
	if err != nil {
		fmt.Printf("Warning: Could not save to keychain: %v\n", err)
		fmt.Println("Token will be saved in config file instead")
	} else {
		fmt.Println("✓ Token saved to macOS Keychain")
		config.Token = "" // Don't store in file if in keychain
	}

	// Save config
	err = saveConfig(config)
	if err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Configuration saved!")
}

// saveToKeychain stores the provided token in the macOS Keychain using the
// `security` CLI. The entry is tagged with the current username and the service
// name "GitUp".
func saveToKeychain(token string) error {
	cmd := exec.Command("security", "add-generic-password",
		"-a", os.Getenv("USER"),
		"-s", "GitUp",
		"-w", token,
		"-U") // -U updates if exists
	return cmd.Run()
}

// loadFromKeychain retrieves the token previously saved under the "GitUp."
// service in the macOS Keychain using the `security` CLI.
func loadFromKeychain() (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-a", os.Getenv("USER"),
		"-s", "GitUp",
		"-w")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// saveConfig writes the provided Config to the user's config directory as JSON.
// The config directory is created with restrictive permissions.
func saveConfig(config Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine home directory: %w", err)
	}
	configPath := filepath.Join(homeDir, configDir)

	// Create config directory
	if err := os.MkdirAll(configPath, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	configFilePath := filepath.Join(configPath, configFile)
	return os.WriteFile(configFilePath, data, 0600)
}

// loadConfig reads the JSON configuration file from disk, unmarshals it into a
// Config, and attempts to load the token from the keychain if it is not present
// in the file.
func loadConfig() (Config, error) {
	var config Config

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config, fmt.Errorf("failed to determine home directory: %w", err)
	}
	configFilePath := filepath.Join(homeDir, configDir, configFile)

	// Read config file
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	// Try to load token from keychain if not in config
	if config.Token == "" {
		token, err := loadFromKeychain()
		if err == nil {
			config.Token = token
		}
	}

	if err := validateRepository(config.Repository); err != nil {
		return config, err
	}

	return config, nil
}

func validateRepository(repo string) error {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("repository must be in 'owner/repo' format")
	}
	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("repository owner and name must be non-empty")
	}
	return nil
}

var transliterationMap = map[rune]string{
	'ä': "ae",
	'ö': "oe",
	'ü': "ue",
	'ß': "ss",
	'á': "a",
	'à': "a",
	'â': "a",
	'ã': "a",
	'å': "a",
	'ā': "a",
	'é': "e",
	'è': "e",
	'ê': "e",
	'ë': "e",
	'ī': "i",
	'í': "i",
	'ì': "i",
	'î': "i",
	'ñ': "n",
	'ó': "o",
	'ò': "o",
	'ô': "o",
	'õ': "o",
	'ø': "o",
	'ū': "u",
	'ú': "u",
	'ù': "u",
	'û': "u",
	'ç': "c",
	'ý': "y",
	'ÿ': "y",
}

// transliterateToASCII replaces a subset of non-ASCII runes with ASCII
// equivalents to keep sanitized filenames predictable.
func transliterateToASCII(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r <= 127:
			b.WriteRune(r)
		case transliterationMap[r] != "":
			b.WriteString(transliterationMap[r])
		default:
			// skip unsupported characters
		}
	}
	return b.String()
}

// sanitizeFilename normalizes the provided filename by lowercasing it,
// converting spaces to dashes, stripping non-alphanumeric characters (except dashes),
// and preserving the extension (which is also lowercased).
func sanitizeFilename(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = transliterateToASCII(strings.ToLower(base))

	var b strings.Builder
	for _, r := range base {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case r == '-':
			b.WriteRune(r)
		}
	}

	sanitized := b.String()
	if sanitized == "" {
		sanitized = "file"
	}

	return sanitized + ext
}

// getUploadFolder returns the appropriate folder under the repository based on
// the file's extension. Unknown extensions map to "files".
func getUploadFolder(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// Map extensions to folders
	folderMap := map[string]string{
		// Images
		".png":  "img",
		".jpg":  "img",
		".jpeg": "img",
		".gif":  "img",
		".svg":  "img",
		".webp": "img",
		".ico":  "img",

		// Data files
		".json": "data",
		".xml":  "data",
		".csv":  "data",
		".yaml": "data",
		".yml":  "data",
		".toml": "data",

		// Documents
		".pdf":  "docs",
		".md":   "docs",
		".txt":  "docs",
		".doc":  "docs",
		".docx": "docs",

		// Videos
		".mp4":  "video",
		".mov":  "video",
		".avi":  "video",
		".webm": "video",

		// Audio
		".mp3":  "audio",
		".wav":  "audio",
		".ogg":  "audio",
		".flac": "audio",

		// Archives
		".zip": "archives",
		".tar": "archives",
		".gz":  "archives",
		".rar": "archives",
	}

	if folder, exists := folderMap[ext]; exists {
		return folder
	}

	// Default folder for unknown extensions
	return "files"
}

// formatOutput returns a markdown-formatted string appropriate for the file type.
// Images are rendered using markdown image syntax, other files use a link.
func formatOutput(filename, url string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// Image extensions use markdown image syntax
	imageExts := []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return fmt.Sprintf("![%s](%s)", filename, url)
		}
	}

	// Everything else uses markdown link syntax
	return fmt.Sprintf("[%s](%s)", filename, url)
}

// ensureUniqueFilename checks if the given path already exists in the repository
// and appends -1, -2, etc. before the extension until a free name is found.
func ensureUniqueFilename(owner, repo, folder, filename, token string) (string, error) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := filepath.Ext(filename)

	candidate := filename
	counter := 1
	for {
		exists, err := pathExistsOnGitHub(owner, repo, folder, candidate, token)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d%s", base, counter, ext)
		counter++
	}
}

// pathExistsOnGitHub performs a HEAD request against the GitHub contents API to
// determine whether a file already exists at the given folder and filename.
func pathExistsOnGitHub(owner, repo, folder, filename, token string) (bool, error) {
	path := filepath.Join(folder, filename)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, err
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close response body: %v\n", err)
		}
	}(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return false, fmt.Errorf("GitHub API auth error while checking path (%s): %s", path, resp.Status)
	default:
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("unexpected GitHub API response while checking path (%s): %s - %s", path, resp.Status, string(body))
	}
}

// validateInputFile checks if the given file path refers to a valid, accessible file.
func validateInputFile(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("could not access file: %v", err)
	}

	if info.IsDir() {
		return fmt.Errorf("expected a file but found a directory")
	}

	if info.Size() == 0 {
		return fmt.Errorf("file is empty")
	}

	if info.Size() > maxUploadSizeBytes {
		return fmt.Errorf("file exceeds maximum upload size of %d bytes", maxUploadSizeBytes)
	}

	return nil
}

// Uploader is a struct that handles file uploading with configurable options.
type Uploader struct {
	Client     *http.Client // HTTP client for making requests
	Branch     string       // Git branch for uploaded files
	Verbose    bool         // Enable verbose logging
	Repository string       // Target repository in the format owner/repo
	Token      string       // GitHub Personal Access Token
}

// Upload uploads the given file to the configured GitHub repository.
func (u *Uploader) Upload(filePath string) error {
	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Get filename
	filename := filepath.Base(filePath)

	// Sanitize filename
	sanitizedFilename := sanitizeFilename(filename)

	// Determine upload folder based on file extension
	folder := getUploadFolder(sanitizedFilename)

	// Construct GitHub API URL
	parts := strings.Split(u.Repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format. Use: owner/repo")
	}
	owner, repo := parts[0], parts[1]

	uniqueFilename, err := ensureUniqueFilename(owner, repo, folder, sanitizedFilename, u.Token)
	if err != nil {
		return fmt.Errorf("failed to determine unique filename: %w", err)
	}

	uploadPath := filepath.Join(folder, uniqueFilename)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s",
		owner, repo, uploadPath)

	// Prepare request body
	requestBody := map[string]string{
		"message": fmt.Sprintf("Upload %s via GitUp", filename),
		"content": base64.StdEncoding.EncodeToString(fileData),
	}

	bodyJSON, _ := json.Marshal(requestBody)

	// Create HTTP request
	req, err := http.NewRequest("PUT", apiURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+u.Token)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := u.Client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error: %s - %s", resp.Status, string(body))
	}

	// Print success message with URL
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, u.Branch, uploadPath)
	output := formatOutput(filename, rawURL)

	fmt.Println(output)

	return nil
}
