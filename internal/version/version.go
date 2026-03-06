package version

// Build-time variables set via ldflags:
//
//	go build -ldflags "-X github.com/trungtran/coder/internal/version.Version=1.0.0 ..."
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

const (
	RepoOwner  = "takayatomose"
	RepoName   = "coder-skills"
	BinaryName = "coder"
)

// GitHubReleasesURL returns the GitHub releases page URL.
func GitHubReleasesURL() string {
	return "https://github.com/" + RepoOwner + "/" + RepoName + "/releases"
}

// GitHubAPILatestURL returns the GitHub API URL for the latest release.
func GitHubAPILatestURL() string {
	return "https://api.github.com/repos/" + RepoOwner + "/" + RepoName + "/releases/latest"
}
