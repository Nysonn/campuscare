package chatbot

import "strings"

type ResponseMode string

const (
	ResponseModeNormal               ResponseMode = "normal"
	ResponseModeClarify              ResponseMode = "clarify"
	ResponseModeReferralFirst        ResponseMode = "referral_first"
	ResponseModeLowConfidenceSupport ResponseMode = "low_confidence_support"
)

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

var vagueEmotionSignals = []string{
	"help",
	"i feel bad",
	"i feel weird",
	"i feel off",
	"not okay",
	"sad",
	"stressed",
	"anxious",
	"overwhelmed",
	"tired",
	"lost",
	"confused",
}

var persistentSevereSignals = []string{
	"for months",
	"for weeks",
	"every day",
	"all the time",
	"getting worse",
	"worse lately",
	"can't function",
	"cannot function",
	"can't cope",
	"cannot cope",
	"can't sleep for days",
	"haven't slept",
	"stopped eating",
	"missing classes",
	"failing everything",
	"falling apart",
}

var lowConfidenceSignals = []string{
	"i don't know why",
	"i dont know why",
	"not sure why",
	"something feels off",
	"maybe",
	"i guess",
	"kind of",
	"hard to explain",
	"don't know what's wrong",
	"dont know what's wrong",
	"don't know what is wrong",
	"dont know what is wrong",
}

func DetectCrisis(message string) bool {
	lower := normalizeMessage(message)

	for _, k := range crisisKeywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func DecideResponseMode(message string) ResponseMode {
	lower := normalizeMessage(message)

	if IsVagueConcern(lower) {
		return ResponseModeClarify
	}

	if SoundsPersistentOrSevere(lower) {
		return ResponseModeReferralFirst
	}

	if HasLowConfidenceSignals(lower) {
		return ResponseModeLowConfidenceSupport
	}

	return ResponseModeNormal
}

func BuildModeInstruction(mode ResponseMode) string {
	switch mode {
	case ResponseModeClarify:
		return "Use Clarify mode for this reply. Ask exactly one short clarifying question before giving advice, and keep the response brief."
	case ResponseModeReferralFirst:
		return "Use Referral-first mode for this reply. Prioritise counselor or trusted human support before any coping tips, and limit coping advice to at most one simple grounding step."
	case ResponseModeLowConfidenceSupport:
		return "Use Support mode with low confidence. Avoid specific or prescriptive advice, and offer only general grounding, validation, and support-seeking options."
	default:
		return "Use Support mode for this reply."
	}
}

func IsVagueConcern(message string) bool {
	if message == "" {
		return true
	}

	wordCount := len(strings.Fields(message))
	if wordCount > 8 {
		return false
	}

	for _, signal := range vagueEmotionSignals {
		if strings.Contains(message, signal) {
			return true
		}
	}

	return false
}

func SoundsPersistentOrSevere(message string) bool {
	for _, signal := range persistentSevereSignals {
		if strings.Contains(message, signal) {
			return true
		}
	}

	return false
}

func HasLowConfidenceSignals(message string) bool {
	for _, signal := range lowConfidenceSignals {
		if strings.Contains(message, signal) {
			return true
		}
	}

	return false
}

func normalizeMessage(message string) string {
	return strings.ToLower(strings.TrimSpace(message))
}

func SafeRewrite(reply string) string {
	return CrisisResponse()
}
