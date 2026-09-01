package setting

import (
	"fmt"
	"strings"
)

const (
	H5ChatTemplateConfigKey = "H5ChatTemplate"
	DefaultH5ChatTemplate   = "chat_page.html"
)

var allowedH5ChatTemplates = map[string]struct{}{
	"chat_page.html":         {},
	"chat_page1.html":        {},
	"chat_page1_notime.html": {},
	"chat_page3.html":        {},
	"chat_page3_notime.html": {},
	"chat_page3-1.html":      {},
	"chat_page4.html":        {},
	"chat_page5.html":        {},
	"chat_wx.html":           {},
}

// ValidateH5ChatTemplate only accepts templates shipped with the application.
func ValidateH5ChatTemplate(value string) (string, error) {
	template := strings.TrimSpace(value)
	if _, ok := allowedH5ChatTemplates[template]; !ok {
		return "", fmt.Errorf("不支持的H5界面模板")
	}
	return template, nil
}

// ResolveH5ChatTemplate prefers the value saved in the admin database, then
// falls back to the legacy config-file value for backward compatibility.
func ResolveH5ChatTemplate(databaseValue, configFileValue string) string {
	if template, err := ValidateH5ChatTemplate(databaseValue); err == nil {
		return template
	}
	if template, err := ValidateH5ChatTemplate(configFileValue); err == nil {
		return template
	}
	return DefaultH5ChatTemplate
}

// ResolveEntH5ChatTemplate lets an enterprise override the system template
// while preserving the existing database and config-file fallback chain.
func ResolveEntH5ChatTemplate(entValue, databaseValue, configFileValue string) string {
	if template, err := ValidateH5ChatTemplate(entValue); err == nil {
		return template
	}
	return ResolveH5ChatTemplate(databaseValue, configFileValue)
}
