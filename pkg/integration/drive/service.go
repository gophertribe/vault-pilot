package drive

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	googleauth "github.com/mklimuk/vault-pilot/pkg/integration/google"
	gdrive "google.golang.org/api/drive/v3"
)

// FileInfo represents metadata about a Drive file.
type FileInfo struct {
	ID         string
	Name       string
	MimeType   string
	ModifiedAt time.Time
	Size       int64
}

// DriveAPI is the interface used by Backup and Watcher for testability.
type DriveAPI interface {
	ListFiles(ctx context.Context) ([]FileInfo, error)
	UploadFile(ctx context.Context, localPath, fileName, existingFileID string) (string, error)
	DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error)
}

// Service wraps the Google Drive API.
type Service struct {
	srv      *gdrive.Service
	folderID string
}

// NewService creates a new Drive service using service account credentials.
func NewService(ctx context.Context, credentialsFile, folderID string) (*Service, error) {
	opt := googleauth.ClientOption(credentialsFile)
	srv, err := gdrive.NewService(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}
	return &Service{srv: srv, folderID: folderID}, nil
}

// ListFiles returns all files in the configured folder.
func (s *Service) ListFiles(ctx context.Context) ([]FileInfo, error) {
	query := fmt.Sprintf("'%s' in parents and trashed = false", s.folderID)
	var result []FileInfo

	pageToken := ""
	for {
		call := s.srv.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name, mimeType, modifiedTime, size)").
			Context(ctx)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		for _, f := range resp.Files {
			modTime, _ := time.Parse(time.RFC3339, f.ModifiedTime)
			result = append(result, FileInfo{
				ID:         f.Id,
				Name:       f.Name,
				MimeType:   f.MimeType,
				ModifiedAt: modTime,
				Size:       f.Size,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return result, nil
}

// UploadFile uploads a local file to the Drive folder. If existingFileID is non-empty,
// it updates the existing file; otherwise it creates a new one. Returns the file ID.
func (s *Service) UploadFile(ctx context.Context, localPath, fileName, existingFileID string) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("open local file: %w", err)
	}
	defer f.Close()

	if existingFileID != "" {
		// Update existing file
		file := &gdrive.File{Name: fileName}
		updated, err := s.srv.Files.Update(existingFileID, file).
			Media(f).
			Context(ctx).
			Do()
		if err != nil {
			return "", fmt.Errorf("update file: %w", err)
		}
		return updated.Id, nil
	}

	// Create new file
	file := &gdrive.File{
		Name:    fileName,
		Parents: []string{s.folderID},
	}
	created, err := s.srv.Files.Create(file).
		Media(f).
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	return created.Id, nil
}

// DownloadFile downloads a file from Drive by its ID.
func (s *Service) DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error) {
	resp, err := s.srv.Files.Get(fileID).Context(ctx).Download()
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}
	return resp.Body, nil
}
