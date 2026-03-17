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
			"full-lifecycle-delivery.md",
			"new-requirement.md",
			"execute-plan.md",
			"qa-testing.md",
			"code-review.md",
			"debug.md",
			"debug-leak.md",
			"writing-test.md",
			"check-implementation.md",
			"remember.md",
			"capture-knowledge.md",
			"technical-writer-review.md",
			"update-planning.md",
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
			"new-requirement.md",
			"execute-plan.md",
			"qa-testing.md",
			"code-review.md",
			"debug.md",
			"writing-test.md",
			"review-design.md",
			"check-implementation.md",
			"remember.md",
			"capture-knowledge.md",
			"simplify-implementation.md",
			"technical-writer-review.md",
		},
		AgentFile:       "coder-fe.agent.md",
		ClaudeAgentFile: "coder-fe.md",
	},
	"fullstack": {
		Name:            "fullstack",
		Description:     "Full-stack development (backend + frontend)",
		Rules:           nil, // all
		Workflows:       nil, // all
		AgentFile:       "coder.agent.md",
		ClaudeAgentFile: "coder.md",
	},
	"all": {
		Name:            "all",
		Description:     "All available files, rules, and workflows",
		Rules:           nil, // all
		Workflows:       nil, // all
		AgentFile:       "", // all VS Code agent files as-is
		ClaudeAgentFile: "", // all Claude agent files as-is
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
	fmt.Println("  fullstack  Full-stack development (backend + frontend)")
	fmt.Println("  all        All available files, rules, and workflows")
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
