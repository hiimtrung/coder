package installer

import (
	"embed"
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

// Manifest records what was installed, enabling update without specifying a profile.
type Manifest struct {
	Version     string    `json:"version"`
	Profile     string    `json:"profile"`
	Skills      []string  `json:"skills"`
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

// Install copies skills, rules, workflows, and agents from the embedded FS
// into targetDir according to the given profile.
func Install(agentFS embed.FS, profile profiles.Profile, targetDir string, opts Options) error {
	if opts.DryRun {
		fmt.Printf("DRY RUN — no files will be modified\n\n")
	}
	fmt.Printf("Installing profile \"%s\" → %s\n\n", profile.Name, targetDir)

	result := &Result{}

	if err := installSkills(agentFS, profile, targetDir, opts, result); err != nil {
		return err
	}
	if profile.IncludeRules {
		fmt.Println("  Installing rules...")
		if err := installDir(agentFS, ".agents/rules", filepath.Join(targetDir, ".agents", "rules"), opts, result, targetDir); err != nil {
			return err
		}
	}
	if profile.IncludeWorkflows {
		fmt.Println("  Installing workflows...")
		if err := installDir(agentFS, ".agents/workflows", filepath.Join(targetDir, ".agents", "workflows"), opts, result, targetDir); err != nil {
			return err
		}
	}
	if profile.IncludeAgents {
		fmt.Println("  Installing agents...")
		if err := installDir(agentFS, ".github/agents", filepath.Join(targetDir, ".github", "agents"), opts, result, targetDir); err != nil {
			return err
		}
	}

	if err := generateCopilotInstructions(agentFS, targetDir, opts, result); err != nil {
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

func installSkills(agentFS embed.FS, profile profiles.Profile, targetDir string, opts Options, result *Result) error {
	fmt.Println("  Installing skills...")

	skillNames := profile.Skills
	if len(skillNames) == 1 && skillNames[0] == "all" {
		entries, err := agentFS.ReadDir(".agents/skills")
		if err != nil {
			return fmt.Errorf("failed to read embedded skills: %w", err)
		}
		skillNames = nil
		for _, e := range entries {
			if e.IsDir() {
				skillNames = append(skillNames, e.Name())
			}
		}
	}

	for _, skill := range skillNames {
		srcPath := ".agents/skills/" + skill
		dstPath := filepath.Join(targetDir, ".agents", "skills", skill)
		if err := installDir(agentFS, srcPath, dstPath, opts, result, targetDir); err != nil {
			return fmt.Errorf("failed to install skill %q: %w", skill, err)
		}
	}
	return nil
}

// installDir copies all files from srcDir (in agentFS) to dstDir on the local filesystem.
func installDir(agentFS embed.FS, srcDir, dstDir string, opts Options, result *Result, targetDir string) error {
	return fs.WalkDir(agentFS, srcDir, func(fsPath string, d fs.DirEntry, err error) error {
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

		return writeFile(agentFS, fsPath, dstPath, opts, result, targetDir)
	})
}

func writeFile(agentFS embed.FS, srcPath, dstPath string, opts Options, result *Result, targetDir string) error {
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

	data, err := agentFS.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded file %s: %w", srcPath, err)
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

// generateCopilotInstructions reads general.instructions.md from embedded files
// and merges it with the destination's copilot-instructions.md file.
// If destination doesn't exist, creates it. If it does, appends new content.
func generateCopilotInstructions(agentFS embed.FS, targetDir string, opts Options, result *Result) error {
	data, err := agentFS.ReadFile(".agents/rules/general.instructions.md")
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

func writeFileContent(data []byte, dstPath, display string, opts Options, result *Result) error {
	_, statErr := os.Stat(dstPath)
	fileExists := statErr == nil

	if fileExists && !opts.Force {
		result.Skipped = append(result.Skipped, display)
		return nil
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
		return err
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return err
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

func writeManifest(targetDir string, profile profiles.Profile) error {
	skills := profile.Skills
	if len(skills) == 1 && skills[0] == "all" {
		skills = profiles.AllSkills
	}
	m := Manifest{
		Version:     version.Version,
		Profile:     profile.Name,
		Skills:      skills,
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
