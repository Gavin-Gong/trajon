// Package locales owns the app's JSON/ICU catalogs and typed message accessors.
package locales

import "github.com/egoist/mygo/i18n"

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

// Locale is the tag App.SetLocale accepts; an empty tag follows the system.
func (p Preference) Locale() string {
	if p == PreferChinese || p == PreferEnglish {
		return string(p)
	}
	return ""
}

// Resolve supports standalone views without a MyGo application host. Native
// windows follow App.Locale and OnLocaleChanged instead of resolving separately.
func Resolve(preference Preference, preferred ...string) Language {
	if preference == PreferChinese || preference == PreferEnglish {
		return Language(preference)
	}
	return Match(preferred...)
}

// Match uses MyGo's BCP 47/POSIX and CLDR matching, including script boundaries
// and ordered preferred languages. Unsupported languages fall back to English.
func Match(preferred ...string) Language {
	if tag := i18n.Match(preferred, Catalog.Languages()); tag != "" {
		return Language(tag)
	}
	return English
}
