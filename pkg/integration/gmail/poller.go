package gmail

import (
	"context"
	"log"
	"time"
)

// Poller checks for new emails periodically
type Poller struct {
	service  *Service
	interval time.Duration
	handler  func(subject, body string) error
	stop     chan struct{}
}

// NewPoller creates a new Poller
func NewPoller(service *Service, interval time.Duration, handler func(subject, body string) error) *Poller {
	return &Poller{
		service:  service,
		interval: interval,
		handler:  handler,
		stop:     make(chan struct{}),
	}
}

// Start starts the polling loop
func (p *Poller) Start() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := p.poll(); err != nil {
				log.Printf("Gmail poll failed: %v", err)
			}
		case <-p.stop:
			return
		}
	}
}

// Stop stops the poller
func (p *Poller) Stop() {
	close(p.stop)
}

func (p *Poller) poll() error {
	ctx := context.Background()
	msgs, err := p.service.FetchUnreadEmails(ctx)
	if err != nil {
		return err
	}

	for _, msg := range msgs {
		// Extract Subject
		subject := ""
		for _, h := range msg.Payload.Headers {
			if h.Name == "Subject" {
				subject = h.Value
				break
			}
		}

		body := GetBody(msg)

		// Handle (e.g., create inbox item)
		if err := p.handler(subject, body); err != nil {
			log.Printf("Failed to handle email %s: %v", msg.Id, err)
			continue
		}

		// Mark as read (remove UNREAD label)
		// Not implemented in Service yet, but should be.
		// p.service.MarkAsRead(msg.Id)
	}
	return nil
}
