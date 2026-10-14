package locales

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestMatchSystemLocale(t *testing.T) {
	for _, tc := range []struct {
		locale string
		want   Language
	}{
		{"zh", Chinese}, {"zh-CN", Chinese}, {"zh-SG", Chinese}, {"ZH_hans_cn.UTF-8", Chinese},
		{"zh_CN.UTF-8@variant", Chinese}, {" zh-Hans-HK ", Chinese},
		{"zh-TW", English}, {"zh_HK.UTF-8", English}, {"zh-MO", English}, {"zh-Hant-CN", English},
		{"en", English}, {"en-GB", English}, {"en_US.UTF-8", English},
		{"ja-JP", English}, {"fr-FR", English}, {"C", English}, {"", English}, {"unknown", English},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			if got := Match(tc.locale); got != tc.want {
				t.Fatalf("Match(%q) = %q, want %q", tc.locale, got, tc.want)
			}
		})
	}
}

func TestResolveLanguagePreference(t *testing.T) {
	for _, tc := range []struct {
		preference Preference
		locale     string
		want       Language
	}{
		{System, "zh-CN", Chinese}, {System, "en-US", English},
		{PreferChinese, "en-US", Chinese}, {PreferEnglish, "zh-CN", English},
		{Preference("unsupported"), "zh-CN", Chinese},
	} {
		if got := Resolve(tc.preference, tc.locale); got != tc.want {
			t.Errorf("Resolve(%q, %q) = %q, want %q", tc.preference, tc.locale, got, tc.want)
		}
	}
	for _, preference := range []Preference{System, PreferChinese, PreferEnglish} {
		if !preference.Valid() {
			t.Errorf("supported preference %q rejected", preference)
		}
	}
	for _, preference := range []Preference{"", "zh-CN", "EN", "fr"} {
		if preference.Valid() {
			t.Errorf("unsupported preference %q accepted", preference)
		}
	}
}

func TestCatalogsAreCompleteAndFormatsAcceptTheirArguments(t *testing.T) {
	formats := map[string]int{
		"ChooseInstrumentSymbol": 1, "HistoryInstrument": 1, "InstrumentLoadFailed": 2,
		"MissingSource": 1, "NoCandles": 1, "CandleChart": 1,
		"ApplicationLogo": 1, "NewVersion": 1,
		"IncreaseNumber": 1, "DecreaseNumber": 1, "AdjustNumber": 1,
	}
	for _, language := range []Language{Chinese, English} {
		t.Run(string(language), func(t *testing.T) {
			value := reflect.ValueOf(For(language)).Elem()
			for i := range value.NumField() {
				name, text := value.Type().Field(i).Name, value.Field(i).String()
				if strings.TrimSpace(text) == "" {
					t.Errorf("missing %s", name)
				}
				count, formatted := formats[name]
				if !formatted {
					continue
				}
				args := make([]any, count)
				for j := range args {
					args[j] = fmt.Sprintf("argument%d", j)
				}
				output := fmt.Sprintf(text, args...)
				if strings.Contains(output, "%!") {
					t.Errorf("%s: invalid format %q", name, text)
				}
				for _, arg := range args {
					if !strings.Contains(output, arg.(string)) {
						t.Errorf("%s omits %q", name, arg)
					}
				}
			}
		})
	}
	if For(Language("fr")) != For(English) {
		t.Fatal("unsupported catalog did not fall back to English")
	}
}
