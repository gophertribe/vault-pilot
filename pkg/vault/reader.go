package vault

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ReadNote reads a markdown file and parses its frontmatter and content
func ReadNote(path string) (*Note, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var frontmatterLines []string
	var contentLines []string
	inFrontmatter := false
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if lineCount == 1 && line == "---" {
			inFrontmatter = true
			continue
		}

		if inFrontmatter {
			if line == "---" {
				inFrontmatter = false
				continue
			}
			frontmatterLines = append(frontmatterLines, line)
		} else {
			contentLines = append(contentLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Parse Frontmatter
	fmData := strings.Join(frontmatterLines, "\n")

	// We might want to parse into specific structs based on "type" field,
	// but for generic reading, map[string]interface{} or a generic struct is safer first.
	// Let's try to parse into a map first to check the type.
	var rawFM map[string]interface{}
	if len(fmData) > 0 {
		if err := yaml.Unmarshal([]byte(fmData), &rawFM); err != nil {
			return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
		}
	}

	// Now try to map to specific structs if needed, or just return the map/generic struct
	// For now, let's return the raw map in the Note struct, and helper functions can cast it.

	return &Note{
		Path:        path,
		Frontmatter: rawFM,
		Content:     strings.Join(contentLines, "\n"),
	}, nil
}

// ParseInboxItem parses the frontmatter into an InboxItem struct
func ParseInboxItem(n *Note) (*InboxItem, error) {
	data, err := yaml.Marshal(n.Frontmatter)
	if err != nil {
		return nil, err
	}
	var item InboxItem
	if err := yaml.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// ParseProject parses the frontmatter into a Project struct
func ParseProject(n *Note) (*Project, error) {
	data, err := yaml.Marshal(n.Frontmatter)
	if err != nil {
		return nil, err
	}
	var item Project
	if err := yaml.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	return &item, nil
}
