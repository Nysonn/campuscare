package calendar

import (
	"context"
	"os"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func NewService() (*calendar.Service, error) {
	ctx := context.Background()
	return calendar.NewService(
		ctx,
		option.WithCredentialsFile(os.Getenv("GOOGLE_CREDENTIALS_FILE")),
	)
}

func CreateEvent(srv *calendar.Service, summary string, start, end string) error {

	event := &calendar.Event{
		Summary: summary,
		Start: &calendar.EventDateTime{
			DateTime: start,
		},
		End: &calendar.EventDateTime{
			DateTime: end,
		},
	}

	_, err := srv.Events.Insert("primary", event).Do()
	return err
}
