package calendar

import (
	"context"
	"fmt"
	"time"

	googleauth "github.com/mklimuk/vault-pilot/pkg/integration/google"
	gcal "google.golang.org/api/calendar/v3"
)

// Event is a simplified calendar event used for sync.
type Event struct {
	ID          string
	Summary     string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
}

// CalendarAPI is the interface used by Syncer for testability.
type CalendarAPI interface {
	FetchUpcoming(ctx context.Context, horizon time.Duration) ([]Event, error)
	CreateEvent(ctx context.Context, e Event) (string, error)
	UpdateEvent(ctx context.Context, eventID string, e Event) error
}

// Service wraps the Google Calendar API.
type Service struct {
	srv        *gcal.Service
	calendarID string
}

// NewService creates a new Calendar service using service account credentials.
func NewService(ctx context.Context, credentialsFile, calendarID string) (*Service, error) {
	opt := googleauth.ClientOption(credentialsFile)
	srv, err := gcal.NewService(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}
	return &Service{srv: srv, calendarID: calendarID}, nil
}

// FetchUpcoming returns events within the given horizon from now.
func (s *Service) FetchUpcoming(ctx context.Context, horizon time.Duration) ([]Event, error) {
	now := time.Now()
	timeMin := now.Format(time.RFC3339)
	timeMax := now.Add(horizon).Format(time.RFC3339)

	events, err := s.srv.Events.List(s.calendarID).
		TimeMin(timeMin).
		TimeMax(timeMax).
		SingleEvents(true).
		OrderBy("startTime").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	var result []Event
	for _, item := range events.Items {
		e, err := parseGCalEvent(item)
		if err != nil {
			continue
		}
		result = append(result, e)
	}
	return result, nil
}

// CreateEvent creates a new event and returns its ID.
func (s *Service) CreateEvent(ctx context.Context, e Event) (string, error) {
	gcalEvent := toGCalEvent(e)
	created, err := s.srv.Events.Insert(s.calendarID, gcalEvent).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create event: %w", err)
	}
	return created.Id, nil
}

// UpdateEvent updates an existing event by ID.
func (s *Service) UpdateEvent(ctx context.Context, eventID string, e Event) error {
	gcalEvent := toGCalEvent(e)
	_, err := s.srv.Events.Update(s.calendarID, eventID, gcalEvent).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}
	return nil
}

func parseGCalEvent(item *gcal.Event) (Event, error) {
	start, err := parseEventTime(item.Start)
	if err != nil {
		return Event{}, fmt.Errorf("failed to parse start time: %w", err)
	}
	end, err := parseEventTime(item.End)
	if err != nil {
		return Event{}, fmt.Errorf("failed to parse end time: %w", err)
	}

	return Event{
		ID:          item.Id,
		Summary:     item.Summary,
		Description: item.Description,
		Location:    item.Location,
		StartTime:   start,
		EndTime:     end,
	}, nil
}

func parseEventTime(edt *gcal.EventDateTime) (time.Time, error) {
	if edt == nil {
		return time.Time{}, fmt.Errorf("nil event datetime")
	}
	if edt.DateTime != "" {
		return time.Parse(time.RFC3339, edt.DateTime)
	}
	if edt.Date != "" {
		return time.Parse("2006-01-02", edt.Date)
	}
	return time.Time{}, fmt.Errorf("empty event datetime")
}

func toGCalEvent(e Event) *gcal.Event {
	return &gcal.Event{
		Summary:     e.Summary,
		Description: e.Description,
		Location:    e.Location,
		Start: &gcal.EventDateTime{
			DateTime: e.StartTime.Format(time.RFC3339),
		},
		End: &gcal.EventDateTime{
			DateTime: e.EndTime.Format(time.RFC3339),
		},
	}
}
