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

func CreateEvent(srv *calendar.Service, summary, start, end, studentEmail string, online bool) error {

	event := &calendar.Event{
		Summary: summary,
		Start:   &calendar.EventDateTime{DateTime: start},
		End:     &calendar.EventDateTime{DateTime: end},
		Attendees: []*calendar.EventAttendee{
			{Email: studentEmail},
		},
	}

	call := srv.Events.Insert("primary", event).SendUpdates("all")

	if online {
		event.ConferenceData = &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId:             start + studentEmail,
				ConferenceSolutionKey: &calendar.ConferenceSolutionKey{Type: "hangoutsMeet"},
			},
		}
		call = call.ConferenceDataVersion(1)
	}

	_, err := call.Do()
	return err
}
