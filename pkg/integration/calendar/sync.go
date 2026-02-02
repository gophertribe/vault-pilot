package calendar

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mklimuk/vault-pilot/pkg/db"
	"github.com/mklimuk/vault-pilot/pkg/sync"
	"github.com/mklimuk/vault-pilot/pkg/vault"
)

// Syncer performs bidirectional sync between Google Calendar and the vault.
type Syncer struct {
	service    CalendarAPI
	repo       *db.Repository
	vaultPath  string
	tmplEngine *vault.TemplateEngine
	git        *sync.GitManager
	interval   time.Duration
	horizon    time.Duration
	stopCh     chan struct{}
}

// NewSyncer creates a new calendar syncer.
func NewSyncer(
	service CalendarAPI,
	repo *db.Repository,
	vaultPath string,
	tmplEngine *vault.TemplateEngine,
	git *sync.GitManager,
	interval, horizon time.Duration,
) *Syncer {
	return &Syncer{
		service:    service,
		repo:       repo,
		vaultPath:  vaultPath,
		tmplEngine: tmplEngine,
		git:        git,
		interval:   interval,
		horizon:    horizon,
		stopCh:     make(chan struct{}),
	}
}

// Start begins the periodic sync loop.
func (s *Syncer) Start() error {
	// Run once immediately
	if err := s.syncOnce(); err != nil {
		log.Printf("Calendar initial sync error: %v", err)
	}

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.syncOnce(); err != nil {
					log.Printf("Calendar sync error: %v", err)
				}
			case <-s.stopCh:
				return
			}
		}
	}()
	return nil
}

// Stop stops the sync loop.
func (s *Syncer) Stop() {
	close(s.stopCh)
}

func (s *Syncer) syncOnce() error {
	ctx := context.Background()

	modified := false

	pullMod, err := s.pull(ctx)
	if err != nil {
		log.Printf("Calendar pull error: %v", err)
	}
	modified = modified || pullMod

	pushMod, err := s.push(ctx)
	if err != nil {
		log.Printf("Calendar push error: %v", err)
	}
	modified = modified || pushMod

	if modified && s.git != nil {
		go func() {
			if err := s.git.Sync("Calendar sync"); err != nil {
				log.Printf("Git sync after calendar: %v", err)
			}
		}()
	}

	return nil
}

// pull fetches events from Calendar and creates/updates vault notes.
func (s *Syncer) pull(ctx context.Context) (bool, error) {
	events, err := s.service.FetchUpcoming(ctx, s.horizon)
	if err != nil {
		return false, fmt.Errorf("fetch upcoming: %w", err)
	}

	modified := false
	for _, evt := range events {
		syncKey := buildSyncKey(evt)
		rec, err := s.repo.GetCalendarSyncByEventID(evt.ID)
		if err != nil {
			log.Printf("Calendar pull: db error for %s: %v", evt.ID, err)
			continue
		}

		if rec == nil {
			// New event — create vault note
			vaultFilePath, err := s.createCalendarNote(evt)
			if err != nil {
				log.Printf("Calendar pull: create note for %s: %v", evt.ID, err)
				continue
			}
			if err := s.repo.InsertCalendarSync(evt.ID, vaultFilePath, syncKey, "pull"); err != nil {
				log.Printf("Calendar pull: insert sync for %s: %v", evt.ID, err)
				continue
			}
			modified = true
		} else if rec.SyncKey != syncKey {
			// Changed event — update vault note
			if err := s.updateCalendarNote(rec.VaultPath, evt); err != nil {
				log.Printf("Calendar pull: update note for %s: %v", evt.ID, err)
				continue
			}
			if err := s.repo.UpdateCalendarSync(evt.ID, syncKey); err != nil {
				log.Printf("Calendar pull: update sync for %s: %v", evt.ID, err)
				continue
			}
			modified = true
		}
	}

	return modified, nil
}

// push scans vault for notes with due_date and pushes new/changed events to Calendar.
func (s *Syncer) push(ctx context.Context) (bool, error) {
	modified := false

	dirs := []string{
		filepath.Join(s.vaultPath, "2. Next Actions"),
		filepath.Join(s.vaultPath, "3. Projects"),
	}

	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip inaccessible
			}
			if info.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}

			note, err := vault.ReadNote(path)
			if err != nil {
				return nil
			}

			fm, ok := note.Frontmatter.(map[string]interface{})
			if !ok {
				return nil
			}

			// Skip notes that came from calendar pull (have calendar_id)
			if _, hasCalID := fm["calendar_id"]; hasCalID {
				return nil
			}

			dueDateStr, ok := fm["due_date"].(string)
			if !ok || dueDateStr == "" {
				return nil
			}

			relPath, _ := filepath.Rel(s.vaultPath, path)
			rec, err := s.repo.GetCalendarSyncByVaultPath(relPath)
			if err != nil {
				return nil
			}

			dueDate, err := time.Parse("2006-01-02", dueDateStr)
			if err != nil {
				return nil
			}

			title := ""
			if t, ok := fm["title"].(string); ok {
				title = t
			} else {
				// Use filename without extension
				title = strings.TrimSuffix(filepath.Base(path), ".md")
			}

			evt := Event{
				Summary:   title,
				StartTime: dueDate,
				EndTime:   dueDate.Add(time.Hour),
			}

			if rec == nil {
				// New — create event
				eventID, err := s.service.CreateEvent(ctx, evt)
				if err != nil {
					log.Printf("Calendar push: create event for %s: %v", relPath, err)
					return nil
				}
				if err := s.repo.InsertCalendarSync(eventID, relPath, dueDateStr, "push"); err != nil {
					log.Printf("Calendar push: insert sync for %s: %v", relPath, err)
					return nil
				}
				modified = true
			} else if rec.SyncKey != dueDateStr {
				// Changed due_date — update event
				if err := s.service.UpdateEvent(ctx, rec.EventID, evt); err != nil {
					log.Printf("Calendar push: update event for %s: %v", relPath, err)
					return nil
				}
				if err := s.repo.UpdateCalendarSync(rec.EventID, dueDateStr); err != nil {
					log.Printf("Calendar push: update sync for %s: %v", relPath, err)
					return nil
				}
				modified = true
			}

			return nil
		})
		if err != nil {
			log.Printf("Calendar push: walk %s: %v", dir, err)
		}
	}

	return modified, nil
}

func (s *Syncer) createCalendarNote(evt Event) (string, error) {
	calDir := filepath.Join(s.vaultPath, "2. Next Actions", "@calendar")
	if err := os.MkdirAll(calDir, 0755); err != nil {
		return "", fmt.Errorf("create calendar dir: %w", err)
	}

	filename := vault.SanitizeFilename(evt.Summary) + ".md"
	fullPath := filepath.Join(calDir, filename)
	relPath := filepath.Join("2. Next Actions", "@calendar", filename)

	fm := map[string]interface{}{
		"created":     time.Now().Format("2006-01-02"),
		"status":      "scheduled",
		"context":     "@calendar",
		"calendar_id": evt.ID,
		"due_date":    evt.StartTime.Format("2006-01-02"),
	}

	body := fmt.Sprintf("\n# %s\n", evt.Summary)
	if evt.Description != "" {
		body += "\n" + evt.Description + "\n"
	}
	if evt.Location != "" {
		body += fmt.Sprintf("\n**Location:** %s\n", evt.Location)
	}
	body += fmt.Sprintf("\n**Time:** %s - %s\n",
		evt.StartTime.Format("15:04"),
		evt.EndTime.Format("15:04"))

	note := &vault.Note{
		Path:        fullPath,
		Frontmatter: fm,
		Content:     body,
	}

	if err := vault.WriteNote(note); err != nil {
		return "", fmt.Errorf("write note: %w", err)
	}

	return relPath, nil
}

func (s *Syncer) updateCalendarNote(relVaultPath string, evt Event) error {
	fullPath := filepath.Join(s.vaultPath, relVaultPath)

	note, err := vault.ReadNote(fullPath)
	if err != nil {
		return fmt.Errorf("read note: %w", err)
	}

	fm, ok := note.Frontmatter.(map[string]interface{})
	if !ok {
		fm = make(map[string]interface{})
	}

	fm["due_date"] = evt.StartTime.Format("2006-01-02")
	note.Frontmatter = fm

	body := fmt.Sprintf("\n# %s\n", evt.Summary)
	if evt.Description != "" {
		body += "\n" + evt.Description + "\n"
	}
	if evt.Location != "" {
		body += fmt.Sprintf("\n**Location:** %s\n", evt.Location)
	}
	body += fmt.Sprintf("\n**Time:** %s - %s\n",
		evt.StartTime.Format("15:04"),
		evt.EndTime.Format("15:04"))

	note.Content = body

	return vault.WriteNote(note)
}

func buildSyncKey(evt Event) string {
	return fmt.Sprintf("%s|%s|%s",
		evt.Summary,
		evt.StartTime.Format(time.RFC3339),
		evt.EndTime.Format(time.RFC3339))
}
