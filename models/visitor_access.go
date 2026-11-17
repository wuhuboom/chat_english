package models

const VisitorAccessConfigKey = "VisitorVerificationRequired"

func VisitorVerificationRequired() bool {
	return FindConfig(VisitorAccessConfigKey) == "true"
}
