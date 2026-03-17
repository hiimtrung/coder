package installer

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trungtran/coder/internal/profiles"
	"github.com/trungtran/coder/internal/version"
)

const ManifestPath = ".agents/.coder.json"

// FileSystem is a minimal interface required for installation.
// embed.FS and fs.FS (with ReadDirFS) implement this.
type FileSystem interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

// Manifest records what was installed, enabling update without specifying a profile.
type Manifest struct {
	Version     string    `json:"version"`
	Profile     string    `json:"profile"`
	InstalledAt time.Time `json:"installed_at"`
}

// ReadManifest reads the manifest from targetDir, if present.
func ReadManifest(targetDir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(targetDir, ManifestPath))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Options controls installer behavior.
type Options struct {
	DryRun bool
	Force  bool
}

// Result tracks what was installed.
type Result struct {
	Created []string
	Updated []string
	Skipped []string
}

// Install copies rules, workflows, and agents from the provider FS
// into targetDir according to the given profile.
func Install(srcFS FileSystem, profile profiles.Profile, targetDir string, opts Options) error {
	if opts.DryRun {
		fmt.Printf("DRY RUN — no files will be modified\n\n")
	}
	fmt.Printf("Installing profile \"%s\" → %s\n\n", profile.Name, targetDir)

	result := &Result{}

	fmt.Println("  Installing rules...")
	if err := installRulesFiltered(srcFS, profile.Rules, targetDir, opts, result); err != nil {
		return err
	}

	fmt.Println("  Installing workflows...")
	if err := installWorkflowsFiltered(srcFS, profile.Workflows, targetDir, opts, result); err != nil {
		return err
	}

	fmt.Println("  Installing agents...")
	if err := installAgentVariant(srcFS, profile.AgentFile, targetDir, opts, result); err != nil {
		return err
	}

	fmt.Println("  Installing Claude agents...")
	if err := installClaudeAgentVariant(srcFS, profile.ClaudeAgentFile, targetDir, opts, result); err != nil {
		return err
	}

	if err := generateCopilotInstructions(srcFS, targetDir, opts, result); err != nil {
		return err
	}

	if !opts.DryRun {
		if err := writeManifest(targetDir, profile); err != nil {
			fmt.Printf("  Warning: failed to write manifest: %v\n", err)
		}
	}

	printSummary(result, opts, targetDir)
	return nil
}

// installRulesFiltered copies rule files from .agents/rules/ into targetDir.
// If filter is nil, all files are copied. Otherwise only files in the list are copied.
func installRulesFiltered(srcFS FileSystem, filter []string, targetDir string, opts Options, result *Result) error {
	srcDir := ".agents/rules"
	dstDir := filepath.Join(targetDir, ".agents", "rules")
	if filter == nil {
		return installDir(srcFS, srcDir, dstDir, opts, result, targetDir)
	}
	filterSet := toSet(filter)
	return fs.WalkDir(srcFS, srcDir, func(fsPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == ".DS_Store" {
			return nil
		}
		if !filterSet[d.Name()] {
			return nil
		}
		rel := strings.TrimPrefix(fsPath, srcDir)
		rel = strings.TrimPrefix(rel, "/")
		dstPath := filepath.Join(dstDir, filepath.FromSlash(rel))
		return writeFile(srcFS, fsPath, dstPath, opts, result, targetDir)
	})
}

// installWorkflowsFiltered copies workflow files from .agents/workflows/ into targetDir.
// If filter is nil, all files are copied. Otherwise only files in the list are copied.
func installWorkflowsFiltered(srcFS FileSystem, filter []string, targetDir string, opts Options, result *Result) error {
	srcDir := ".agents/workflows"
	dstDir := filepath.Join(targetDir, ".agents", "workflows")
	if filter == nil {
		return installDir(srcFS, srcDir, dstDir, opts, result, targetDir)
	}
	filterSet := toSet(filter)
	return fs.WalkDir(srcFS, srcDir, func(fsPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == ".DS_Store" {
			return nil
		}
		if !filterSet[d.Name()] {
			return nil
		}
		rel := strings.TrimPrefix(fsPath, srcDir)
		rel = strings.TrimPrefix(rel, "/")
		dstPath := filepath.Join(dstDir, filepath.FromSlash(rel))
		return writeFile(srcFS, fsPath, dstPath, opts, result, targetDir)
	})
}

// installClaudeAgentVariant installs Claude CLI agent files from .claude/agents/ into targetDir.
// If claudeAgentFile is empty, all files are copied as-is.
// If claudeAgentFile is set, only that file is copied keeping its original filename.
func installClaudeAgentVariant(srcFS FileSystem, claudeAgentFile string, targetDir string, opts Options, result *Result) error {
	srcDir := ".claude/agents"
	dstDir := filepath.Join(targetDir, ".claude", "agents")

	if claudeAgentFile == "" {
		return installDir(srcFS, srcDir, dstDir, opts, result, targetDir)
	}

	srcPath := srcDir + "/" + claudeAgentFile
	dstPath := filepath.Join(dstDir, claudeAgentFile)
	return writeFile(srcFS, srcPath, dstPath, opts, result, targetDir)
}

// installAgentVariant installs agent files from .github/agents/ into targetDir.
// If agentFile is empty, all files are copied as-is.
// If agentFile is set, only that file is copied and renamed to coder.agent.md.
func installAgentVariant(srcFS FileSystem, agentFile string, targetDir string, opts Options, result *Result) error {
	srcDir := ".github/agents"
	dstDir := filepath.Join(targetDir, ".github", "agents")

	if agentFile == "" {
		return installDir(srcFS, srcDir, dstDir, opts, result, targetDir)
	}

	srcPath := srcDir + "/" + agentFile
	dstPath := filepath.Join(dstDir, "coder.agent.md")
	return writeFile(srcFS, srcPath, dstPath, opts, result, targetDir)
}

// installDir copies all files from srcDir (in srcFS) to dstDir on the local filesystem.
func installDir(srcFS FileSystem, srcDir, dstDir string, opts Options, result *Result, targetDir string) error {
	return fs.WalkDir(srcFS, srcDir, func(fsPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Skip macOS metadata files
		if d.Name() == ".DS_Store" {
			return nil
		}

		// Compute destination path by stripping the srcDir prefix
		rel := strings.TrimPrefix(fsPath, srcDir)
		rel = strings.TrimPrefix(rel, "/")
		dstPath := filepath.Join(dstDir, filepath.FromSlash(rel))

		return writeFile(srcFS, fsPath, dstPath, opts, result, targetDir)
	})
}

func writeFile(srcFS FileSystem, srcPath, dstPath string, opts Options, result *Result, targetDir string) error {
	_, statErr := os.Stat(dstPath)
	fileExists := statErr == nil

	display, _ := filepath.Rel(targetDir, dstPath)
	if display == "" {
		display = dstPath
	}

	if fileExists && !opts.Force {
		result.Skipped = append(result.Skipped, display)
		return nil
	}

	data, err := srcFS.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", srcPath, err)
	}

	if opts.DryRun {
		if fileExists {
			fmt.Printf("    ~ %s\n", display)
			result.Updated = append(result.Updated, display)
		} else {
			fmt.Printf("    + %s\n", display)
			result.Created = append(result.Created, display)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", dstPath, err)
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", dstPath, err)
	}

	if fileExists {
		fmt.Printf("    ~ %s\n", display)
		result.Updated = append(result.Updated, display)
	} else {
		fmt.Printf("    + %s\n", display)
		result.Created = append(result.Created, display)
	}
	return nil
}

// generateCopilotInstructions reads general.instructions.md from source
// and merges it with the destination's copilot-instructions.md file.
// If destination doesn't exist, creates it. If it does, appends new content.
func generateCopilotInstructions(srcFS FileSystem, targetDir string, opts Options, result *Result) error {
	data, err := srcFS.ReadFile(".agents/rules/general.instructions.md")
	if err != nil {
		return nil // skip if not found
	}

	content := stripFrontmatter(string(data))
	dstPath := filepath.Join(targetDir, ".github", "copilot-instructions.md")
	display, _ := filepath.Rel(targetDir, dstPath)

	fmt.Println("  Merging .github/copilot-instructions.md...")

	if opts.DryRun {
		_, statErr := os.Stat(dstPath)
		if statErr == nil {
			fmt.Printf("    ~ %s (append)\n", display)
			result.Updated = append(result.Updated, display)
		} else {
			fmt.Printf("    + %s\n", display)
			result.Created = append(result.Created, display)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}

	// Check if file exists; if so, append
	existingData, err := os.ReadFile(dstPath)
	var finalContent []byte

	if err == nil {
		// File exists: append new content with separator
		finalContent = append(existingData, []byte("\n\n---\n\n")...)
		finalContent = append(finalContent, []byte(content)...)
		if err := os.WriteFile(dstPath, finalContent, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", dstPath, err)
		}
		fmt.Printf("    ~ %s (append)\n", display)
		result.Updated = append(result.Updated, display)
	} else {
		// File doesn't exist: create new
		if err := os.WriteFile(dstPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", dstPath, err)
		}
		fmt.Printf("    + %s\n", display)
		result.Created = append(result.Created, display)
	}

	return nil
}

// stripFrontmatter removes the YAML frontmatter (--- ... ---) from markdown content.
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	// Find the closing ---
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return content
	}
	return strings.TrimSpace(rest[idx+4:])
}

func writeManifest(targetDir string, profile profiles.Profile) error {
	m := Manifest{
		Version:     version.Version,
		Profile:     profile.Name,
		InstalledAt: time.Now().UTC(),
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	dest := filepath.Join(targetDir, ManifestPath)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

func printSummary(result *Result, opts Options, targetDir string) {
	total := len(result.Created) + len(result.Updated)
	fmt.Println()

	if opts.DryRun {
		fmt.Printf("Summary (dry run): %d to create, %d to update, %d to skip\n",
			len(result.Created), len(result.Updated), len(result.Skipped))
		return
	}

	if len(result.Skipped) > 0 {
		fmt.Printf("Skipped %d existing file(s) — use --force to overwrite\n", len(result.Skipped))
	}
	fmt.Printf("Done! %d file(s) installed to %s\n", total, targetDir)
}

// toSet converts a slice of strings to a set (map[string]bool) for O(1) lookup.
func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
