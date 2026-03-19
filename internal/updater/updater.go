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

// tmpDir returns ~/.coder — a user-writable directory used for staging downloads.
// This avoids permission errors when the installed binary lives in a system path
// like /usr/local/bin that requires root to write.
func tmpDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".coder")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create ~/.coder: %w", err)
	}
	return dir, nil
}

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
// The download is staged in ~/.coder/ so it always succeeds even when the installed binary
// lives in a system directory (e.g. /usr/local/bin) that requires elevated permissions.
func SelfUpdate(asset Asset) error {
	// Resolve the path of the running binary.
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate current binary: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Stage download in ~/.coder/ — always user-writable regardless of where
	// the binary is installed.
	stageDir, err := tmpDir()
	if err != nil {
		return err
	}
	tmpPath := filepath.Join(stageDir, version.BinaryName+".tmp")
	// Always clean up the temp file when we're done.
	defer os.Remove(tmpPath)

	fmt.Printf("Downloading %s (%s)...\n", asset.Name, humanSize(asset.Size))
	if err := downloadFile(asset.BrowserDownloadURL, tmpPath); err != nil {
		return err
	}

	// Make the downloaded binary executable before moving it into place.
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("failed to chmod downloaded binary: %w", err)
	}

	// Replace the running binary.
	if err := replaceBinary(tmpPath, execPath); err != nil {
		return err
	}
	return nil
}

// replaceBinary atomically swaps src into dst.
//
// Strategy:
//  1. Back up dst → dst.old (same directory, same filesystem — fast rename).
//  2. Try os.Rename(src, dst).  This works when src and dst are on the same
//     filesystem.  If they are on different filesystems (e.g. src is on the
//     user's home partition, dst is on /usr/local), os.Rename returns an
//     "invalid cross-device link" error; in that case we fall back to a
//     byte-level copy so the caller only needs write access to dst's directory.
//  3. On any failure after the backup, the old binary is restored.
func replaceBinary(src, dst string) error {
	oldPath := dst + ".old"
	_ = os.Remove(oldPath)

	// Back up the current binary (requires write permission on dst's directory).
	if err := os.Rename(dst, oldPath); err != nil {
		return fmt.Errorf("failed to back up current binary (try running with sudo): %w", err)
	}

	// Attempt a rename — fast path for same-filesystem installs.
	if err := os.Rename(src, dst); err != nil {
		// Cross-device: fall back to copy.
		if copyErr := copyFile(src, dst); copyErr != nil {
			// Restore backup so the user isn't left without a binary.
			_ = os.Rename(oldPath, dst)
			return fmt.Errorf("failed to replace binary (try running with sudo): %w", copyErr)
		}
	}

	_ = os.Remove(oldPath)
	return nil
}

// copyFile copies src to dst byte-for-byte, preserving permissions.
// Used as a cross-filesystem fallback when os.Rename returns EXDEV.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
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
