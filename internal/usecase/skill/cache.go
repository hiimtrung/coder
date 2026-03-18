package ucskill

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	skilldomain "github.com/trungtran/coder/internal/domain/skill"
)

// cacheManifestEntry tracks a skill's cache state.
type cacheManifestEntry struct {
	Version   string    `json:"version"`
	CachedAt  time.Time `json:"cached_at"`
	FileCount int       `json:"file_count"`
}

// CacheManager manages skill script/data files in ~/.coder/cache/<skill>/.
// It operates through the skilldomain.SkillClient interface (gRPC or HTTP) so no direct
// database connection is required — the same transport used by other skill commands.
type CacheManager struct {
	client   skilldomain.SkillClient
	cacheDir string // defaults to ~/.coder/cache
}

// NewCacheManager creates a CacheManager backed by a skilldomain.SkillClient.
func NewCacheManager(client skilldomain.SkillClient) *CacheManager {
	home, _ := os.UserHomeDir()
	return &CacheManager{
		client:   client,
		cacheDir: filepath.Join(home, ".coder", "cache"),
	}
}

// SkillCacheDir returns the directory for a specific skill: ~/.coder/cache/<skillName>.
func (c *CacheManager) SkillCacheDir(skillName string) string {
	return filepath.Join(c.cacheDir, skillName)
}

// IsCached returns true if the skill is already cached at the given version.
func (c *CacheManager) IsCached(skillName, version string) bool {
	manifest := c.loadManifest()
	entry, ok := manifest[skillName]
	if !ok {
		return false
	}
	return entry.Version == version && entry.FileCount > 0
}

// EnsureCached checks the cache; extracts from remote if missing or stale.
// Returns the cache directory path.
func (c *CacheManager) EnsureCached(ctx context.Context, skillName string) (string, error) {
	sk, _, err := c.client.GetSkill(ctx, skillName)
	if err != nil {
		return "", fmt.Errorf("skill %q not found: %w", skillName, err)
	}

	if c.IsCached(skillName, sk.Version) {
		return c.SkillCacheDir(skillName), nil
	}

	return c.Pull(ctx, skillName)
}

// Pull force-extracts all files for a skill to ~/.coder/cache/<skillName>/.
func (c *CacheManager) Pull(ctx context.Context, skillName string) (string, error) {
	sk, _, err := c.client.GetSkill(ctx, skillName)
	if err != nil {
		return "", fmt.Errorf("skill %q not found: %w", skillName, err)
	}

	files, err := c.client.GetSkillFiles(ctx, skillName)
	if err != nil {
		return "", fmt.Errorf("failed to get files for %q: %w", skillName, err)
	}
	if len(files) == 0 {
		return "", fmt.Errorf("skill %q has no stored files (run: coder skill ingest --source local --include-files)", skillName)
	}

	destDir := c.SkillCacheDir(skillName)

	// Clear old cache for this skill
	if err := os.RemoveAll(destDir); err != nil {
		return "", fmt.Errorf("failed to clear cache dir: %w", err)
	}

	for _, f := range files {
		dest := filepath.Join(destDir, f.RelPath)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return "", err
		}
		perm := os.FileMode(0644)
		if f.ContentType == "text/x-python" || f.ContentType == "text/x-sh" {
			perm = 0755
		}
		if err := os.WriteFile(dest, f.Content, perm); err != nil {
			return "", fmt.Errorf("failed to write %s: %w", f.RelPath, err)
		}
	}

	c.updateManifest(skillName, sk.Version, len(files))
	return destDir, nil
}

// PullAll extracts all skills that have stored files.
func (c *CacheManager) PullAll(ctx context.Context) (int, int, error) {
	skills, err := c.client.ListSkills(ctx, "", "", 1000, 0)
	if err != nil {
		return 0, 0, err
	}

	ok, fail := 0, 0
	for _, sk := range skills {
		// Check if this skill has any files before attempting to pull.
		files, err := c.client.GetSkillFiles(ctx, sk.Name)
		if err != nil || len(files) == 0 {
			continue // no files stored — skip silently
		}
		if _, err := c.Pull(ctx, sk.Name); err != nil {
			fail++
		} else {
			ok++
		}
	}
	return ok, fail, nil
}

// Clear removes the cache for a single skill (or all if skillName == "").
func (c *CacheManager) Clear(skillName string) error {
	if skillName == "" {
		manifest := c.loadManifest()
		for name := range manifest {
			os.RemoveAll(c.SkillCacheDir(name))
		}
		return os.WriteFile(c.manifestPath(), []byte("{}"), 0644)
	}
	os.RemoveAll(c.SkillCacheDir(skillName))
	manifest := c.loadManifest()
	delete(manifest, skillName)
	return c.saveManifest(manifest)
}

// ListCached returns all skills currently in cache with their entry info.
func (c *CacheManager) ListCached() map[string]cacheManifestEntry {
	return c.loadManifest()
}

// ── manifest helpers ──────────────────────────────────────────────────────────

func (c *CacheManager) manifestPath() string {
	return filepath.Join(c.cacheDir, "cache.json")
}

func (c *CacheManager) loadManifest() map[string]cacheManifestEntry {
	data, err := os.ReadFile(c.manifestPath())
	if err != nil {
		return make(map[string]cacheManifestEntry)
	}
	var m map[string]cacheManifestEntry
	if err := json.Unmarshal(data, &m); err != nil {
		return make(map[string]cacheManifestEntry)
	}
	return m
}

func (c *CacheManager) updateManifest(skillName, version string, fileCount int) {
	manifest := c.loadManifest()
	manifest[skillName] = cacheManifestEntry{
		Version:   version,
		CachedAt:  time.Now(),
		FileCount: fileCount,
	}
	c.saveManifest(manifest)
}

func (c *CacheManager) saveManifest(m map[string]cacheManifestEntry) error {
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.manifestPath(), data, 0644)
}
