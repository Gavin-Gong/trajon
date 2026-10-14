package locales

import "strings"

type Language string

const (
	Chinese Language = "zh-Hans"
	English Language = "en"
)

// Preference is either a supported language or System, never a display label.
type Preference string

const (
	System        Preference = "system"
	PreferChinese Preference = Preference(Chinese)
	PreferEnglish Preference = Preference(English)
)

func (p Preference) Valid() bool {
	return p == System || p == PreferChinese || p == PreferEnglish
}

func Resolve(preference Preference, systemLocale string) Language {
	switch preference {
	case PreferChinese:
		return Chinese
	case PreferEnglish:
		return English
	default:
		return Match(systemLocale)
	}
}

// Match follows MyGo's updater language matching: accept BCP 47 and POSIX
// locales, infer Chinese scripts from regions, and use English for languages
// or scripts we do not translate (including Traditional Chinese).
func Match(locale string) Language {
	locale, _, _ = strings.Cut(strings.TrimSpace(locale), ".")
	locale, _, _ = strings.Cut(locale, "@")
	parts := strings.FieldsFunc(locale, func(r rune) bool { return r == '-' || r == '_' })
	if len(parts) == 0 || !strings.EqualFold(parts[0], "zh") {
		return English
	}
	var script, region string
	for _, part := range parts[1:] {
		switch len(part) {
		case 4:
			script = strings.ToLower(part)
		case 2, 3:
			region = strings.ToUpper(part)
		}
	}
	if script == "" {
		script = "hans"
		if region == "TW" || region == "HK" || region == "MO" {
			script = "hant"
		}
	}
	if script == "hans" {
		return Chinese
	}
	return English
}
