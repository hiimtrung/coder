package profiles

import "fmt"

// Profile defines a named configuration for installation.
// Rules/Workflows nil means "install all". AgentFile/ClaudeAgentFile empty means "install all files as-is".
type Profile struct {
	Name            string
	Description     string
	Rules           []string // rule file basenames from .agents/rules/ (nil = all)
	Workflows       []string // workflow file basenames from .agents/workflows/ (nil = all)
	AgentFile       string   // VS Code Copilot agent: .github/agents/<file> → .github/agents/coder.agent.md (empty = all as-is)
	ClaudeAgentFile string   // Claude CLI agent: .claude/agents/<file> → .claude/agents/<file> (empty = all as-is)
}

var predefined = map[string]Profile{
	"be": {
		Name:        "be",
		Description: "Backend development (NestJS, Java, Go, Python, Rust, C, Dart)",
		Rules: []string{
			"general.instructions.md",
			"be.instructions.md",
		},
		Workflows: []string{
			"clarify-requirements.md",
			"architecture-design.md",
			"implement-feature.md",
			"code-review.md",
			"qa-test.md",
			"debug-issue.md",
			"debug-leak.md",
			"writing-test.md",
			"check-implementation.md",
			"release-readiness.md",
			"knowledge-capture.md",
			"technical-writer-review.md",
			"review-requirements.md",
			"simplify-implementation.md",
		},
		AgentFile:       "coder-be.agent.md",
		ClaudeAgentFile: "coder-be.md",
	},
	"fe": {
		Name:        "fe",
		Description: "Frontend development (React, Next.js, React Native)",
		Rules: []string{
			"general.instructions.md",
			"fe.instructions.md",
		},
		Workflows: []string{
			"clarify-requirements.md",
			"implement-feature.md",
			"code-review.md",
			"qa-test.md",
			"debug-issue.md",
			"writing-test.md",
			"review-design.md",
			"check-implementation.md",
			"knowledge-capture.md",
			"simplify-implementation.md",
			"technical-writer-review.md",
			"write-documentation.md",
		},
		AgentFile:       "coder-fe.agent.md",
		ClaudeAgentFile: "coder-fe.md",
	},
	"fullstack": {
		Name:            "fullstack",
		Description:     "Full-stack delivery with complete professional team simulation",
		Rules:           nil, // all
		Workflows:       nil, // all
		AgentFile:       "coder.agent.md",
		ClaudeAgentFile: "", // all Claude agent files (coder + all specialists)
	},
	"all": {
		Name:            "all",
		Description:     "All available files, rules, workflows, and agent definitions",
		Rules:           nil,
		Workflows:       nil,
		AgentFile:       "",
		ClaudeAgentFile: "",
	},
}

// Get returns the Profile for the given name. Supports predefined profiles only.
func Get(name string) (Profile, error) {
	if p, ok := predefined[name]; ok {
		return p, nil
	}
	return Profile{}, fmt.Errorf(
		"unknown profile: %q\n\nAvailable profiles: be, fe, fullstack, all\nRun 'coder list' to see all profiles",
		name,
	)
}

// PrintAll prints all predefined profiles.
func PrintAll() {
	fmt.Println("Available profiles:")
	fmt.Println("  be         Backend development (NestJS, Java, Go, Python, Rust, C, Dart)")
	fmt.Println("  fe         Frontend development (React, Next.js, React Native)")
	fmt.Println("  fullstack  Full-stack delivery with complete professional team simulation")
	fmt.Println("  all        All available files, rules, workflows, and agent definitions")
}

// PrintProfile prints the details of a single profile.
func PrintProfile(p Profile) {
	fmt.Printf("Profile: %s\n", p.Name)
	fmt.Printf("Description: %s\n", p.Description)

	fmt.Println("Workflows:")
	if p.Workflows == nil {
		fmt.Println("  (all)")
	} else {
		for _, w := range p.Workflows {
			fmt.Printf("  - %s\n", w)
		}
	}

	fmt.Println("Rules:")
	if p.Rules == nil {
		fmt.Println("  (all)")
	} else {
		for _, r := range p.Rules {
			fmt.Printf("  - %s\n", r)
		}
	}

	if p.AgentFile == "" {
		fmt.Println("VS Code Agent: (all files as-is)")
	} else {
		fmt.Printf("VS Code Agent: %s → coder.agent.md\n", p.AgentFile)
	}

	if p.ClaudeAgentFile == "" {
		fmt.Println("Claude Agent: (all files as-is)")
	} else {
		fmt.Printf("Claude Agent: %s\n", p.ClaudeAgentFile)
	}
}
