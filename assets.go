package tasagent

import "embed"

// AgentFS contains all embedded skills, rules, workflows, and agent definitions.
// Files starting with '.' (e.g. .DS_Store) are excluded unless the all: prefix is used.
// The all: prefix is required to include files starting with '_' (e.g. _sections.md, _template.md).
//
//go:embed all:.agents/skills all:.agents/rules all:.agents/workflows all:.github/agents
var AgentFS embed.FS
