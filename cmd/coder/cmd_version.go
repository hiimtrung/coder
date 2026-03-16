package main

import (
	"fmt"
	"os"

	"github.com/trungtran/coder/internal/updater"
	"github.com/trungtran/coder/internal/version"
)

func printVersion() {
	fmt.Printf("coder %s\n", version.Version)
	fmt.Printf("  commit:     %s\n", version.Commit)
	fmt.Printf("  build date: %s\n", version.BuildDate)
	fmt.Printf("  releases:   %s\n", version.GitHubReleasesURL())
}

func runCheckUpdate() {
	fmt.Printf("Checking for updates (current: %s)...\n", version.Version)
	release, err := updater.CheckLatestRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	latest := release.TagName
	if updater.IsNewer(version.Version, latest) {
		fmt.Printf("\n✓ New version available: %s → %s\n", version.Version, latest)
		fmt.Printf("  Run 'coder self-update' to upgrade.\n")
		fmt.Printf("  Or download manually: %s\n", release.HTMLURL)
	} else {
		fmt.Printf("✓ You are up to date (%s)\n", version.Version)
	}
}

func runSelfUpdate() {
	fmt.Printf("Checking for updates (current: %s)...\n", version.Version)
	release, err := updater.CheckLatestRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	latest := release.TagName
	if !updater.IsNewer(version.Version, latest) {
		fmt.Printf("✓ Already up to date (%s)\n", version.Version)
		return
	}

	fmt.Printf("New version: %s → %s\n", version.Version, latest)

	asset, ok := updater.FindAsset(release)
	if !ok {
		fmt.Fprintf(os.Stderr,
			"Error: no binary found for your platform (%s)\n"+
				"Download manually: %s\n",
			updater.CurrentPlatformAsset(), release.HTMLURL)
		os.Exit(1)
	}

	if err := updater.SelfUpdate(asset); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Updated to %s. Run 'coder version' to verify.\n", latest)
}
