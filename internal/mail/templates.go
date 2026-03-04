package mail

import "fmt"

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
