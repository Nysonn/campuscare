package chatbot

import "strings"

var crisisKeywords = []string{
	"suicide",
	"suicidal",
	"kill myself",
	"killing myself",
	"want to die",
	"want to kill",
	"end my life",
	"ending my life",
	"take my life",
	"taking my life",
	"don't want to live",
	"don't want to be alive",
	"no reason to live",
	"self harm",
	"self-harm",
	"cutting myself",
	"hurt myself",
	"hurting myself",
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
