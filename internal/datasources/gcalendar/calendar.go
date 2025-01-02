package gcalendar

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/michaelgalloway/sophia/internal/auth"
	"github.com/michaelgalloway/sophia/internal/datasources"
)

type GoogleCalendarSource struct {
	service     *calendar.Service
	creds       []byte
	tokenDir    string
	tokenMgr    *auth.TokenManager
}

type Config struct {
	Credentials string
	TokenDir    string
}

func New(config map[string]interface{}) (datasources.DataSource, error) {
	credentials, ok := config["credentials"].(string)
	if !ok {
		return nil, fmt.Errorf("credentials not provided in config")
	}

	tokenDir, ok := config["token_dir"].(string)
	if !ok {
		return nil, fmt.Errorf("token_dir not provided in config")
	}

	return &GoogleCalendarSource{
		creds:    []byte(credentials),
		tokenDir: tokenDir,
		tokenMgr: auth.NewTokenManager(tokenDir),
	}, nil
}

func (g *GoogleCalendarSource) Name() string {
	return "google_calendar"
}

func (g *GoogleCalendarSource) Initialize(ctx context.Context) error {
	// Create OAuth2 config from credentials
	config, err := google.ConfigFromJSON(g.creds, calendar.CalendarReadonlyScope)
	if err != nil {
		return fmt.Errorf("failed to parse client secret file to config: %w", err)
	}

	// Get OAuth2 token
	token, err := g.tokenMgr.GetToken(ctx, config, "calendar")
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Create HTTP client with token
	client := config.Client(ctx, token)

	// Create the Calendar service
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create calendar service: %w", err)
	}

	g.service = service
	return nil
}

func (g *GoogleCalendarSource) FetchData(ctx context.Context, since time.Time) ([]datasources.Document, error) {
	if g.service == nil {
		return nil, fmt.Errorf("calendar service not initialized")
	}

	timeMin := time.Now().Format(time.RFC3339)
	timeMax := time.Date(2025, time.January, 10, 23, 0, 0, 0, time.UTC).Format(time.RFC3339)

	events, err := g.service.Events.List("primary").
		TimeMin(timeMin).
		TimeMax(timeMax).
		OrderBy("startTime").
		SingleEvents(true).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calendar events: %w", err)
	}

	var docs []datasources.Document
	for _, event := range events.Items {
		content := fmt.Sprintf("Event: %s\nDescription: %s\nStart: %s\nEnd: %s\nAttendees: %s",
			event.Summary,
			event.Description,
			event.Start.DateTime,
			event.End.DateTime,
			formatAttendees(event.Attendees),
		)

		doc := datasources.Document{
			ID:      event.Id,
			Content: content,
			Metadata: map[string]interface{}{
				"summary":    event.Summary,
				"start_time": event.Start.DateTime,
				"end_time":   event.End.DateTime,
				"location":   event.Location,
			},
			Source:    g.Name(),
			Timestamp: time.Now(),
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

func formatAttendees(attendees []*calendar.EventAttendee) string {
	if len(attendees) == 0 {
		return "No attendees"
	}

	result := ""
	for i, attendee := range attendees {
		if i > 0 {
			result += ", "
		}
		result += attendee.Email
	}
	return result
}
