package drive

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/mklimuk/vault-pilot/pkg/db"
	pkgsync "github.com/mklimuk/vault-pilot/pkg/sync"
	"github.com/mklimuk/vault-pilot/pkg/vault"
)

// Watcher monitors a Google Drive folder and creates inbox items for new files.
type Watcher struct {
	service    DriveAPI
	repo       *db.Repository
	vaultPath  string
	tmplEngine *vault.TemplateEngine
	git        *pkgsync.GitManager
	interval   time.Duration
	stopCh     chan struct{}
}

// NewWatcher creates a new Drive watcher.
func NewWatcher(
	service DriveAPI,
	repo *db.Repository,
	vaultPath string,
	tmplEngine *vault.TemplateEngine,
	git *pkgsync.GitManager,
	interval time.Duration,
) *Watcher {
	return &Watcher{
		service:    service,
		repo:       repo,
		vaultPath:  vaultPath,
		tmplEngine: tmplEngine,
		git:        git,
		interval:   interval,
		stopCh:     make(chan struct{}),
	}
}

// Start begins the periodic watch loop.
func (w *Watcher) Start() error {
	// Run once immediately
	if err := w.watchOnce(); err != nil {
		log.Printf("Drive watch initial error: %v", err)
	}

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := w.watchOnce(); err != nil {
					log.Printf("Drive watch error: %v", err)
				}
			case <-w.stopCh:
				return
			}
		}
	}()
	return nil
}

// Stop stops the watch loop.
func (w *Watcher) Stop() {
	close(w.stopCh)
}

func (w *Watcher) watchOnce() error {
	ctx := context.Background()
	files, err := w.service.ListFiles(ctx)
	if err != nil {
		return fmt.Errorf("list files: %w", err)
	}

	modified := false
	for _, f := range files {
		rec, err := w.repo.GetDriveWatchByFileID(f.ID)
		if err != nil {
			log.Printf("Drive watch: db error for %s: %v", f.ID, err)
			continue
		}
		if rec != nil {
			continue // already processed
		}

		// Download file content
		reader, err := w.service.DownloadFile(ctx, f.ID)
		if err != nil {
			log.Printf("Drive watch: download %s: %v", f.Name, err)
			continue
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			log.Printf("Drive watch: read %s: %v", f.Name, err)
			continue
		}

		// Create inbox item
		title := strings.TrimSuffix(f.Name, ".md")
		title = strings.TrimSuffix(title, ".txt")
		content := fmt.Sprintf("Imported from Google Drive: %s\n\n%s", f.Name, string(data))

		if err := vault.CreateInboxItem(w.vaultPath, w.tmplEngine, title, content); err != nil {
			log.Printf("Drive watch: create inbox item for %s: %v", f.Name, err)
			continue
		}

		// Record as processed
		if err := w.repo.InsertDriveWatch(f.ID, f.Name, time.Now()); err != nil {
			log.Printf("Drive watch: insert watch record for %s: %v", f.Name, err)
			continue
		}

		modified = true
	}

	if modified && w.git != nil {
		go func() {
			if err := w.git.Sync("Add Drive watch items"); err != nil {
				log.Printf("Git sync after drive watch: %v", err)
			}
		}()
	}

	return nil
}
