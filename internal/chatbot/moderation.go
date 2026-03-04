package chatbot

import "strings"

var crisisKeywords = []string{
	"suicide",
	"kill myself",
	"end my life",
	"self harm",
	"cutting",
	"overdose",
}

func DetectCrisis(message string) bool {
	lower := strings.ToLower(message)

	for _, k := range crisisKeywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func SafeRewrite(reply string) string {
	return CrisisResponse()
}
