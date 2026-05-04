package mail

import "fmt"

func BookingAcceptedTemplate(studentName, counselorName, sessionType, startTime, endTime, location, meetLink string) string {
	sessionLabel := "In-Person Session"
	meetNote := ""
	locationRow := ""
	if sessionType == "online" {
		sessionLabel = "Online Session"
		if meetLink != "" {
			meetNote = `<p style="color:#2f855a;"><strong>Join link:</strong><br/><a href="` + meetLink + `">` + meetLink + `</a></p>`
		} else {
			meetNote = `<p style="color:#2f855a;"><strong>Note:</strong> This session is online. A join link will be shared with you shortly.</p>`
		}
	} else if location != "" {
		locationRow = `<tr><td style="padding:8px; font-weight:bold;">Location</td><td style="padding:8px;">` + location + `</td></tr>`
	}
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:20px;">
		<h2 style="color:#2f855a;">CampusCare</h2>
		<p>Dear ` + studentName + `,</p>
		<p>Your counselling session has been <strong>accepted</strong>.</p>
		<table style="border-collapse:collapse; width:100%; margin:16px 0;">
			<tr><td style="padding:8px; font-weight:bold;">Counselor</td><td style="padding:8px;">` + counselorName + `</td></tr>
			<tr style="background:#e6ffed;"><td style="padding:8px; font-weight:bold;">Type</td><td style="padding:8px;">` + sessionLabel + `</td></tr>
			<tr><td style="padding:8px; font-weight:bold;">Date &amp; Time</td><td style="padding:8px;">` + startTime + ` – ` + endTime + `</td></tr>
			` + locationRow + `
		</table>
		` + meetNote + `
		<p>Please be available at the scheduled time. If you need to reschedule, contact your counselor.</p>
		<p style="color:#2f855a;">Best regards,<br/>CampusCare Team</p>
	</div>`
}

func BookingDeclinedTemplate(studentName, counselorName, startTime string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#fff5f5; padding:20px;">
		<h2 style="color:#c53030;">CampusCare</h2>
		<p>Dear ` + studentName + `,</p>
		<p>Unfortunately, your counselling session request with <strong>` + counselorName + `</strong> on <strong>` + startTime + `</strong> has been <strong>declined</strong>.</p>
		<p>Please book a new session at a different time or with another counselor.</p>
		<p style="color:#c53030;">Best regards,<br/>CampusCare Team</p>
	</div>`
}

func DonationReceiptTemplate(name string, amount int64) string {
	return `
	<div style="font-family: Playfair Display, serif; background:#f0fff4; padding:20px;">
		<h2 style="color:#2f855a;">CampusCare</h2>
		<p>Dear ` + name + `,</p>
		<p>Thank you for your generous contribution of <strong>UGX ` +
		fmt.Sprint(amount) + `</strong>.</p>
		<p>Your kindness helps students access mental health support and emergency funding.</p>
		<p style="color:#2f855a;">With gratitude,<br/>CampusCare Team</p>
	</div>`
}

func OnlineMeetingStudentTemplate(studentName, counselorName, startTime, endTime, meetLink string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:20px;">
		<h2 style="color:#2f855a;">CampusCare</h2>
		<p>Dear ` + studentName + `,</p>
		<p>Your online counselling request has been <strong>accepted</strong>.</p>
		<table style="border-collapse:collapse; width:100%; margin:16px 0;">
			<tr><td style="padding:8px; font-weight:bold;">Counselor</td><td style="padding:8px;">` + counselorName + `</td></tr>
			<tr style="background:#e6ffed;"><td style="padding:8px; font-weight:bold;">Type</td><td style="padding:8px;">Online Session</td></tr>
			<tr><td style="padding:8px; font-weight:bold;">Date &amp; Time</td><td style="padding:8px;">` + startTime + ` - ` + endTime + `</td></tr>
		</table>
		<p><strong>Google Meet Link:</strong><br/><a href="` + meetLink + `">` + meetLink + `</a></p>
		<p>Please join a few minutes early.</p>
		<p style="color:#2f855a;">Best regards,<br/>CampusCare Team</p>
	</div>`
}

func OnlineMeetingCounselorTemplate(counselorName, studentName, startTime, endTime, meetLink string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#ebf8ff; padding:20px;">
		<h2 style="color:#2b6cb0;">CampusCare</h2>
		<p>Dear ` + counselorName + `,</p>
		<p>You have accepted an <strong>online counselling session</strong>.</p>
		<table style="border-collapse:collapse; width:100%; margin:16px 0;">
			<tr><td style="padding:8px; font-weight:bold;">Student</td><td style="padding:8px;">` + studentName + `</td></tr>
			<tr style="background:#bee3f8;"><td style="padding:8px; font-weight:bold;">Type</td><td style="padding:8px;">Online Session</td></tr>
			<tr><td style="padding:8px; font-weight:bold;">Date &amp; Time</td><td style="padding:8px;">` + startTime + ` - ` + endTime + `</td></tr>
		</table>
		<p><strong>Google Meet Link:</strong><br/><a href="` + meetLink + `">` + meetLink + `</a></p>
		<p style="color:#2b6cb0;">Best regards,<br/>CampusCare Team</p>
	</div>`
}

func WelcomeTemplate(name, role string) string {
	roleLabel := "student"
	roleNote := "You can now book counselling sessions, create campaigns, and access mental health resources."
	if role == "counselor" {
		roleLabel = "counsellor"
		roleNote = "You can now manage appointment requests, conduct online and physical sessions, and support students who need you."
	}
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#2f855a; margin-bottom:4px;">Welcome to CampusCare</h2>
		<p style="color:#4a5568;">Hi ` + name + `,</p>
		<p style="color:#4a5568;">
			Your account has been created successfully as a <strong>` + roleLabel + `</strong>.
			` + roleNote + `
		</p>
		<p style="color:#4a5568;">If you have any questions or need help getting started, feel free to reach out to our support team.</p>
		<p style="margin-top:24px; color:#2f855a;">Welcome aboard,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

func SponsorRequestReceivedTemplate(sponsorName, requesterName string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#2f855a; margin-bottom:4px;">CampusCare — Sponsor Request</h2>
		<p style="color:#4a5568;">Hi ` + sponsorName + `,</p>
		<p style="color:#4a5568;">
			A fellow student, <strong>` + requesterName + `</strong>, has reached out and is hoping you could be their sponsor.
			As a sponsor, you offer a safe space for someone to share what they're going through and receive encouragement
			on their wellbeing journey.
		</p>
		<p style="color:#4a5568;">
			Log in to your CampusCare dashboard to review their request and decide whether you'd like to accept.
			There's no pressure — only accept if you feel ready to support someone at this time.
		</p>
		<p style="margin-top:24px; color:#2f855a;">With care,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

func SponsorRequestAcceptedTemplate(sponseeName, sponsorName string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#2f855a; margin-bottom:4px;">CampusCare — Your Request Was Accepted!</h2>
		<p style="color:#4a5568;">Hi ` + sponseeName + `,</p>
		<p style="color:#4a5568;">
			Great news! <strong>` + sponsorName + `</strong> has accepted your request to be your sponsor.
		</p>
		<p style="color:#4a5568;">
			You can now open a private conversation with them directly from your CampusCare dashboard.
			This is your space — feel free to share how you're doing, ask for advice, or simply talk things through.
			Your sponsor is here to listen without judgement.
		</p>
		<p style="color:#4a5568; font-style:italic;">
			Remember: this is a peer support connection. For urgent mental health concerns, please also reach out
			to a professional counsellor through your bookings section.
		</p>
		<p style="margin-top:24px; color:#2f855a;">Take care,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

func SponsorRequestDeclinedTemplate(sponseeName, sponsorName string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#fffbeb; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#b7791f; margin-bottom:4px;">CampusCare — Sponsor Request Update</h2>
		<p style="color:#4a5568;">Hi ` + sponseeName + `,</p>
		<p style="color:#4a5568;">
			<strong>` + sponsorName + `</strong> was unable to take on a new sponsee at this time.
			Please don't take this personally — sponsors sometimes reach capacity or have their own commitments.
		</p>
		<p style="color:#4a5568;">
			You can browse other sponsors in your dashboard and send a new request whenever you feel ready.
			You're also always welcome to book a session with one of our professional counsellors.
		</p>
		<p style="margin-top:24px; color:#b7791f;">Take care,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

func SponsorshipTerminatedTemplate(sponseeName, sponsorName string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#fff5f5; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#c53030; margin-bottom:4px;">CampusCare — Sponsorship Ended</h2>
		<p style="color:#4a5568;">Hi ` + sponseeName + `,</p>
		<p style="color:#4a5568;">
			Your sponsorship with <strong>` + sponsorName + `</strong> has come to an end.
			This can happen when a sponsor needs to step back for their own wellbeing.
		</p>
		<p style="color:#4a5568;">
			Your conversations and support history are preserved. You're welcome to look for a new sponsor
			from your dashboard, or book a session with a professional counsellor if you need additional support
			during this transition.
		</p>
		<p style="color:#4a5568; font-style:italic;">
			You are not alone, and help is always available on CampusCare.
		</p>
		<p style="margin-top:24px; color:#c53030;">With care,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

// SponsorTerminatedBySponsoreeTemplate notifies the sponsor that their sponsee ended the sponsorship.
func SponsorTerminatedBySponsoreeTemplate(sponsorName, sponseeName string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#fff5f5; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#c53030; margin-bottom:4px;">CampusCare — Sponsorship Ended</h2>
		<p style="color:#4a5568;">Hi ` + sponsorName + `,</p>
		<p style="color:#4a5568;">
			<strong>` + sponseeName + `</strong> has decided to end your sponsorship connection.
			This is a personal decision and is not a reflection of the support you provided.
		</p>
		<p style="color:#4a5568;">
			You are still listed as an active sponsor and can connect with other students who need support.
			Thank you for being part of the CampusCare community.
		</p>
		<p style="margin-top:24px; color:#c53030;">With gratitude,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

func BookingAcceptedCounselorTemplate(counselorName, studentName, sessionType, startTime, endTime, location, meetLink string) string {
	sessionLabel := "In-Person Session"
	meetRow := ""
	locationRow := ""
	if sessionType == "online" {
		sessionLabel = "Online Session"
		if meetLink != "" {
			meetRow = `<tr><td style="padding:8px; font-weight:bold;">Join Link</td><td style="padding:8px;"><a href="` + meetLink + `">` + meetLink + `</a></td></tr>`
		}
	} else if location != "" {
		locationRow = `<tr><td style="padding:8px; font-weight:bold;">Location</td><td style="padding:8px;">` + location + `</td></tr>`
	}
	return `
	<div style="font-family: Arial, sans-serif; background:#ebf8ff; padding:20px;">
		<h2 style="color:#2b6cb0;">CampusCare</h2>
		<p>Dear ` + counselorName + `,</p>
		<p>You have <strong>confirmed</strong> a counselling session with the following details:</p>
		<table style="border-collapse:collapse; width:100%; margin:16px 0;">
			<tr><td style="padding:8px; font-weight:bold;">Student</td><td style="padding:8px;">` + studentName + `</td></tr>
			<tr style="background:#bee3f8;"><td style="padding:8px; font-weight:bold;">Type</td><td style="padding:8px;">` + sessionLabel + `</td></tr>
			<tr><td style="padding:8px; font-weight:bold;">Date &amp; Time</td><td style="padding:8px;">` + startTime + ` – ` + endTime + `</td></tr>
			` + locationRow + `
			` + meetRow + `
		</table>
		<p>The student has been notified. Please be available at the scheduled time.</p>
		<p style="color:#2b6cb0;">Best regards,<br/>CampusCare Team</p>
	</div>`
}

func CampaignApprovedTemplate(studentName, campaignTitle string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#2f855a; margin-bottom:4px;">CampusCare — Campaign Approved!</h2>
		<p style="color:#4a5568;">Hi ` + studentName + `,</p>
		<p style="color:#4a5568;">
			We're pleased to let you know that your campaign <strong>"` + campaignTitle + `"</strong>
			has been <strong>approved</strong> by the CampusCare team.
		</p>
		<p style="color:#4a5568;">
			Your campaign is now publicly visible and can start receiving contributions from the community.
			You can track donations and manage your campaign from your CampusCare dashboard.
		</p>
		<p style="color:#4a5568; font-style:italic;">
			Thank you for trusting CampusCare as a platform to share your story and reach out for support.
		</p>
		<p style="margin-top:24px; color:#2f855a;">With care,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

func CounselorApprovedTemplate(fullName string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#2f855a; margin-bottom:4px;">CampusCare — You're Approved!</h2>
		<p style="color:#4a5568;">Hi ` + fullName + `,</p>
		<p style="color:#4a5568;">
			Great news! Your CampusCare counsellor account has been <strong>verified and approved</strong>
			by our admin team.
		</p>
		<p style="color:#4a5568;">
			You can now log in to your dashboard to manage appointment requests, conduct online and
			physical sessions, and start supporting students who need you.
		</p>
		<p style="color:#4a5568; font-style:italic;">
			Thank you for joining CampusCare. Your expertise makes a real difference in student lives.
		</p>
		<p style="margin-top:24px; color:#2f855a;">Welcome aboard,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

func PasswordResetTemplate(name, resetLink string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#2f855a; margin-bottom:4px;">CampusCare — Password Reset</h2>
		<p style="color:#4a5568;">Hi ` + name + `,</p>
		<p style="color:#4a5568;">
			We received a request to reset the password for your CampusCare account.
			Click the button below to choose a new password. This link expires in <strong>1 hour</strong>.
		</p>
		<div style="text-align:center; margin:32px 0;">
			<a href="` + resetLink + `"
			   style="background:#2f855a; color:#fff; text-decoration:none; padding:14px 32px;
			          border-radius:8px; font-weight:bold; font-size:15px; display:inline-block;">
				Reset My Password
			</a>
		</div>
		<p style="color:#718096; font-size:13px;">
			If the button above doesn't work, copy and paste this link into your browser:<br/>
			<a href="` + resetLink + `" style="color:#2f855a; word-break:break-all;">` + resetLink + `</a>
		</p>
		<p style="color:#718096; font-size:13px; margin-top:24px;">
			If you did not request a password reset, you can safely ignore this email.
			Your password will not change.
		</p>
		<p style="margin-top:24px; color:#2f855a;">The CampusCare Team</p>
	</div>`
}

func SessionReminderStudentTemplate(studentName, counselorName, sessionType, startTime, endTime, location, meetLink string) string {
	sessionLabel := "In-Person Session"
	detailRow := ""
	if sessionType == "online" {
		sessionLabel = "Online Session"
		if meetLink != "" {
			detailRow = `<tr><td style="padding:8px; font-weight:bold;">Join Link</td><td style="padding:8px;"><a href="` + meetLink + `">` + meetLink + `</a></td></tr>`
		}
	} else if location != "" {
		detailRow = `<tr><td style="padding:8px; font-weight:bold;">Location</td><td style="padding:8px;">` + location + `</td></tr>`
	}
	return `
	<div style="font-family: Arial, sans-serif; background:#fffbeb; padding:20px; max-width:560px; margin:auto;">
		<h2 style="color:#b7791f;">CampusCare — Session Reminder</h2>
		<p>Dear ` + studentName + `,</p>
		<p>This is a friendly reminder that your counselling session starts in <strong>30 minutes</strong>.</p>
		<table style="border-collapse:collapse; width:100%; margin:16px 0;">
			<tr><td style="padding:8px; font-weight:bold;">Counselor</td><td style="padding:8px;">` + counselorName + `</td></tr>
			<tr style="background:#fefcbf;"><td style="padding:8px; font-weight:bold;">Type</td><td style="padding:8px;">` + sessionLabel + `</td></tr>
			<tr><td style="padding:8px; font-weight:bold;">Date &amp; Time</td><td style="padding:8px;">` + startTime + ` – ` + endTime + `</td></tr>
			` + detailRow + `
		</table>
		<p>Please make sure you are ready and available at the scheduled time.</p>
		<p style="color:#b7791f;">Best regards,<br/>CampusCare Team</p>
	</div>`
}

func SessionReminderCounselorTemplate(counselorName, studentName, sessionType, startTime, endTime, location, meetLink string) string {
	sessionLabel := "In-Person Session"
	detailRow := ""
	if sessionType == "online" {
		sessionLabel = "Online Session"
		if meetLink != "" {
			detailRow = `<tr><td style="padding:8px; font-weight:bold;">Join Link</td><td style="padding:8px;"><a href="` + meetLink + `">` + meetLink + `</a></td></tr>`
		}
	} else if location != "" {
		detailRow = `<tr><td style="padding:8px; font-weight:bold;">Location</td><td style="padding:8px;">` + location + `</td></tr>`
	}
	return `
	<div style="font-family: Arial, sans-serif; background:#ebf8ff; padding:20px; max-width:560px; margin:auto;">
		<h2 style="color:#2b6cb0;">CampusCare — Session Reminder</h2>
		<p>Dear ` + counselorName + `,</p>
		<p>This is a reminder that you have a counselling session starting in <strong>30 minutes</strong>.</p>
		<table style="border-collapse:collapse; width:100%; margin:16px 0;">
			<tr><td style="padding:8px; font-weight:bold;">Student</td><td style="padding:8px;">` + studentName + `</td></tr>
			<tr style="background:#bee3f8;"><td style="padding:8px; font-weight:bold;">Type</td><td style="padding:8px;">` + sessionLabel + `</td></tr>
			<tr><td style="padding:8px; font-weight:bold;">Date &amp; Time</td><td style="padding:8px;">` + startTime + ` – ` + endTime + `</td></tr>
			` + detailRow + `
		</table>
		<p>Please be prepared and available for your student at the scheduled time.</p>
		<p style="color:#2b6cb0;">Best regards,<br/>CampusCare Team</p>
	</div>`
}

func NewSponsorTemplate(sponsorName string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#2f855a; margin-bottom:4px;">CampusCare — You're Now a Sponsor!</h2>
		<p style="color:#4a5568;">Hi ` + sponsorName + `,</p>
		<p style="color:#4a5568;">
			Thank you for stepping up to support your fellow students. You are now listed as an
			<strong>active sponsor</strong> on CampusCare.
		</p>
		<p style="color:#4a5568;">
			Students who are looking for peer support will be able to find your profile and send you
			a connection request. You can review and accept or decline requests at any time from
			your CampusCare dashboard.
		</p>
		<p style="color:#4a5568; font-style:italic;">
			Your presence as a sponsor makes a real difference. Thank you for being part of our community.
		</p>
		<p style="margin-top:24px; color:#2f855a;">With gratitude,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

func HabitGoalCreatedTemplate(studentName, goalTitle, direction, startDate, endDate string) string {
	directionLabel := "build the habit of"
	directionTip := "Consistency is built one day at a time. Even small steps forward count."
	if direction == "quit" {
		directionLabel = "quit"
		directionTip := "Every day you resist is a victory. Be proud of every single one."
		_ = directionTip
	}
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#2f855a; margin-bottom:4px;">CampusCare — Your Goal is Set! 🎯</h2>
		<p style="color:#4a5568;">Hi ` + studentName + `,</p>
		<p style="color:#4a5568;">
			You've just taken a powerful step. Your new behaviour goal has been created on CampusCare:
		</p>
		<div style="background:#e6ffed; border-left:4px solid #2f855a; padding:16px; margin:20px 0; border-radius:6px;">
			<p style="margin:0; font-weight:bold; color:#276749; font-size:16px;">` + goalTitle + `</p>
			<p style="margin:6px 0 0; color:#4a5568; font-size:13px;">
				Goal: <strong>` + directionLabel + ` "` + goalTitle + `"</strong><br/>
				Period: <strong>` + startDate + `</strong> → <strong>` + endDate + `</strong>
			</p>
		</div>
		<p style="color:#4a5568;">` + directionTip + `</p>
		<p style="color:#4a5568;">
			Log your progress each day from your <strong>Behaviour</strong> section on the CampusCare dashboard.
			We'll be cheering you on every step of the way.
		</p>
		<div style="text-align:center; margin:28px 0;">
			<a href="http://campuscare.me/" style="background:#2f855a; color:#fff; text-decoration:none; padding:12px 28px; border-radius:8px; font-weight:bold; font-size:15px; display:inline-block;">Open CampusCare</a>
		</div>
		<p style="margin-top:24px; color:#2f855a;">You've got this,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

func DailyMotivationTemplate(studentName, goalTitle, direction string, successDays int) string {
	actionWord := "build"
	if direction == "quit" {
		actionWord = "resist"
	}
	successNote := fmt.Sprintf("You've already succeeded <strong>%d day(s)</strong> — that's real progress.", successDays)
	if successDays == 0 {
		successNote = "Every journey starts with a single step. Today is your chance to begin."
	}
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#2f855a; margin-bottom:4px;">CampusCare — Keep Going! 💪</h2>
		<p style="color:#4a5568;">Hi ` + studentName + `,</p>
		<p style="color:#4a5568;">
			Just a quick note to remind you about your active goal on CampusCare:
		</p>
		<div style="background:#e6ffed; border-left:4px solid #2f855a; padding:16px; margin:20px 0; border-radius:6px;">
			<p style="margin:0; font-weight:bold; color:#276749; font-size:15px;">` + goalTitle + `</p>
		</div>
		<p style="color:#4a5568;">` + successNote + `</p>
		<p style="color:#4a5568;">
			Remember to ` + actionWord + ` your habit today and log it in your dashboard. Small daily actions
			create lasting change. You are capable of more than you know.
		</p>
		<p style="color:#4a5568; font-style:italic;">
			"It does not matter how slowly you go as long as you do not stop." — Confucius
		</p>
		<div style="text-align:center; margin:28px 0;">
			<a href="http://campuscare.me/" style="background:#2f855a; color:#fff; text-decoration:none; padding:12px 28px; border-radius:8px; font-weight:bold; font-size:15px; display:inline-block;">Log Today's Progress</a>
		</div>
		<p style="margin-top:24px; color:#2f855a;">Believing in you,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

func HabitMissedTemplate(studentName, goalTitle string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#fffbeb; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#b7791f; margin-bottom:4px;">CampusCare — We Miss You! 🌱</h2>
		<p style="color:#4a5568;">Hi ` + studentName + `,</p>
		<p style="color:#4a5568;">
			We noticed you haven't logged your habit for the past two days on CampusCare:
		</p>
		<div style="background:#fefcbf; border-left:4px solid #b7791f; padding:16px; margin:20px 0; border-radius:6px;">
			<p style="margin:0; font-weight:bold; color:#744210; font-size:15px;">` + goalTitle + `</p>
		</div>
		<p style="color:#4a5568;">
			That's okay — life gets busy and sometimes we lose track. What matters is that you come back.
			Streaks can be rebuilt, and your commitment to yourself is what counts.
		</p>
		<p style="color:#4a5568;">
			Log today's progress from your CampusCare dashboard and keep your momentum going. We're rooting for you!
		</p>
		<p style="color:#4a5568; font-style:italic;">
			"Fall seven times, stand up eight." — Japanese Proverb
		</p>
		<div style="text-align:center; margin:28px 0;">
			<a href="http://campuscare.me/" style="background:#b7791f; color:#fff; text-decoration:none; padding:12px 28px; border-radius:8px; font-weight:bold; font-size:15px; display:inline-block;">Return to CampusCare</a>
		</div>
		<p style="margin-top:24px; color:#b7791f;">Come back strong,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}

// SponsorChatNotificationTemplate notifies a user that their sponsor/sponsee
// has sent them a message and they should log in to reply.
func SponsorChatNotificationTemplate(recipientName, senderName, senderRole string) string {
	return `
	<div style="font-family: Arial, sans-serif; background:#f0fff4; padding:32px; max-width:560px; margin:auto;">
		<h2 style="color:#2f855a; margin-bottom:4px;">CampusCare — New Message</h2>
		<p style="color:#4a5568;">Hi ` + recipientName + `,</p>
		<p style="color:#4a5568;">
			Your ` + senderRole + ` <strong>` + senderName + `</strong> has sent you a message on CampusCare.
		</p>
		<p style="color:#4a5568;">
			Log in to your dashboard to read and reply.
		</p>
		<p style="margin-top:24px; color:#2f855a;">With care,<br/><strong>The CampusCare Team</strong></p>
	</div>`
}
