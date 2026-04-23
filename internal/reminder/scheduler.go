package reminder

import (
	"context"
	"log"
	"time"

	"github.com/Nysonn/campuscare/internal/mail"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Scheduler struct {
	DB            *pgxpool.Pool
	Mailer        *mail.Mailer
	lastDailyDate string // YYYY-MM-DD of the last time daily habit jobs ran
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
	s.runDailyHabitJobs(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sendDueReminders(ctx)
			// Daily habit jobs fire once per calendar day when the local hour >= 8.
			today := time.Now().Format("2006-01-02")
			if time.Now().Hour() >= 8 && s.lastDailyDate != today {
				s.lastDailyDate = today
				s.runDailyHabitJobs(ctx)
			}
		}
	}
}

// runDailyHabitJobs sends motivation emails to students with active goals and
// missed-tracking emails for students who haven't logged the past 2 days.
func (s *Scheduler) runDailyHabitJobs(ctx context.Context) {
	s.sendMotivationEmails(ctx)
	s.sendMissedHabitEmails(ctx)
}

// ── Booking session reminders ─────────────────────────────────────────────────

type bookingReminder struct {
	ID             string
	StudentEmail   string
	StudentName    string
	CounselorEmail string
	CounselorName  string
	SessionType    string
	StartTime      time.Time
	EndTime        time.Time
	Location       string
	MeetLink       string
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

// ── Daily habit motivation emails ─────────────────────────────────────────────

func (s *Scheduler) sendMotivationEmails(ctx context.Context) {
	rows, err := s.DB.Query(ctx, `
		SELECT
			bg.id,
			bg.title,
			bg.direction,
			u.email,
			COALESCE(sp.display_name, u.email) AS student_name,
			COALESCE(
			  (SELECT COUNT(*) FROM behaviour_logs bl
			   WHERE bl.goal_id = bg.id AND bl.did_it = TRUE),
			  0
			) AS success_days
		FROM behaviour_goals bg
		JOIN users u ON u.id = bg.student_id
		LEFT JOIN student_profiles sp ON sp.user_id = bg.student_id
		WHERE bg.status = 'active'
		  AND (bg.last_motivation_sent IS NULL OR bg.last_motivation_sent < CURRENT_DATE)
	`)
	if err != nil {
		log.Printf("[habit-motivation] query error: %v", err)
		return
	}
	defer rows.Close()

	type goalRow struct {
		ID          string
		Title       string
		Direction   string
		Email       string
		StudentName string
		SuccessDays int
	}

	var goals []goalRow
	for rows.Next() {
		var g goalRow
		if err := rows.Scan(&g.ID, &g.Title, &g.Direction, &g.Email, &g.StudentName, &g.SuccessDays); err != nil {
			continue
		}
		goals = append(goals, g)
	}

	for _, g := range goals {
		body := mail.DailyMotivationTemplate(g.StudentName, g.Title, g.Direction, g.SuccessDays)
		s.Mailer.SendAsync(g.Email, "Daily reminder: keep your habit streak going! 💪", body)

		if _, err := s.DB.Exec(ctx,
			`UPDATE behaviour_goals SET last_motivation_sent = CURRENT_DATE WHERE id = $1`, g.ID,
		); err != nil {
			log.Printf("[habit-motivation] failed to update last_motivation_sent for goal %s: %v", g.ID, err)
		}
	}

	if len(goals) > 0 {
		log.Printf("[habit-motivation] sent motivation emails to %d students", len(goals))
	}
}

// ── Missed-habit notification emails ─────────────────────────────────────────

func (s *Scheduler) sendMissedHabitEmails(ctx context.Context) {
	// Find active goals that started at least 2 days ago, where there are no log
	// entries for both yesterday AND the day before, and we haven't already notified today.
	rows, err := s.DB.Query(ctx, `
		SELECT
			bg.id,
			bg.title,
			u.email,
			COALESCE(sp.display_name, u.email) AS student_name
		FROM behaviour_goals bg
		JOIN users u ON u.id = bg.student_id
		LEFT JOIN student_profiles sp ON sp.user_id = bg.student_id
		WHERE bg.status = 'active'
		  AND bg.start_date <= CURRENT_DATE - INTERVAL '2 days'
		  AND (bg.missed_notified_date IS NULL OR bg.missed_notified_date < CURRENT_DATE)
		  AND NOT EXISTS (
		      SELECT 1 FROM behaviour_logs bl
		      WHERE bl.goal_id = bg.id
		        AND bl.log_date = CURRENT_DATE - INTERVAL '1 day'
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM behaviour_logs bl
		      WHERE bl.goal_id = bg.id
		        AND bl.log_date = CURRENT_DATE - INTERVAL '2 days'
		  )
	`)
	if err != nil {
		log.Printf("[habit-missed] query error: %v", err)
		return
	}
	defer rows.Close()

	type goalRow struct {
		ID          string
		Title       string
		Email       string
		StudentName string
	}

	var goals []goalRow
	for rows.Next() {
		var g goalRow
		if err := rows.Scan(&g.ID, &g.Title, &g.Email, &g.StudentName); err != nil {
			continue
		}
		goals = append(goals, g)
	}

	for _, g := range goals {
		body := mail.HabitMissedTemplate(g.StudentName, g.Title)
		s.Mailer.SendAsync(g.Email, "We miss you — come back to your habit goal 🌱", body)

		if _, err := s.DB.Exec(ctx,
			`UPDATE behaviour_goals SET missed_notified_date = CURRENT_DATE WHERE id = $1`, g.ID,
		); err != nil {
			log.Printf("[habit-missed] failed to update missed_notified_date for goal %s: %v", g.ID, err)
		}
	}

	if len(goals) > 0 {
		log.Printf("[habit-missed] sent missed-habit emails to %d students", len(goals))
	}
}
