package drive

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mklimuk/vault-pilot/pkg/db"
)

// Backup performs incremental vault backup to Google Drive.
type Backup struct {
	service   DriveAPI
	repo      *db.Repository
	vaultPath string
	interval  time.Duration
	stopCh    chan struct{}
}

// NewBackup creates a new Drive backup service.
func NewBackup(service DriveAPI, repo *db.Repository, vaultPath string, interval time.Duration) *Backup {
	return &Backup{
		service:   service,
		repo:      repo,
		vaultPath: vaultPath,
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the periodic backup loop.
func (b *Backup) Start() error {
	// Run once immediately
	if err := b.backupOnce(); err != nil {
		log.Printf("Drive backup initial error: %v", err)
	}

	go func() {
		ticker := time.NewTicker(b.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := b.backupOnce(); err != nil {
					log.Printf("Drive backup error: %v", err)
				}
			case <-b.stopCh:
				return
			}
		}
	}()
	return nil
}

// Stop stops the backup loop.
func (b *Backup) Stop() {
	close(b.stopCh)
}

func (b *Backup) backupOnce() error {
	return filepath.Walk(b.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden dirs (.git, .obsidian)
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Only back up .md files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		relPath, _ := filepath.Rel(b.vaultPath, path)

		rec, err := b.repo.GetDriveSyncByLocalPath(relPath)
		if err != nil {
			log.Printf("Drive backup: db error for %s: %v", relPath, err)
			return nil
		}

		modTime := info.ModTime().Truncate(time.Second)

		if rec == nil {
			// New file — upload
			ctx := context.Background()
			fileID, err := b.service.UploadFile(ctx, path, relPath, "")
			if err != nil {
				log.Printf("Drive backup: upload %s: %v", relPath, err)
				return nil
			}
			if err := b.repo.InsertDriveSync(fileID, relPath, modTime, "upload"); err != nil {
				log.Printf("Drive backup: insert sync %s: %v", relPath, err)
			}
		} else if modTime.After(rec.LastSyncedAt) {
			// Modified file — re-upload
			ctx := context.Background()
			_, err := b.service.UploadFile(ctx, path, relPath, rec.DriveFileID)
			if err != nil {
				log.Printf("Drive backup: re-upload %s: %v", relPath, err)
				return nil
			}
			if err := b.repo.UpdateDriveSync(rec.DriveFileID, modTime); err != nil {
				log.Printf("Drive backup: update sync %s: %v", relPath, err)
			}
		}

		return nil
	})
}
