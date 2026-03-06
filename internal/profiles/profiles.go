package profiles

import "fmt"

// Profile defines a named set of skills, rules, workflows, and agents to install.
type Profile struct {
	Name             string
	Description      string
	Skills           []string // skill directory names; "all" means every available skill
	IncludeRules     bool
	IncludeWorkflows bool
	IncludeAgents    bool
}

// AllSkills is the ordered list of all skill directory names embedded in the binary.
var AllSkills = []string{
	"architecture",
	"c",
	"composition-patterns",
	"dart",
	"database",
	"development",
	"docs-analysis",
	"frontend",
	"general-patterns",
	"golang",
	"java",
	"nestjs",
	"python",
	"react-best-practices",
	"react-native-skills",
	"rust",
	"testing",
	"ui-ux-pro-max",
	"web-design-guidelines",
}

// coreSkills are included in every predefined profile.
var coreSkills = []string{
	"architecture",
	"general-patterns",
	"development",
	"database",
	"testing",
	"docs-analysis",
}

var predefined = map[string]Profile{
	"be": {
		Name:        "be",
		Description: "Backend development (NestJS, Java, Go, Python, Rust, C, Dart)",
		Skills: concat(coreSkills,
			"nestjs", "java", "golang", "python", "rust", "c", "dart",
		),
		IncludeRules:     true,
		IncludeWorkflows: true,
		IncludeAgents:    true,
	},
	"fe": {
		Name:        "fe",
		Description: "Frontend development (React, Next.js, Vue, Svelte, React Native)",
		Skills: concat(coreSkills,
			"frontend", "react-best-practices", "react-native-skills",
			"composition-patterns", "web-design-guidelines", "ui-ux-pro-max",
		),
		IncludeRules:     true,
		IncludeWorkflows: true,
		IncludeAgents:    true,
	},
	"fullstack": {
		Name:        "fullstack",
		Description: "Full-stack development (backend + frontend)",
		Skills: concat(coreSkills,
			"nestjs", "java", "golang", "python", "rust", "c", "dart",
			"frontend", "react-best-practices", "react-native-skills",
			"composition-patterns", "web-design-guidelines", "ui-ux-pro-max",
		),
		IncludeRules:     true,
		IncludeWorkflows: true,
		IncludeAgents:    true,
	},
	"all": {
		Name:             "all",
		Description:      "All available skills, rules, and workflows",
		Skills:           []string{"all"},
		IncludeRules:     true,
		IncludeWorkflows: true,
		IncludeAgents:    true,
	},
}

// Get returns the Profile for the given name. Supports predefined profiles and
// individual skill names (installs core skills + the named skill).
func Get(name string) (Profile, error) {
	if p, ok := predefined[name]; ok {
		return p, nil
	}
	for _, s := range AllSkills {
		if s == name {
			return Profile{
				Name:             name,
				Description:      fmt.Sprintf("Individual skill: %s", name),
				Skills:           []string{name},
				IncludeRules:     true,
				IncludeWorkflows: true,
				IncludeAgents:    true,
			}, nil
		}
	}
	return Profile{}, fmt.Errorf(
		"unknown profile or skill: %q\n\nAvailable profiles: be, fe, fullstack, all\nRun 'coder list' to see all available skills",
		name,
	)
}

// PrintAll prints all predefined profiles and available skills.
func PrintAll() {
	fmt.Println("Predefined profiles:")
	fmt.Println("  be         Backend development (NestJS, Java, Go, Python, Rust, C, Dart)")
	fmt.Println("  fe         Frontend development (React, Next.js, Vue, Svelte, React Native)")
	fmt.Println("  fullstack  Full-stack development (backend + frontend)")
	fmt.Println("  all        All available skills, rules, and workflows")
	fmt.Println()
	fmt.Println("Available skills (install individually):")
	for _, s := range AllSkills {
		fmt.Printf("  %s\n", s)
	}
}

// PrintProfile prints the details of a single profile.
func PrintProfile(p Profile) {
	fmt.Printf("Profile: %s\n", p.Name)
	fmt.Printf("Description: %s\n", p.Description)
	fmt.Println("Skills:")
	skills := p.Skills
	if len(skills) == 1 && skills[0] == "all" {
		skills = AllSkills
	}
	for _, s := range skills {
		fmt.Printf("  - %s\n", s)
	}
	fmt.Printf("Includes rules: %v\n", p.IncludeRules)
	fmt.Printf("Includes workflows: %v\n", p.IncludeWorkflows)
	fmt.Printf("Includes agents: %v\n", p.IncludeAgents)
}

func concat(base []string, extra ...string) []string {
	result := make([]string, len(base)+len(extra))
	copy(result, base)
	copy(result[len(base):], extra)
	return result
}
