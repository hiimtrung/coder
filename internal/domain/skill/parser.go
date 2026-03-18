package skill

import (
	"strings"
)

// ParsedSkill holds the parsed result from a SKILL.md file.
type ParsedSkill struct {
	Name        string
	Description string
	Category    string
	Tags        []string
	Sections    []ParsedSection
}

// ParsedSection represents a chunk extracted from SKILL.md body.
type ParsedSection struct {
	Title     string
	Content   string
	Type      string // "description", "rule", "example", "workflow"
	SectionID string // set by ingestor to link all parts split from this section
}

// ParseSkillMD parses a SKILL.md file content and extracts metadata + sections.
func ParseSkillMD(name string, content string) ParsedSkill {
	ps := ParsedSkill{Name: name}

	// Strip YAML frontmatter if present
	body := content
	if strings.HasPrefix(content, "---") {
		rest := content[3:]
		if frontmatter, after, found := strings.Cut(rest, "\n---"); found {
			body = strings.TrimSpace(after)

			// Parse simple frontmatter fields
			for _, line := range strings.Split(frontmatter, "\n") {
				line = strings.TrimSpace(line)
				if v, ok := strings.CutPrefix(line, "name:"); ok {
					ps.Name = strings.Trim(strings.TrimSpace(v), "\"'")
				}
				if v, ok := strings.CutPrefix(line, "description:"); ok {
					ps.Description = strings.Trim(strings.TrimSpace(v), "\"'")
				}
				if v, ok := strings.CutPrefix(line, "category:"); ok {
					ps.Category = strings.Trim(strings.TrimSpace(v), "\"'")
				}
			}
		}
	}

	// Split body by ## headers into sections
	sections := splitBySections(body)

	// First section (before any ##) is always description
	if len(sections) > 0 && sections[0].Title == "" {
		if ps.Description == "" {
			// Use first ~200 chars as description
			desc := sections[0].Content
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			ps.Description = strings.TrimSpace(desc)
		}
		sections[0].Type = "description"
		if sections[0].Title == "" {
			sections[0].Title = "Overview"
		}
	}

	ps.Sections = sections
	return ps
}

// splitBySections splits markdown content by ## headers.
func splitBySections(body string) []ParsedSection {
	lines := strings.Split(body, "\n")
	var sections []ParsedSection
	var current *ParsedSection

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// Save current section
			if current != nil {
				current.Content = strings.TrimSpace(current.Content)
				sections = append(sections, *current)
			}
			title := strings.TrimPrefix(line, "## ")
			title = strings.TrimSpace(title)
			stype := classifySection(title)
			current = &ParsedSection{
				Title: title,
				Type:  stype,
			}
		} else if strings.HasPrefix(line, "# ") && current == nil {
			// Top-level header, start description section
			current = &ParsedSection{
				Title: strings.TrimSpace(strings.TrimPrefix(line, "# ")),
				Type:  "description",
			}
		} else {
			if current == nil {
				current = &ParsedSection{Type: "description"}
			}
			current.Content += line + "\n"
		}
	}

	// Don't forget the last section
	if current != nil {
		current.Content = strings.TrimSpace(current.Content)
		sections = append(sections, *current)
	}

	return sections
}

// classifySection tries to classify a section based on its title.
func classifySection(title string) string {
	lower := strings.ToLower(title)

	if strings.Contains(lower, "rule") || strings.Contains(lower, "principle") || strings.Contains(lower, "guideline") || strings.Contains(lower, "constraint") {
		return "rule"
	}
	if strings.Contains(lower, "example") || strings.Contains(lower, "usage") || strings.Contains(lower, "demo") {
		return "example"
	}
	if strings.Contains(lower, "workflow") || strings.Contains(lower, "process") || strings.Contains(lower, "step") {
		return "workflow"
	}
	return "rule" // Default to rule for skill sections
}

// ParseRuleFile parses a single rule markdown file into one or more sections.
// If the file contains ## headers, it is split by those headers so large files
// are naturally chunked by semantic boundary before size-based splitting.
// If no headers are found, the entire file is returned as one section.
func ParseRuleFile(path, content string) []ParsedSection {
	// Extract filename without extension as fallback title
	parts := strings.Split(path, "/")
	filename := parts[len(parts)-1]
	fileTitle := strings.TrimSuffix(filename, ".md")
	fileTitle = strings.ReplaceAll(fileTitle, "-", " ")
	fileTitle = strings.ReplaceAll(fileTitle, "_", " ")

	// Strip frontmatter
	body := content
	if strings.HasPrefix(content, "---") {
		rest := content[3:]
		if _, after, found := strings.Cut(rest, "\n---"); found {
			body = strings.TrimSpace(after)
		}
	}

	// Split by ## headers for natural semantic chunking
	sections := splitBySections(body)

	// If no ## sections were found, return the whole body as one section
	if len(sections) == 0 {
		return []ParsedSection{{Title: fileTitle, Content: body, Type: "rule"}}
	}

	// Prefix unnamed sections with the filename title for context
	for i := range sections {
		if sections[i].Title == "" || sections[i].Title == "Overview" {
			sections[i].Title = fileTitle
		} else {
			sections[i].Title = fileTitle + " — " + sections[i].Title
		}
		sections[i].Type = "rule"
	}

	return sections
}
