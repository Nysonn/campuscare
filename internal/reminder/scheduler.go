package reminder

import (
	"context"
	"log"
	"time"

	"github.com/Nysonn/campuscare/internal/mail"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Scheduler struct {
	DB     *pgxpool.Pool
	Mailer *mail.Mailer
}

func NewScheduler(db *pgxpool.Pool, mailer *mail.Mailer) *Scheduler {
	return &Scheduler{DB: db, Mailer: mailer}
}

// Start runs the reminder loop in the background. Call once at server startup.
func (s *Scheduler) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Run once immediately on startup, then on each tick.
	s.sendDueReminders(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sendDueReminders(ctx)
		}
	}
}

type bookingReminder struct {
	ID              string
	StudentEmail    string
	StudentName     string
	CounselorEmail  string
	CounselorName   string
	SessionType     string
	StartTime       time.Time
	EndTime         time.Time
	Location        string
	MeetLink        string
}

func (s *Scheduler) sendDueReminders(ctx context.Context) {
	// Find accepted bookings whose start_time is between 29 and 31 minutes from now
	// and that haven't had a reminder sent yet.
	now := time.Now().UTC()
	windowStart := now.Add(29 * time.Minute)
	windowEnd := now.Add(31 * time.Minute)

	rows, err := s.DB.Query(ctx, `
		SELECT
			b.id,
			su.email            AS student_email,
			COALESCE(sp.full_name, su.email) AS student_name,
			cu.email            AS counselor_email,
			COALESCE(cp.full_name, cu.email) AS counselor_name,
			b.type,
			b.start_time,
			b.end_time,
			COALESCE(b.location, ''),
			COALESCE(b.google_event_id, '')
		FROM bookings b
		JOIN users su ON su.id = b.student_id
		JOIN users cu ON cu.id = b.counselor_id
		LEFT JOIN student_profiles sp ON sp.user_id = b.student_id
		LEFT JOIN counselor_profiles cp ON cp.user_id = b.counselor_id
		WHERE b.status = 'accepted'
		  AND b.reminder_sent = FALSE
		  AND b.deleted_at IS NULL
		  AND b.start_time >= $1
		  AND b.start_time < $2
	`, windowStart, windowEnd)

	if err != nil {
		log.Printf("[reminder] query error: %v", err)
		return
	}
	defer rows.Close()

	var bookings []bookingReminder
	for rows.Next() {
		var br bookingReminder
		if err := rows.Scan(
			&br.ID,
			&br.StudentEmail,
			&br.StudentName,
			&br.CounselorEmail,
			&br.CounselorName,
			&br.SessionType,
			&br.StartTime,
			&br.EndTime,
			&br.Location,
			&br.MeetLink,
		); err != nil {
			log.Printf("[reminder] scan error: %v", err)
			continue
		}
		bookings = append(bookings, br)
	}

	for _, br := range bookings {
		startFmt := br.StartTime.Format("Mon, 02 Jan 2006 15:04 MST")
		endFmt := br.EndTime.Format("15:04 MST")

		meetLink := ""
		// google_event_id is stored; meet link is not a direct column — leave blank for physical
		if br.SessionType == "online" {
			meetLink = br.MeetLink
		}

		studentBody := mail.SessionReminderStudentTemplate(
			br.StudentName, br.CounselorName, br.SessionType,
			startFmt, endFmt, br.Location, meetLink,
		)
		counselorBody := mail.SessionReminderCounselorTemplate(
			br.CounselorName, br.StudentName, br.SessionType,
			startFmt, endFmt, br.Location, meetLink,
		)

		s.Mailer.SendAsync(br.StudentEmail, "Reminder: Your session starts in 30 minutes", studentBody)
		s.Mailer.SendAsync(br.CounselorEmail, "Reminder: Upcoming counselling session in 30 minutes", counselorBody)

		// Mark reminder as sent so we don't send it again.
		if _, err := s.DB.Exec(ctx,
			`UPDATE bookings SET reminder_sent = TRUE WHERE id = $1`, br.ID,
		); err != nil {
			log.Printf("[reminder] failed to mark reminder sent for booking %s: %v", br.ID, err)
		} else {
			log.Printf("[reminder] sent 30-min reminder for booking %s", br.ID)
		}
	}
}
