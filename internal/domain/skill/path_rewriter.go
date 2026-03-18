package skill

import "strings"

// knownPathPrefixes lists all path patterns that may appear in SKILL.md bash blocks
// referring to a skill's own files, ordered from most-specific to least-specific.
var knownPathPrefixes = []string{
	// absolute: ~/.claude/skills/{name}/
	"~/.claude/skills/",
	// absolute: ~/.agents/skills/{name}/
	"~/.agents/skills/",
	// relative with agents prefix: .agents/skills/{name}/
	".agents/skills/",
	// bare relative: skills/{name}/
	"skills/",
}

// RewriteSkillPaths rewrites all script/data path references inside a chunk's markdown content
// so they point to the local cache directory ~/.coder/cache/<skillName>/.
//
// Examples:
//
//	"python3 .agents/skills/ui-ux-pro-max/scripts/search.py"
//	  → "python3 ~/.coder/cache/ui-ux-pro-max/scripts/search.py"
//
//	"python3 ~/.claude/skills/design/scripts/logo/generate.py"
//	  → "python3 ~/.coder/cache/design/scripts/logo/generate.py"
func RewriteSkillPaths(content string, skillName string) string {
	cacheBase := "~/.coder/cache/" + skillName + "/"

	for _, prefix := range knownPathPrefixes {
		old := prefix + skillName + "/"
		content = strings.ReplaceAll(content, old, cacheBase)
	}

	return content
}
