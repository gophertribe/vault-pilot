package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mklimuk/vault-pilot/pkg/ai"
	"github.com/mklimuk/vault-pilot/pkg/db"
	"github.com/mklimuk/vault-pilot/pkg/sync"
	"github.com/mklimuk/vault-pilot/pkg/vault"
)

// Handler holds dependencies for API handlers
type Handler struct {
	Repo       *db.Repository
	AI         ai.Generator
	TmplEngine *vault.TemplateEngine
	VaultPath  string
	Git        *sync.GitManager
}

// CreateInboxRequest represents the payload for creating an inbox item
type CreateInboxRequest struct {
	Content string `json:"content"`
}

// HandleCreateInboxItem handles POST /inbox
func (h *Handler) HandleCreateInboxItem(w http.ResponseWriter, r *http.Request) {
	var req CreateInboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Analyze content with AI
	prompt := ai.AnalyzeInboxPrompt(req.Content)
	analysisJSON, err := h.AI.GenerateText(r.Context(), prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse AI response (expecting JSON)
	// For simplicity, we'll just try to unmarshal it into a map or struct
	// Note: LLMs might return markdown code blocks, need to strip them
	analysisJSON = cleanJSON(analysisJSON)

	var analysis struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		// We could use other fields too
	}
	if err := json.Unmarshal([]byte(analysisJSON), &analysis); err != nil {
		// Fallback if JSON is invalid
		analysis.Title = "New Inbox Item"
		analysis.Description = req.Content
	}

	// 2. Create file using Vault Controller
	err = vault.CreateInboxItem(h.VaultPath, h.TmplEngine, analysis.Title, analysis.Description)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create file: %v", err), http.StatusInternalServerError)
		return
	}

	// 3. Sync with Git
	if h.Git != nil {
		go func() {
			if err := h.Git.Sync("Add inbox item: " + analysis.Title); err != nil {
				fmt.Printf("Git sync failed: %v\n", err)
			}
		}()
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created", "title": analysis.Title})
}

// HandleListProjects handles GET /projects
func (h *Handler) HandleListProjects(w http.ResponseWriter, r *http.Request) {
	// Scan "3. Projects" directory
	projectsDir := filepath.Join(h.VaultPath, "3. Projects")
	var activeProjects []string

	err := filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			// Read note to check status
			note, err := vault.ReadNote(path)
			if err != nil {
				return nil // Skip unreadable
			}

			// Check status in frontmatter
			if fm, ok := note.Frontmatter.(map[string]interface{}); ok {
				if status, ok := fm["status"].(string); ok && status == "active" {
					activeProjects = append(activeProjects, strings.TrimSuffix(info.Name(), ".md"))
				}
			}
		}
		return nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan projects: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"projects": activeProjects})
}

// HandleGenerateWeeklyReview handles POST /review/weekly
func (h *Handler) HandleGenerateWeeklyReview(w http.ResponseWriter, r *http.Request) {
	// 1. Gather Context
	// Count Inbox
	inboxDir := filepath.Join(h.VaultPath, "1. Inbox")
	inboxCount := 0
	filepath.Walk(inboxDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			inboxCount++
		}
		return nil
	})

	// Get Active Projects (reuse logic or refactor)
	// For brevity, let's assume we have a helper or just do it again
	// Actually, let's just call the internal logic if we extracted it, but for now copy-paste is safer for speed
	projectsDir := filepath.Join(h.VaultPath, "3. Projects")
	var activeProjects []string
	filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			note, _ := vault.ReadNote(path)
			if note != nil {
				if fm, ok := note.Frontmatter.(map[string]interface{}); ok {
					if status, ok := fm["status"].(string); ok && status == "active" {
						activeProjects = append(activeProjects, strings.TrimSuffix(info.Name(), ".md"))
					}
				}
			}
		}
		return nil
	})

	// 2. Generate Content with AI
	prompt := ai.GenerateReviewPrompt(activeProjects, inboxCount)
	aiResponse, err := h.AI.GenerateText(r.Context(), prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 3. Create Review File
	// Load Weekly Review Template
	tmpl, err := h.TmplEngine.LoadTemplate("Weekly Review Template")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load template: %v", err), http.StatusInternalServerError)
		return
	}

	// Render basic placeholders
	title := fmt.Sprintf("Weekly Review - %s", time.Now().Format("2006-01-02"))
	content := h.TmplEngine.Render(tmpl, title)

	// Inject AI content
	// This is tricky without a robust parser.
	// We'll append the AI response to the "Reflections" section if possible, or just at the end.
	// Or better, we replace a placeholder if we added one, but we didn't.
	// Let's just append it for now.
	content += "\n\n## AI Insights\n" + aiResponse

	// Populate Active Projects List in the markdown
	// Find "### Active Projects" and inject list
	projectListMD := ""
	for _, p := range activeProjects {
		projectListMD += fmt.Sprintf("- [ ] [[%s]] - Status: Active\n", p)
	}

	// Simple string replacement for the section
	if strings.Contains(content, "### Active Projects") {
		// We want to insert after the header.
		// Regex or split? Split is easier.
		parts := strings.Split(content, "### Active Projects")
		if len(parts) > 1 {
			// Reconstruct: Part 0 + Header + List + Part 1
			// But Part 1 starts with the content after header.
			// We need to be careful not to overwrite existing text if template has some.
			// The template has:
			// ### Active Projects
			// Review each project for:
			// ...

			// So we should probably append to the list or replace the placeholder lines.
			// Let's just inject it after "Review each project for:"
			if strings.Contains(parts[1], "Review each project for:") {
				subParts := strings.Split(parts[1], "Review each project for:")
				content = parts[0] + "### Active Projects" + subParts[0] + "Review each project for:\n" + projectListMD + subParts[1]
			}
		}
	}

	// Write File
	filename := fmt.Sprintf("%s Weekly Review.md", time.Now().Format("2006-W15")) // Using W15 as example format
	// Actually use correct ISO week
	y, weekNum := time.Now().ISOWeek()
	filename = fmt.Sprintf("%d-W%02d Weekly Review.md", y, weekNum)

	path := filepath.Join(h.VaultPath, "6. Weekly Reviews", filename)

	// We need to construct a Note object to write
	// But WriteNote expects parsed frontmatter.
	// Our TemplateEngine returns a string with FM.
	// We should probably just write the string directly like in CreateInboxItem.
	// Refactor Writer to have a WriteFileContent method?
	// Or just use ioutil.WriteFile here.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create dir: %v", err), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write file: %v", err), http.StatusInternalServerError)
		return
	}

	// Log to DB
	weekStr := fmt.Sprintf("%d-W%02d", y, weekNum)
	h.Repo.LogReview(weekStr)

	// Sync with Git
	if h.Git != nil {
		go func() {
			if err := h.Git.Sync("Add Weekly Review " + weekStr); err != nil {
				fmt.Printf("Git sync failed: %v\n", err)
			}
		}()
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created", "path": filename})
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return s
}
