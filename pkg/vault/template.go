package vault

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"time"
)

// TemplateEngine handles loading and rendering of Obsidian templates
type TemplateEngine struct {
	TemplateDir string
}

// NewTemplateEngine creates a new TemplateEngine
func NewTemplateEngine(templateDir string) *TemplateEngine {
	return &TemplateEngine{
		TemplateDir: templateDir,
	}
}

// LoadTemplate reads a template file from the template directory
func (e *TemplateEngine) LoadTemplate(templateName string) (string, error) {
	// Ensure extension
	if !strings.HasSuffix(templateName, ".md") {
		templateName += ".md"
	}

	path := fmt.Sprintf("%s/%s", e.TemplateDir, templateName)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// Render replaces placeholders in the template content
// Supported placeholders:
// {{title}} - Replaced with the provided title
// {{date:FORMAT}} - Replaced with current date formatted according to FORMAT (e.g. YYYY-MM-DD)
func (e *TemplateEngine) Render(content string, title string) string {
	// Replace {{title}}
	content = strings.ReplaceAll(content, "{{title}}", title)

	// Replace {{date:FORMAT}}
	// Regex to find {{date:FORMAT}}
	re := regexp.MustCompile(`\{\{date:(.*?)\}\}`)

	content = re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract format
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		format := parts[1]

		// Convert Go time format
		// This is a simplified conversion, might need more robust mapping
		// Obsidian uses Moment.js format, Go uses reference time
		goFormat := convertMomentToGoFormat(format)

		return time.Now().Format(goFormat)
	})

	return content
}

// convertMomentToGoFormat converts simple Moment.js format strings to Go time format
func convertMomentToGoFormat(format string) string {
	format = strings.ReplaceAll(format, "YYYY", "2006")
	format = strings.ReplaceAll(format, "MM", "01")
	format = strings.ReplaceAll(format, "DD", "02")
	format = strings.ReplaceAll(format, "HH", "15")
	format = strings.ReplaceAll(format, "mm", "04")
	format = strings.ReplaceAll(format, "ss", "05")
	// Handle [W]WW for week number, Go doesn't support ISO week directly in Format easily without external lib or custom logic
	// For now, let's just handle standard date parts.
	// If we need week number, we might need a custom placeholder logic.
	if strings.Contains(format, "W") {
		// Fallback for week number if needed, or just leave it for now/implement later
		// For this MVP, let's assume standard dates.
		// Actually, let's handle the specific case of YYYY-[W]WW which was in the template
		if format == "2006-[W]WW" {
			y, w := time.Now().ISOWeek()
			return fmt.Sprintf("%d-W%02d", y, w)
		}
	}

	return format
}
