package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/trungtran/coder/internal/version"
)

// Release holds relevant fields from the GitHub API response.
type Release struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	HTMLURL    string  `json:"html_url"`
	Assets     []Asset `json:"assets"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
}

// Asset represents a single release file.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckLatestRelease fetches the latest release from GitHub.
func CheckLatestRelease() (*Release, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", version.GitHubAPILatestURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", version.BinaryName+"/"+version.Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found at %s", version.GitHubReleasesURL())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}
	return &release, nil
}

// IsNewer returns true if latest semver tag is strictly newer than current.
// Strips leading 'v' and compares major.minor.patch numerically.
func IsNewer(current, latest string) bool {
	c := parseVersion(current)
	l := parseVersion(latest)
	for i := range c {
		if l[i] > c[i] {
			return true
		}
		if l[i] < c[i] {
			return false
		}
	}
	return false
}

// CurrentPlatformAsset returns the asset name for the current OS/arch.
func CurrentPlatformAsset() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if goos == "windows" {
		return fmt.Sprintf("%s-%s-%s.exe", version.BinaryName, goos, goarch)
	}
	return fmt.Sprintf("%s-%s-%s", version.BinaryName, goos, goarch)
}

// FindAsset returns the download URL for the current platform from a release.
func FindAsset(release *Release) (Asset, bool) {
	name := CurrentPlatformAsset()
	for _, a := range release.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

// SelfUpdate downloads the asset for the current platform and replaces the running binary.
func SelfUpdate(asset Asset) error {
	// Find current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate current binary: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	fmt.Printf("Downloading %s (%s)...\n", asset.Name, humanSize(asset.Size))

	// Download to a temp file next to the executable
	tmpPath := execPath + ".tmp"
	if err := downloadFile(asset.BrowserDownloadURL, tmpPath); err != nil {
		return err
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to chmod temp binary: %w", err)
	}

	// On Windows we can't replace a running binary directly; rename instead.
	oldPath := execPath + ".old"
	_ = os.Remove(oldPath)

	if err := os.Rename(execPath, oldPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to back up current binary: %w", err)
	}
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Try to restore backup
		_ = os.Rename(oldPath, execPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	_ = os.Remove(oldPath)
	return nil
}

func downloadFile(url, destPath string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to write download: %w", err)
	}
	return nil
}

func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n, _ := strconv.Atoi(parts[i])
		result[i] = n
	}
	return result
}

func humanSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	kb := bytes / 1024
	if kb < 1024 {
		return fmt.Sprintf("%d KB", kb)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/1024/1024)
}
