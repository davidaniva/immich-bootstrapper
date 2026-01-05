package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// These placeholders are exactly 128 characters each.
// The Immich server patches these with actual values at download time.
// DO NOT MODIFY THE LENGTH OF THESE STRINGS.
var (
	ServerURL  = "__IMMICH_SERVER_URL_PLACEHOLDER_________________________________________________________________________________________________"
	SetupToken = "__IMMICH_SETUP_TOKEN_PLACEHOLDER_________________________________________________________________________________________________"
)

// GitHub release info for the main importer app
const (
	GitHubRepo = "immich-app/immich-importer"
	AppVersion = "latest"
)

func main() {
	fmt.Println("Immich Google Photos Importer - Bootstrap")
	fmt.Println("==========================================")

	// Validate that we've been patched
	serverURL := strings.TrimRight(ServerURL, "\x00_ ")
	setupToken := strings.TrimRight(SetupToken, "\x00_ ")

	if strings.HasPrefix(serverURL, "__IMMICH_") || serverURL == "" {
		fmt.Fprintln(os.Stderr, "Error: This bootstrap binary was not properly configured.")
		fmt.Fprintln(os.Stderr, "Please download a fresh copy from your Immich server.")
		os.Exit(1)
	}

	fmt.Printf("Server: %s\n", serverURL)
	fmt.Println()

	// Determine platform and file extension
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	// Get user's app data directory
	appDir, err := getAppDataDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting app data directory: %v\n", err)
		os.Exit(1)
	}

	// Create app directory if it doesn't exist
	if err := os.MkdirAll(appDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating app directory: %v\n", err)
		os.Exit(1)
	}

	appPath := filepath.Join(appDir, "immich-importer"+ext)

	// Check if we need to download
	needsDownload := true
	if _, err := os.Stat(appPath); err == nil {
		fmt.Println("Found existing importer, checking for updates...")
		// In a full implementation, we'd check the version/checksum
		// For now, we'll just use what's there
		needsDownload = false
	}

	if needsDownload {
		// Construct download URL
		var downloadURL string
		if AppVersion == "latest" {
			downloadURL = fmt.Sprintf(
				"https://github.com/%s/releases/latest/download/immich-importer-%s%s",
				GitHubRepo, platform, ext,
			)
		} else {
			downloadURL = fmt.Sprintf(
				"https://github.com/%s/releases/download/%s/immich-importer-%s%s",
				GitHubRepo, AppVersion, platform, ext,
			)
		}

		fmt.Printf("Downloading importer from GitHub...\n")
		fmt.Printf("URL: %s\n", downloadURL)
		fmt.Println()

		if err := downloadFile(appPath, downloadURL); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to download importer: %v\n", err)
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Please check your internet connection and try again.")
			fmt.Fprintln(os.Stderr, "If the problem persists, the release may not be available yet.")
			os.Exit(1)
		}

		// Make executable on Unix systems
		if runtime.GOOS != "windows" {
			if err := os.Chmod(appPath, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to make app executable: %v\n", err)
			}
		}

		fmt.Println("Download complete!")
	}

	fmt.Println()
	fmt.Println("Launching Immich Importer...")
	fmt.Println()

	// Launch the main app with server and token
	cmd := exec.Command(appPath, "--server", serverURL, "--token", setupToken)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error running importer: %v\n", err)
		os.Exit(1)
	}
}

func getAppDataDir() (string, error) {
	var baseDir string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(home, "Library", "Application Support", "ImmichImporter")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		baseDir = filepath.Join(appData, "ImmichImporter")
	default: // Linux and others
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(home, ".config")
		}
		baseDir = filepath.Join(configDir, "immich-importer")
	}

	return baseDir, nil
}

func downloadFile(destPath, url string) error {
	// Create temporary file in the same directory
	dir := filepath.Dir(destPath)
	tmpFile, err := os.CreateTemp(dir, "download-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath) // Clean up temp file on error
	}()

	// Follow redirects (GitHub releases redirect to CDN)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Get content length for progress
	contentLength := resp.ContentLength

	// Create progress writer
	hasher := sha256.New()
	multiWriter := io.MultiWriter(tmpFile, hasher)

	var written int64
	buf := make([]byte, 32*1024)
	lastPercent := -1

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			nw, writeErr := multiWriter.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			if nw != n {
				return fmt.Errorf("short write")
			}
			written += int64(n)

			// Show progress
			if contentLength > 0 {
				percent := int(float64(written) / float64(contentLength) * 100)
				if percent != lastPercent && percent%10 == 0 {
					fmt.Printf("  Progress: %d%%\n", percent)
					lastPercent = percent
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read: %w", readErr)
		}
	}

	// Close temp file before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Log the checksum (useful for debugging)
	checksum := hex.EncodeToString(hasher.Sum(nil))
	fmt.Printf("  Downloaded: %.2f MB (SHA256: %s...)\n", float64(written)/1024/1024, checksum[:16])

	// Atomically move temp file to destination
	if err := os.Rename(tmpPath, destPath); err != nil {
		// On Windows, we might need to remove the destination first
		if runtime.GOOS == "windows" {
			os.Remove(destPath)
			if err := os.Rename(tmpPath, destPath); err != nil {
				return fmt.Errorf("failed to move downloaded file: %w", err)
			}
		} else {
			return fmt.Errorf("failed to move downloaded file: %w", err)
		}
	}

	return nil
}
