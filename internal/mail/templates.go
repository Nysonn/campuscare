package mail

import "fmt"

func BookingAcceptedTemplate(studentName, counselorName, sessionType, startTime, endTime string) string {
	sessionLabel := "In-Person Session"
	meetNote := ""
	if sessionType == "online" {
		sessionLabel = "Online Session"
		meetNote = `<p style="color:#2f855a;"><strong>Note:</strong> Your counselor will send you a Google Meet link to your registered email before the session.</p>`
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
