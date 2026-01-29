package util

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// DocumentMeta represents the frontmatter metadata for a SiYuan document
type DocumentMeta struct {
	DocID    string `yaml:"siyuan_doc_id"`
	Notebook string `yaml:"siyuan_notebook"`
	Path     string `yaml:"siyuan_path"`
	HPath    string `yaml:"siyuan_hpath"`
	Updated  string `yaml:"siyuan_updated,omitempty"`
	Title    string `yaml:"title,omitempty"`
}

// MarshalFrontmatter creates YAML frontmatter from metadata
func MarshalFrontmatter(meta *DocumentMeta) (string, error) {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
	}
	return fmt.Sprintf("---\n%s---\n", string(data)), nil
}

// UnmarshalFrontmatter parses YAML frontmatter from content
// Returns the metadata and the remaining content after the frontmatter
func UnmarshalFrontmatter(content string) (*DocumentMeta, string, error) {
	if !strings.HasPrefix(content, "---\n") {
		return nil, content, nil // No frontmatter
	}

	// Find the closing ---
	scanner := bufio.NewScanner(strings.NewReader(content))
	var yamlBuf bytes.Buffer
	lineNum := 0
	foundEnd := false

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if lineNum == 1 {
			continue // Skip the opening ---
		}

		if line == "---" {
			foundEnd = true
			break
		}

		yamlBuf.WriteString(line)
		yamlBuf.WriteString("\n")
	}

	if !foundEnd {
		return nil, content, nil // Malformed frontmatter, return original
	}

	var meta DocumentMeta
	if err := yaml.Unmarshal(yamlBuf.Bytes(), &meta); err != nil {
		return nil, content, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Calculate the position after frontmatter
	// Find the end of the second "---\n"
	remaining := content
	idx := strings.Index(content[4:], "\n---\n")
	if idx != -1 {
		remaining = content[4+idx+5:] // Skip "---\n" + yaml + "\n---\n"
	} else {
		// Try "\n---" at end of file
		idx = strings.Index(content[4:], "\n---")
		if idx != -1 {
			afterIdx := 4 + idx + 4 // Position after "\n---"
			if afterIdx < len(content) && content[afterIdx] == '\n' {
				remaining = content[afterIdx+1:]
			} else {
				remaining = ""
			}
		}
	}

	return &meta, remaining, nil
}

// BuildDocument combines frontmatter and kramdown content
func BuildDocument(meta *DocumentMeta, kramdown string) (string, error) {
	frontmatter, err := MarshalFrontmatter(meta)
	if err != nil {
		return "", err
	}
	return frontmatter + "\n" + kramdown, nil
}

// StripH1Title removes the first H1 heading from content if present
// Returns the title and the remaining content
func StripH1Title(content string) (string, string) {
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) == 0 {
		return "", content
	}

	firstLine := strings.TrimSpace(lines[0])

	// Check if it starts with # (H1)
	if strings.HasPrefix(firstLine, "# ") {
		title := strings.TrimPrefix(firstLine, "# ")
		// Also strip any IAL on the same line or next line
		title = stripIAL(title)

		remaining := ""
		if len(lines) > 1 {
			remaining = lines[1]
			// Also strip any IAL on the next line (for kramdown format)
			remaining = stripLeadingIAL(remaining)
		}

		return title, strings.TrimLeft(remaining, "\n")
	}

	return "", content
}

// stripIAL removes inline attribute list from end of a string
// e.g., "Title {: id=\"xxx\"}" -> "Title"
func stripIAL(s string) string {
	idx := strings.Index(s, "{:")
	if idx != -1 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// stripLeadingIAL removes IAL from the start of content
func stripLeadingIAL(content string) string {
	content = strings.TrimLeft(content, " \t")
	if strings.HasPrefix(content, "{:") {
		// Find the end of the IAL
		idx := strings.Index(content, "}")
		if idx != -1 {
			content = strings.TrimLeft(content[idx+1:], " \t\n")
		}
	}
	return content
}
