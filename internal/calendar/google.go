package calendar

import (
	"context"
	"os"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type CreateEventInput struct {
	Summary     string
	Description string
	Start       string
	End         string
	Location    string
	Attendees   []string
	Online      bool
}

type EventResult struct {
	EventID    string
	MeetLink   string
	HtmlLink   string
	Conference string
}

func NewService() (*calendar.Service, error) {
	ctx := context.Background()
	return calendar.NewService(
		ctx,
		option.WithCredentialsFile(os.Getenv("GOOGLE_CREDENTIALS_FILE")),
	)
}

func CreateEvent(srv *calendar.Service, input CreateEventInput) (*EventResult, error) {
	attendees := make([]*calendar.EventAttendee, 0, len(input.Attendees))
	for _, email := range input.Attendees {
		if email == "" {
			continue
		}
		attendees = append(attendees, &calendar.EventAttendee{Email: email})
	}

	event := &calendar.Event{
		Summary:     input.Summary,
		Description: input.Description,
		Location:    input.Location,
		Start:       &calendar.EventDateTime{DateTime: input.Start},
		End:         &calendar.EventDateTime{DateTime: input.End},
		Attendees:   attendees,
	}

	call := srv.Events.Insert("primary", event).SendUpdates("all")

	if input.Online {
		event.ConferenceData = &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId:             input.Start + input.End,
				ConferenceSolutionKey: &calendar.ConferenceSolutionKey{Type: "hangoutsMeet"},
			},
		}
		call = call.ConferenceDataVersion(1)
	}

	createdEvent, err := call.Do()
	if err != nil {
		return nil, err
	}

	result := &EventResult{
		EventID:  createdEvent.Id,
		MeetLink: createdEvent.HangoutLink,
		HtmlLink: createdEvent.HtmlLink,
	}

	if createdEvent.ConferenceData != nil {
		result.Conference = createdEvent.ConferenceData.ConferenceId
	}

	return result, nil
}
