package installer

import (
	"bytes"
	"io/fs"
	"time"

	"github.com/trungtran/coder/internal/skill"
)

// GitHubFS implements fs.FS, fs.ReadDirFS, and fs.ReadFileFS for a GitHub repository.
type GitHubFS struct {
	repo    string
	branch  string
	fetcher *skill.GitHubFetcher
}

// NewGitHubFS creates a new GitHubFS.
func NewGitHubFS(repo, branch string) *GitHubFS {
	if branch == "" {
		branch = "main"
	}
	return &GitHubFS{
		repo:    repo,
		branch:  branch,
		fetcher: skill.NewGitHubFetcher(),
	}
}

// Open implements fs.FS.
func (g *GitHubFS) Open(name string) (fs.File, error) {
	// For simplicity, we implement a read-only file that just holds content.
	// This is primarily to satisfy the interface if WalkDir calls it.
	info, err := g.Stat(name)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return &githubFile{info: info}, nil
	}

	data, err := g.ReadFile(name)
	if err != nil {
		return nil, err
	}

	return &githubFile{info: info, data: data}, nil
}

// Stat implements fs.StatFS.
func (g *GitHubFS) Stat(name string) (fs.FileInfo, error) {
	// This is a bit expensive as it might need to call GitHub API.
	// For now, we assume everything is a file unless it's a known directory.
	// In a real FS, we'd cache this or check.
	// Since installer usually calls ReadDir first, paths are often known.
	return &githubFileInfo{name: name, isDir: false}, nil
}

// ReadFile implements fs.ReadFileFS.
func (g *GitHubFS) ReadFile(name string) ([]byte, error) {
	content, err := g.fetcher.FetchSingleFile(g.repo, g.branch, name)
	if err != nil {
		return nil, err
	}
	return []byte(content), nil
}

// ReadDir implements fs.ReadDirFS.
func (g *GitHubFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := g.fetcher.ListEntries(g.repo, g.branch, name)
	if err != nil {
		return nil, err
	}

	var fsEntries []fs.DirEntry
	for _, e := range entries {
		fsEntries = append(fsEntries, &githubDirEntry{
			name:  e.Name,
			isDir: e.Type == "dir",
		})
	}
	return fsEntries, nil
}

// githubDirEntry implements fs.DirEntry.
type githubDirEntry struct {
	name  string
	isDir bool
}

func (d *githubDirEntry) Name() string               { return d.name }
func (d *githubDirEntry) IsDir() bool                { return d.isDir }
func (d *githubDirEntry) Type() fs.FileMode          { 
	if d.isDir {
		return fs.ModeDir
	}
	return 0 
}
func (d *githubDirEntry) Info() (fs.FileInfo, error) { return &githubFileInfo{name: d.name, isDir: d.isDir}, nil }

type githubFile struct {
	info   fs.FileInfo
	data   []byte
	reader *bytes.Reader
}

func (f *githubFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *githubFile) Read(b []byte) (int, error) {
	if f.info.IsDir() {
		return 0, &fs.PathError{Op: "read", Path: f.info.Name(), Err: fs.ErrInvalid}
	}
	if f.reader == nil {
		f.reader = bytes.NewReader(f.data)
	}
	return f.reader.Read(b)
}
func (f *githubFile) Close() error { return nil }

type githubFileInfo struct {
	name  string
	isDir bool
}

func (i *githubFileInfo) Name() string       { return i.name }
func (i *githubFileInfo) Size() int64        { return 0 }
func (i *githubFileInfo) Mode() fs.FileMode  { 
	if i.isDir {
		return fs.ModeDir
	}
	return 0 
}
func (i *githubFileInfo) ModTime() time.Time { return time.Now() }
func (i *githubFileInfo) IsDir() bool        { return i.isDir }
func (i *githubFileInfo) Sys() interface{}   { return nil }
