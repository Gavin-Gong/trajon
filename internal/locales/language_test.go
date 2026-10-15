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
		{"zh-Latn", English}, {"zh_CN@latin", English}, {"zh-!", English},
		{"zh-Hans-u-ca-gregory", Chinese},
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

func TestPreferredLanguageOrder(t *testing.T) {
	for _, tc := range []struct {
		preferred []string
		want      Language
	}{
		{[]string{"fr-FR", "zh-CN", "en-US"}, Chinese},
		{[]string{"zh-TW", "en-GB", "zh-CN"}, English},
		{[]string{"en-GB", "zh-CN"}, English},
		{[]string{"zh-Hant", "ja"}, English},
		{nil, English},
	} {
		if got := Resolve(System, tc.preferred...); got != tc.want {
			t.Errorf("Resolve(System, %v) = %q, want %q", tc.preferred, got, tc.want)
		}
	}
}

func TestPreferenceLocaleTag(t *testing.T) {
	for _, tc := range []struct {
		preference Preference
		want       string
	}{
		{System, ""}, {PreferChinese, "zh-Hans"}, {PreferEnglish, "en"},
		{Preference("unsupported"), ""},
	} {
		if got := tc.preference.Locale(); got != tc.want {
			t.Errorf("%q.Locale() = %q, want %q", tc.preference, got, tc.want)
		}
	}
}

func TestCatalogsAreCompleteAndFormatsAcceptTheirArguments(t *testing.T) {
	for _, language := range []Language{Chinese, English} {
		t.Run(string(language), func(t *testing.T) {
			value := reflect.ValueOf(In(string(language)))
			for i := range value.NumMethod() {
				method := value.Method(i)
				name := value.Type().Method(i).Name
				args := make([]reflect.Value, method.Type().NumIn())
				for j := range args {
					switch method.Type().In(j).Kind() {
					case reflect.String:
						args[j] = reflect.ValueOf(fmt.Sprintf("argument%d", j))
					case reflect.Int:
						args[j] = reflect.ValueOf(17 + j)
					default:
						t.Fatalf("%s: unexpected argument type %s", name, method.Type().In(j))
					}
				}
				output := method.Call(args)[0].String()
				if strings.TrimSpace(output) == "" || strings.Contains(output, "%!") {
					t.Errorf("%s: invalid message %q", name, output)
				}
				for _, arg := range args {
					if !strings.Contains(output, fmt.Sprint(arg.Interface())) {
						t.Errorf("%s omits %v", name, arg.Interface())
					}
				}
			}
		})
	}
	if In("fr").Overview() != In("en").Overview() {
		t.Fatal("unsupported catalog did not fall back to English")
	}
}
