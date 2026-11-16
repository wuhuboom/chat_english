package setting

import (
	"net/url"
	"strings"
	"unicode"
)

func safeConfigURLCharacters(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool {
		return r == '\\' || unicode.IsSpace(r) || unicode.IsControl(r)
	}) < 0
}

func SafeIndexDestination(value string) bool {
	if !safeConfigURLCharacters(value) {
		return false
	}
	u, err := url.Parse(value)
	if err != nil || u.User != nil {
		return false
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return u.Host == "" && u.Scheme == ""
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != ""
}

func SafeLandingHost(value string) bool {
	if !safeConfigURLCharacters(value) || strings.ContainsAny(value, "/?#@%") {
		return false
	}
	u, err := url.Parse("//" + value)
	return err == nil && u.Hostname() != "" && u.Host == value
}
