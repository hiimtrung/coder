package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	tasagent "github.com/trungtran/coder"
	"github.com/trungtran/coder/internal/installer"
	"github.com/trungtran/coder/internal/profiles"
	"github.com/trungtran/coder/internal/version"
)

func runInstall(args []string) {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "Error: profile argument required")
		fmt.Fprintln(os.Stderr, "Usage: coder install <profile> [flags]")
		fmt.Fprintln(os.Stderr, "       coder install global [profile] [flags]")
		fmt.Fprintln(os.Stderr, "Run 'coder list' to see available profiles")
		os.Exit(1)
	}

	if args[0] == "global" {
		runInstallGlobal(args[1:])
		return
	}

	fs := flag.NewFlagSet("install", flag.ExitOnError)
	target := fs.String("target", "", "Target directory (default: current directory)")
	fs.StringVar(target, "t", "", "Target directory (shorthand)")
	force := fs.Bool("force", false, "Overwrite existing files")
	fs.BoolVar(force, "f", false, "Overwrite existing files (shorthand)")
	dryRun := fs.Bool("dry-run", false, "Show what would be installed without making changes")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder install <profile> [flags]")
		fs.PrintDefaults()
	}

	profileName := args[0]
	if err := fs.Parse(args[1:]); err != nil {
		os.Exit(1)
	}

	targetDir := resolveTargetDir(*target)
	profile, err := profiles.Get(profileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := installer.Options{DryRun: *dryRun, Force: *force}

	// Remote-first strategy: try GitHub first, fallback to embedded
	fmt.Printf("Fetching latest engine components from GitHub (%s/%s)...\n", version.RepoOwner, version.RepoName)
	remoteFS := installer.NewGitHubFS(version.RepoOwner+"/"+version.RepoName, "main")

	err = installer.Install(remoteFS, profile, targetDir, opts)
	if err != nil {
		fmt.Printf("  ⚠ Remote fetch failed: %v\n", err)
		fmt.Println("  Falling back to embedded engine components...")
		if err := installer.Install(tasagent.AgentFS, profile, targetDir, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func runInstallGlobal(args []string) {
	fs := flag.NewFlagSet("install global", flag.ExitOnError)
	force := fs.Bool("force", false, "Overwrite existing files")
	fs.BoolVar(force, "f", false, "Overwrite existing files (shorthand)")
	dryRun := fs.Bool("dry-run", false, "Show what would be installed without making changes")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder install global [profile] [flags]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Installs AI agent configs to user-level global directories:")
		fmt.Fprintln(os.Stderr, "  ~/.copilot/instructions/                VS Code Copilot custom instructions")
		fmt.Fprintln(os.Stderr, "  ~/.copilot/agents/                      VS Code Copilot custom agents")
		fmt.Fprintln(os.Stderr, "  ~/.copilot/chatmodes/                   VS Code Copilot custom chat modes (workflows)")
		fmt.Fprintln(os.Stderr, "  ~/.claude/rules/                        Claude Code global rules")
		fmt.Fprintln(os.Stderr, "  ~/.claude/commands/                     Claude Code user-level slash commands (workflows)")
		fmt.Fprintln(os.Stderr, "  ~/.claude/CLAUDE.md                     Claude Code global instructions")
		fmt.Fprintln(os.Stderr, "  ~/.gemini/antigravity/global_workflows/  Gemini CLI global workflows")
		fmt.Fprintln(os.Stderr, "  ~/.gemini/GEMINI.md                     Gemini CLI global rules")
		fmt.Fprintln(os.Stderr, "")
		fs.PrintDefaults()
	}

	// Optional profile arg before flags
	profileName := "fullstack"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		profileName = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	profile, err := profiles.Get(profileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := installer.Options{DryRun: *dryRun, Force: *force}

	fmt.Printf("Fetching latest engine components from GitHub (%s/%s)...\n", version.RepoOwner, version.RepoName)
	remoteFS := installer.NewGitHubFS(version.RepoOwner+"/"+version.RepoName, "main")

	err = installer.InstallGlobal(remoteFS, profile, opts)
	if err != nil {
		fmt.Printf("  ⚠ Remote fetch failed: %v\n", err)
		fmt.Println("  Falling back to embedded engine components...")
		if err := installer.InstallGlobal(tasagent.AgentFS, profile, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func runRemove(args []string) {
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "Error: subcommand required")
		fmt.Fprintln(os.Stderr, "Usage: coder remove global [flags]")
		os.Exit(1)
	}

	if args[0] == "global" {
		runRemoveGlobal(args[1:])
		return
	}

	fmt.Fprintf(os.Stderr, "Error: unknown remove subcommand %q\n", args[0])
	fmt.Fprintln(os.Stderr, "Usage: coder remove global [flags]")
	os.Exit(1)
}

func runRemoveGlobal(args []string) {
	fs := flag.NewFlagSet("remove global", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Show what would be removed without making changes")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder remove global [flags]")
		fmt.Fprintln(os.Stderr, "Removes all globally installed files tracked in ~/.coder/global.json")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	opts := installer.Options{DryRun: *dryRun}
	if err := installer.RemoveGlobal(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	target := fs.String("target", "", "Target directory (default: current directory)")
	fs.StringVar(target, "t", "", "Target directory (shorthand)")
	dryRun := fs.Bool("dry-run", false, "Show what would be updated without making changes")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder update [profile] [flags]")
		fs.PrintDefaults()
	}

	// Profile is optional for update — read from manifest if omitted.
	// "global" is a special keyword that updates the global install.
	var profileName string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		profileName = args[0]
		args = args[1:]
	}

	if profileName == "global" {
		if err := fs.Parse(args); err != nil {
			os.Exit(1)
		}
		// Read profile from global manifest
		manifest, err := installer.ReadGlobalManifest()
		if err != nil {
			fmt.Fprintf(os.Stderr,
				"Error: no global install found — run 'coder install global <profile>' first\n")
			os.Exit(1)
		}
		globalProfile, err := profiles.Get(manifest.Profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		opts := installer.Options{DryRun: *dryRun, Force: true}
		fmt.Printf("Fetching latest engine components from GitHub (%s/%s)...\n", version.RepoOwner, version.RepoName)
		remoteFS := installer.NewGitHubFS(version.RepoOwner+"/"+version.RepoName, "main")
		err = installer.InstallGlobal(remoteFS, globalProfile, opts)
		if err != nil {
			fmt.Printf("  ⚠ Remote fetch failed: %v\n", err)
			fmt.Println("  Falling back to embedded engine components...")
			if err := installer.InstallGlobal(tasagent.AgentFS, globalProfile, opts); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
		return
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	targetDir := resolveTargetDir(*target)

	if profileName == "" {
		// Read profile from manifest
		manifest, err := installer.ReadManifest(targetDir)
		if err != nil {
			fmt.Fprintf(os.Stderr,
				"Error: no profile specified and no manifest found in %s\n"+
					"Run 'coder install <profile>' first, or specify a profile: coder update <profile>\n",
				targetDir)
			os.Exit(1)
		}
		profileName = manifest.Profile
		fmt.Printf("Using profile from manifest: %s (installed %s)\n\n",
			manifest.Profile, manifest.InstalledAt.Format("2006-01-02"))
	}

	profile, err := profiles.Get(profileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Update always overwrites
	opts := installer.Options{DryRun: *dryRun, Force: true}

	// Remote-first strategy: try GitHub first, fallback to embedded
	fmt.Printf("Fetching latest engine components from GitHub (%s/%s)...\n", version.RepoOwner, version.RepoName)
	remoteFS := installer.NewGitHubFS(version.RepoOwner+"/"+version.RepoName, "main")

	// Try updating from remote
	err = installer.Install(remoteFS, profile, targetDir, opts)
	if err != nil {
		fmt.Printf("  ⚠ Remote update failed: %v\n", err)
		fmt.Println("  Falling back to embedded engine components...")
		// Fallback to embedded AgentFS
		if err := installer.Install(tasagent.AgentFS, profile, targetDir, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	if !*dryRun {
		// Hook: Automatically ingest local skills into the vector DB
		fmt.Println("Syncing skills to vector database...")

		// Run ingestion synchronously since local ingestion is fast.
		client := getSkillClient()
		defer client.Close()
		runIngestAuto(context.Background(), client, "", false)
	}
}

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() > 0 {
		profile, err := profiles.Get(fs.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		profiles.PrintProfile(profile)
	} else {
		profiles.PrintAll()
	}
}
