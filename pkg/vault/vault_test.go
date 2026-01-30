package vault

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTemplateEngine(t *testing.T) {
	// Setup temporary template directory
	tmpDir, err := ioutil.TempDir("", "vault-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy template
	tmplContent := "---\ncreated: {{date:YYYY-MM-DD}}\n---\n# {{title}}"
	err = ioutil.WriteFile(filepath.Join(tmpDir, "Test Template.md"), []byte(tmplContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	engine := NewTemplateEngine(tmpDir)

	// Test Load
	content, err := engine.LoadTemplate("Test Template")
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	if content != tmplContent {
		t.Errorf("Expected content %q, got %q", tmplContent, content)
	}

	// Test Render
	rendered := engine.Render(content, "My Note")
	expectedDate := time.Now().Format("2006-01-02")

	if !strings.Contains(rendered, expectedDate) {
		t.Errorf("Rendered content missing date: %s", rendered)
	}
	if !strings.Contains(rendered, "# My Note") {
		t.Errorf("Rendered content missing title: %s", rendered)
	}
}

func TestReadWriteNote(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "vault-test-rw")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	notePath := filepath.Join(tmpDir, "test_note.md")

	// Create a note struct
	note := &Note{
		Path: notePath,
		Frontmatter: map[string]interface{}{
			"title": "Test Note",
			"tags":  []string{"test", "go"},
		},
		Content: "\n# Hello World\nThis is a test.",
	}

	// Write it
	err = WriteNote(note)
	if err != nil {
		t.Fatalf("Failed to write note: %v", err)
	}

	// Read it back
	readNote, err := ReadNote(notePath)
	if err != nil {
		t.Fatalf("Failed to read note: %v", err)
	}

	// Verify
	fm, ok := readNote.Frontmatter.(map[string]interface{})
	if !ok {
		t.Fatal("Frontmatter is not a map")
	}

	if fm["title"] != "Test Note" {
		t.Errorf("Expected title 'Test Note', got %v", fm["title"])
	}

	// Check content (trim whitespace to be safe)
	if !strings.Contains(readNote.Content, "# Hello World") {
		t.Errorf("Content mismatch. Got: %s", readNote.Content)
	}
}
