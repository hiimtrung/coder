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

const (
	globalManifestFile = ".coder/global.json"
	coderBeginPrefix   = "<!-- coder:begin"
	coderEndMarker     = "<!-- coder:end -->"
)

// GlobalManifest tracks globally installed files, enabling update and remove.
type GlobalManifest struct {
	Version      string    `json:"version"`
	Profile      string    `json:"profile"`
	InstalledAt  time.Time `json:"installed_at"`
	ManagedFiles []string  `json:"managed_files"` // owned by coder — deleted on remove
	MergedFiles  []string  `json:"merged_files"`  // coder section injected — section stripped on remove
}

// ReadGlobalManifest reads ~/.coder/global.json.
func ReadGlobalManifest() (*GlobalManifest, error) {
	path, err := globalManifestPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m GlobalManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func globalManifestPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, globalManifestFile), nil
}

// InstallGlobal installs profile files to user-level directories:
//   - ~/.copilot/instructions/               — VS Code Copilot custom instructions
//   - ~/.copilot/agents/                     — VS Code Copilot custom agents
//   - ~/.copilot/chatmodes/                  — VS Code Copilot custom chat modes (workflows)
//   - ~/.claude/rules/                       — Claude Code global rules
//   - ~/.claude/commands/                    — Claude Code user-level slash commands (workflows)
//   - ~/.claude/agents/                      — Claude Code global sub-agents
//   - ~/.claude/CLAUDE.md                    — Claude Code global instructions (merged with markers)
//   - ~/.gemini/antigravity/global_workflows/ — Gemini CLI global workflows
//   - ~/.gemini/GEMINI.md                    — Gemini CLI global rules (merged with markers)
//   - ~/.codex/AGENTS.md                     — OpenAI Codex global agent guidance (merged with markers)
func InstallGlobal(srcFS FileSystem, profile profiles.Profile, opts Options) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	if opts.DryRun {
		fmt.Printf("DRY RUN — no files will be modified\n\n")
	}
	fmt.Printf("Installing profile \"%s\" globally...\n\n", profile.Name)

	result := &Result{}
	var managed, merged []string

	// VS Code Copilot custom instructions
	copilotInstructionsDir := filepath.Join(home, ".copilot", "instructions")
	fmt.Println("  Installing → ~/.copilot/instructions/")
	if err := installGlobalRules(srcFS, profile.Rules, copilotInstructionsDir, opts, result, &managed); err != nil {
		return err
	}

	// VS Code Copilot custom agents
	copilotAgentsDir := filepath.Join(home, ".copilot", "agents")
	fmt.Println("  Installing → ~/.copilot/agents/")
	if err := installGlobalAgents(srcFS, profile.AgentFile, copilotAgentsDir, opts, result, &managed); err != nil {
		return err
	}

	// VS Code Copilot custom chat modes (workflows)
	copilotChatmodesDir := filepath.Join(home, ".copilot", "chatmodes")
	fmt.Println("  Installing → ~/.copilot/chatmodes/")
	if err := installGlobalWorkflows(srcFS, profile.Workflows, copilotChatmodesDir, opts, result, &managed); err != nil {
		return err
	}

	// Claude Code global rules
	claudeRulesDir := filepath.Join(home, ".claude", "rules")
	fmt.Println("  Installing → ~/.claude/rules/")
	if err := installGlobalRules(srcFS, profile.Rules, claudeRulesDir, opts, result, &managed); err != nil {
		return err
	}

	// Claude Code user-level slash commands (workflows)
	claudeCommandsDir := filepath.Join(home, ".claude", "commands")
	fmt.Println("  Installing → ~/.claude/commands/")
	if err := installGlobalWorkflows(srcFS, profile.Workflows, claudeCommandsDir, opts, result, &managed); err != nil {
		return err
	}

	// Claude Code sub-agents
	claudeAgentsDir := filepath.Join(home, ".claude", "agents")
	fmt.Println("  Installing → ~/.claude/agents/")
	if err := installGlobalClaudeAgents(srcFS, profile.ClaudeAgentFile, claudeAgentsDir, opts, result, &managed); err != nil {
		return err
	}

	// Claude Code CLAUDE.md — merged with section markers
	claudeMDPath := filepath.Join(home, ".claude", "CLAUDE.md")
	fmt.Println("  Merging → ~/.claude/CLAUDE.md")
	if err := mergeGlobalClaudeMD(srcFS, profile, claudeMDPath, opts, result); err != nil {
		return err
	}
	if !opts.DryRun {
		merged = append(merged, claudeMDPath)
	}

	// Gemini CLI global workflows
	geminiWorkflowsDir := filepath.Join(home, ".gemini", "antigravity", "global_workflows")
	fmt.Println("  Installing → ~/.gemini/antigravity/global_workflows/")
	if err := installGlobalWorkflows(srcFS, profile.Workflows, geminiWorkflowsDir, opts, result, &managed); err != nil {
		return err
	}

	// Gemini CLI GEMINI.md — merged with section markers
	geminiMDPath := filepath.Join(home, ".gemini", "GEMINI.md")
	fmt.Println("  Merging → ~/.gemini/GEMINI.md")
	if err := mergeGlobalClaudeMD(srcFS, profile, geminiMDPath, opts, result); err != nil {
		return err
	}
	if !opts.DryRun {
		merged = append(merged, geminiMDPath)
	}

	// OpenAI Codex AGENTS.md — merged with section markers
	agentsMDPath := filepath.Join(home, ".codex", "AGENTS.md")
	fmt.Println("  Merging → ~/.codex/AGENTS.md")
	if err := mergeGlobalAgentsMD(srcFS, profile, agentsMDPath, opts, result); err != nil {
		return err
	}
	if !opts.DryRun {
		merged = append(merged, agentsMDPath)
	}

	if !opts.DryRun {
		if err := writeGlobalManifest(home, profile, managed, merged); err != nil {
			fmt.Printf("  Warning: failed to write global manifest: %v\n", err)
		}
	}

	printGlobalSummary(result, opts)
	return nil
}

// RemoveGlobal removes all globally installed files tracked in ~/.coder/global.json.
func RemoveGlobal(opts Options) error {
	manifest, err := ReadGlobalManifest()
	if err != nil {
		return fmt.Errorf("no global install found (run 'coder install global' first): %w", err)
	}

	if opts.DryRun {
		fmt.Printf("DRY RUN — no files will be modified\n\n")
	}
	fmt.Printf("Removing globally installed profile \"%s\"...\n\n", manifest.Profile)

	removed := 0

	for _, path := range manifest.ManagedFiles {
		display := tildePath(path)
		if opts.DryRun {
			fmt.Printf("    - %s\n", display)
			removed++
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			fmt.Printf("  Warning: failed to remove %s: %v\n", display, err)
		} else {
			fmt.Printf("    - %s\n", display)
			removed++
		}
	}

	for _, path := range manifest.MergedFiles {
		display := tildePath(path)
		if opts.DryRun {
			fmt.Printf("    ~ %s (strip coder section)\n", display)
			removed++
			continue
		}
		if err := stripCoderSection(path); err != nil {
			fmt.Printf("  Warning: failed to strip section from %s: %v\n", display, err)
		} else {
			fmt.Printf("    ~ %s (section stripped)\n", display)
			removed++
		}
	}

	if !opts.DryRun {
		if path, err := globalManifestPath(); err == nil {
			os.Remove(path)
		}
		fmt.Printf("\nDone! %d file(s) removed.\n", removed)
	} else {
		fmt.Printf("\nSummary (dry run): %d file(s) would be removed/stripped\n", removed)
	}
	return nil
}

// installGlobalRules copies rule files from .agents/rules/ to dstDir.
// When filter is non-nil, files are read directly by name (avoids fs.WalkDir
// which breaks with GitHubFS because its Stat always returns isDir=false).
// When filter is nil (install all), fs.WalkDir is used — works with embed.FS.
func installGlobalRules(srcFS FileSystem, filter []string, dstDir string, opts Options, result *Result, managed *[]string) error {
	srcDir := ".agents/rules"

	if filter != nil {
		for _, filename := range filter {
			srcPath := srcDir + "/" + filename
			dstPath := filepath.Join(dstDir, filename)
			if err := writeGlobalFile(srcFS, srcPath, dstPath, opts, result); err != nil {
				return err
			}
			if !opts.DryRun {
				*managed = append(*managed, dstPath)
			}
		}
		return nil
	}

	// nil filter — install all files via WalkDir (works with embed.FS)
	return fs.WalkDir(srcFS, srcDir, func(fsPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == ".DS_Store" {
			return nil
		}
		rel := strings.TrimPrefix(fsPath, srcDir+"/")
		dstPath := filepath.Join(dstDir, filepath.FromSlash(rel))
		if err := writeGlobalFile(srcFS, fsPath, dstPath, opts, result); err != nil {
			return err
		}
		if !opts.DryRun {
			*managed = append(*managed, dstPath)
		}
		return nil
	})
}

// installGlobalAgents copies agent files from .github/agents/ to dstDir.
// Keeps original filenames (unlike project install which renames to coder.agent.md)
// so multiple profiles can coexist in the global agents directory.
func installGlobalAgents(srcFS FileSystem, agentFile string, dstDir string, opts Options, result *Result, managed *[]string) error {
	srcDir := ".github/agents"

	if agentFile != "" {
		srcPath := srcDir + "/" + agentFile
		dstPath := filepath.Join(dstDir, agentFile)
		if err := writeGlobalFile(srcFS, srcPath, dstPath, opts, result); err != nil {
			return err
		}
		if !opts.DryRun {
			*managed = append(*managed, dstPath)
		}
		return nil
	}

	return fs.WalkDir(srcFS, srcDir, func(fsPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == ".DS_Store" {
			return nil
		}
		rel := strings.TrimPrefix(fsPath, srcDir+"/")
		dstPath := filepath.Join(dstDir, filepath.FromSlash(rel))
		if err := writeGlobalFile(srcFS, fsPath, dstPath, opts, result); err != nil {
			return err
		}
		if !opts.DryRun {
			*managed = append(*managed, dstPath)
		}
		return nil
	})
}

// installGlobalClaudeAgents copies Claude CLI agent files from .claude/agents/ to dstDir.
// Keeps original filenames so multiple profiles can coexist in the global agents directory.
func installGlobalClaudeAgents(srcFS FileSystem, claudeAgentFile string, dstDir string, opts Options, result *Result, managed *[]string) error {
	srcDir := ".claude/agents"

	if claudeAgentFile != "" {
		srcPath := srcDir + "/" + claudeAgentFile
		dstPath := filepath.Join(dstDir, claudeAgentFile)
		if err := writeGlobalFile(srcFS, srcPath, dstPath, opts, result); err != nil {
			return err
		}
		if !opts.DryRun {
			*managed = append(*managed, dstPath)
		}
		return nil
	}

	return fs.WalkDir(srcFS, srcDir, func(fsPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == ".DS_Store" {
			return nil
		}
		rel := strings.TrimPrefix(fsPath, srcDir+"/")
		dstPath := filepath.Join(dstDir, filepath.FromSlash(rel))
		if err := writeGlobalFile(srcFS, fsPath, dstPath, opts, result); err != nil {
			return err
		}
		if !opts.DryRun {
			*managed = append(*managed, dstPath)
		}
		return nil
	})
}

// installGlobalWorkflows copies workflow files from .agents/workflows/ to dstDir.
// When filter is non-nil, files are read directly by name; otherwise all files are walked.
func installGlobalWorkflows(srcFS FileSystem, filter []string, dstDir string, opts Options, result *Result, managed *[]string) error {
	srcDir := ".agents/workflows"

	if filter != nil {
		for _, filename := range filter {
			srcPath := srcDir + "/" + filename
			dstPath := filepath.Join(dstDir, filename)
			if err := writeGlobalFile(srcFS, srcPath, dstPath, opts, result); err != nil {
				return err
			}
			if !opts.DryRun {
				*managed = append(*managed, dstPath)
			}
		}
		return nil
	}

	return fs.WalkDir(srcFS, srcDir, func(fsPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == ".DS_Store" {
			return nil
		}
		rel := strings.TrimPrefix(fsPath, srcDir+"/")
		dstPath := filepath.Join(dstDir, filepath.FromSlash(rel))
		if err := writeGlobalFile(srcFS, fsPath, dstPath, opts, result); err != nil {
			return err
		}
		if !opts.DryRun {
			*managed = append(*managed, dstPath)
		}
		return nil
	})
}

// mergeGlobalClaudeMD injects or updates the coder-managed section in ~/.claude/CLAUDE.md.
// The section is wrapped with <!-- coder:begin ... --> / <!-- coder:end --> markers,
// allowing it to be precisely located for future updates and removal.
func mergeGlobalClaudeMD(srcFS FileSystem, profile profiles.Profile, dstPath string, opts Options, result *Result) error {
	data, err := srcFS.ReadFile(".agents/rules/general.instructions.md")
	if err != nil {
		return nil // skip if source not found
	}
	content := stripFrontmatter(string(data))

	section := fmt.Sprintf("%s profile=%s version=%s -->\n%s\n%s",
		coderBeginPrefix, profile.Name, version.Version, content, coderEndMarker)

	display := tildePath(dstPath)

	if opts.DryRun {
		if _, statErr := os.Stat(dstPath); statErr == nil {
			fmt.Printf("    ~ %s (update coder section)\n", display)
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

	existing, err := os.ReadFile(dstPath)
	if err != nil {
		// File doesn't exist: create with just the coder section
		if err := os.WriteFile(dstPath, []byte(section+"\n"), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", dstPath, err)
		}
		fmt.Printf("    + %s\n", display)
		result.Created = append(result.Created, display)
		return nil
	}

	updated := replaceOrAppendCoderSection(string(existing), section)
	if err := os.WriteFile(dstPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", dstPath, err)
	}
	fmt.Printf("    ~ %s\n", display)
	result.Updated = append(result.Updated, display)
	return nil
}

// mergeGlobalAgentsMD injects or updates the coder-managed section in ~/.codex/AGENTS.md.
// It reads the profile's agent file (.github/agents/<agentFile>), strips YAML frontmatter,
// and wraps the content with <!-- coder:begin ... --> / <!-- coder:end --> markers so that
// future `coder update global` calls can precisely locate and refresh the section, and
// `coder remove global` can cleanly strip it.
func mergeGlobalAgentsMD(srcFS FileSystem, profile profiles.Profile, dstPath string, opts Options, result *Result) error {
	agentFile := profile.AgentFile
	if agentFile == "" {
		agentFile = "coder.agent.md"
	}
	srcPath := ".github/agents/" + agentFile

	data, err := srcFS.ReadFile(srcPath)
	if err != nil {
		return nil // source not found — skip silently
	}
	content := stripFrontmatter(string(data))

	section := fmt.Sprintf("%s profile=%s version=%s -->\n%s\n%s",
		coderBeginPrefix, profile.Name, version.Version, content, coderEndMarker)

	display := tildePath(dstPath)

	if opts.DryRun {
		if _, statErr := os.Stat(dstPath); statErr == nil {
			fmt.Printf("    ~ %s (update coder section)\n", display)
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

	existing, err := os.ReadFile(dstPath)
	if err != nil {
		// File doesn't exist: create with just the coder section
		if err := os.WriteFile(dstPath, []byte(section+"\n"), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", dstPath, err)
		}
		fmt.Printf("    + %s\n", display)
		result.Created = append(result.Created, display)
		return nil
	}

	updated := replaceOrAppendCoderSection(string(existing), section)
	if err := os.WriteFile(dstPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", dstPath, err)
	}
	fmt.Printf("    ~ %s\n", display)
	result.Updated = append(result.Updated, display)
	return nil
}

// replaceOrAppendCoderSection replaces an existing coder section or appends a new one.
func replaceOrAppendCoderSection(existing, section string) string {
	beginIdx := strings.Index(existing, coderBeginPrefix)
	endIdx := strings.Index(existing, coderEndMarker)
	if beginIdx >= 0 && endIdx > beginIdx {
		before := strings.TrimRight(existing[:beginIdx], "\n")
		after := strings.TrimLeft(existing[endIdx+len(coderEndMarker):], "\n")
		if before != "" {
			return before + "\n\n" + section + "\n\n" + after
		}
		return section + "\n\n" + after
	}
	return strings.TrimRight(existing, "\n") + "\n\n" + section + "\n"
}

// stripCoderSection removes the <!-- coder:begin ... --> ... <!-- coder:end --> block from a file.
func stripCoderSection(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content := string(data)
	beginIdx := strings.Index(content, coderBeginPrefix)
	endIdx := strings.Index(content, coderEndMarker)
	if beginIdx < 0 || endIdx < beginIdx {
		return nil // nothing to strip
	}

	before := strings.TrimRight(content[:beginIdx], "\n")
	after := strings.TrimLeft(content[endIdx+len(coderEndMarker):], "\n")

	var result string
	switch {
	case before != "" && after != "":
		result = before + "\n\n" + after + "\n"
	case before != "":
		result = before + "\n"
	case after != "":
		result = after + "\n"
	default:
		return os.Remove(path)
	}
	return os.WriteFile(path, []byte(result), 0o644)
}

func writeGlobalFile(srcFS FileSystem, srcPath, dstPath string, opts Options, result *Result) error {
	_, statErr := os.Stat(dstPath)
	fileExists := statErr == nil
	display := tildePath(dstPath)

	if fileExists && !opts.Force {
		result.Skipped = append(result.Skipped, display)
		return nil
	}

	data, err := srcFS.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", srcPath, err)
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

func writeGlobalManifest(home string, profile profiles.Profile, managed, merged []string) error {
	m := GlobalManifest{
		Version:      version.Version,
		Profile:      profile.Name,
		InstalledAt:  time.Now().UTC(),
		ManagedFiles: managed,
		MergedFiles:  merged,
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(home, globalManifestFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func printGlobalSummary(result *Result, opts Options) {
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
	fmt.Printf("Done! %d file(s) installed globally.\n", total)
}

// tildePath replaces the home directory prefix with ~.
func tildePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
