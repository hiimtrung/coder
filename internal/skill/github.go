package skill

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GitHubSkillEntry represents a single skill in the antigravity skills_index.json.
type GitHubSkillEntry struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Category    string `json:"category"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Risk        string `json:"risk"`
	Source      string `json:"source"`
	DateAdded   string `json:"date_added"`
}

// GitHubFetcher handles downloading skills from GitHub repositories.
type GitHubFetcher struct {
	client *http.Client
}

// NewGitHubFetcher creates a new GitHub skill fetcher.
func NewGitHubFetcher() *GitHubFetcher {
	return &GitHubFetcher{client: &http.Client{}}
}

// FetchSkillIndex downloads and parses the skills_index.json from a repo.
func (f *GitHubFetcher) FetchSkillIndex(repo string) ([]GitHubSkillEntry, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/skills_index.json", repo)

	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skill index from %s: %w", repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch skill index: HTTP %d", resp.StatusCode)
	}

	var entries []GitHubSkillEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to parse skill index: %w", err)
	}

	return entries, nil
}

// FetchSkillMD downloads the SKILL.md content for a given skill path.
func (f *GitHubFetcher) FetchSkillMD(repo string, skillPath string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/%s/SKILL.md", repo, skillPath)

	resp, err := f.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch SKILL.md from %s/%s: %w", repo, skillPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SKILL.md not found at %s: HTTP %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FetchSingleFile downloads a single file from a GitHub repo.
func (f *GitHubFetcher) FetchSingleFile(repo, branch, path string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repo, branch, path)

	resp, err := f.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("file not found: %s (HTTP %d)", path, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FilterSkills filters skills by name list. If names is empty, returns all.
func FilterSkills(entries []GitHubSkillEntry, names []string) []GitHubSkillEntry {
	if len(names) == 0 {
		return entries
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[strings.ToLower(strings.TrimSpace(n))] = true
	}

	var filtered []GitHubSkillEntry
	for _, e := range entries {
		if nameSet[strings.ToLower(e.Name)] || nameSet[strings.ToLower(e.ID)] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
