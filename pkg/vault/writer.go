package vault

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// WriteNote writes a note to the specified path
func WriteNote(note *Note) error {
	// Marshal Frontmatter
	fmData, err := yaml.Marshal(note.Frontmatter)
	if err != nil {
		return fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Construct file content
	content := fmt.Sprintf("---\n%s---\n%s", string(fmData), note.Content)

	// Ensure directory exists
	dir := filepath.Dir(note.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write file
	if err := ioutil.WriteFile(note.Path, []byte(content), 0644); err != nil {
		return err
	}

	return nil
}

// CreateInboxItem creates a new inbox item from a template
func CreateInboxItem(vaultPath string, templateEngine *TemplateEngine, title string, content string) error {
	// Load Template
	tmpl, err := templateEngine.LoadTemplate("Inbox Item Template")
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Render Template
	rendered := templateEngine.Render(tmpl, title)

	// Parse rendered content to separate frontmatter (if we need to modify it programmatically)
	// For now, let's just append the user content if the template has a specific place,
	// or just assume the template is the starting point.
	// The template engine returns the full file content (FM + Body).

	// If we want to inject specific content into the "Description" or "Notes" section,
	// we might need a more sophisticated parser or just simple string replacement.
	// For MVP, let's assume the "Description" in the template is a placeholder or we just append.

	// Let's replace "Brief description of the item" with actual content if provided
	if content != "" {
		rendered = strings.Replace(rendered, "Brief description of the item", content, 1)
	}

	// Generate Filename (sanitize title)
	filename := SanitizeFilename(title) + ".md"
	path := filepath.Join(vaultPath, "1. Inbox", filename)

	// Write to file
	// We can just write the rendered string directly since it already has FM.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(path, []byte(rendered), 0644)
}

// SanitizeFilename removes characters invalid in filenames.
func SanitizeFilename(name string) string {
	// Simple sanitization
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalid {
		name = strings.ReplaceAll(name, char, "-")
	}
	return name
}
