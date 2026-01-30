package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Service wraps the Gmail API service
type Service struct {
	srv *gmail.Service
}

// NewService creates a new Gmail service using an authenticated HTTP client
func NewService(ctx context.Context, client *http.Client) (*Service, error) {
	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Gmail client: %w", err)
	}

	return &Service{srv: srv}, nil
}

// FetchUnreadEmails returns a list of unread emails
func (s *Service) FetchUnreadEmails(ctx context.Context) ([]*gmail.Message, error) {
	user := "me"
	r, err := s.srv.Users.Messages.List(user).Q("is:unread").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve messages: %w", err)
	}

	var messages []*gmail.Message
	for _, m := range r.Messages {
		msg, err := s.srv.Users.Messages.Get(user, m.Id).Do()
		if err != nil {
			continue // Skip if fail to get details
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetBody extracts the body from a message
func GetBody(msg *gmail.Message) string {
	// Logic to decode body from parts (text/plain vs text/html)
	// Simplified:
	if msg.Payload.Body.Data != "" {
		data, _ := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		return string(data)
	}
	return ""
}
