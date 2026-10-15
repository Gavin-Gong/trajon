package app

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"trajon/internal/locales"
)

func TestHostLocaleSynchronizesAppAndNativeControls(t *testing.T) {
	app := &application{
		language: locales.English, languagePreference: locales.PreferEnglish,
		currentInstrument: defaultInstrument(),
	}
	requested := "not called"
	app.changeLocale = func(tag string) { requested = tag }
	color, quantity := ui.Hex("#123456"), 1.0
	identity := app.currentInstrument.Identity()
	var nativeLocale string
	tt := ui.NewTester(func(c *ui.Context) {
		nativeLocale = c.Locale()
		ui.Column(c).Fill().Gap(8).Children(func() {
			components.Text(c, app.text().Overview())
			components.WithLocalization(c, app.componentLocalization(), func() {
				components.NumberInput(c.Key("quantity"), &quantity, 0, 100, 1).
					Label(app.text().OrderQuantity())
			})
			ui.ColorPicker(c.Key("native-color"), &color)
		})
	}, 760, 520)
	// Model the host event, without changing global App state or user defaults.
	publish := func(tag string) {
		app.applyLocale(tag)
		tt.SetLocale(tag)
	}
	publish("en")
	clickReplaySetting(t, tt, "Hex")
	replaceReplaySetting(tt, "A") // An incomplete draft must survive rebuilding.
	before := color
	draft := func() string {
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Key(ui.Cmd, ui.KeyC)
		return tt.Clipboard()
	}
	if got := draft(); got != "A" {
		t.Fatalf("initial incomplete draft = %q", got)
	}
	app.setLanguagePreference(locales.PreferChinese)
	if requested != "zh-Hans" || app.language != locales.English {
		t.Fatal("language choice must request the host, not guess its shown locale")
	}
	publish("zh-Hans")
	for _, label := range []string{"概览", "饱和度与亮度", "十六进制", "增加 下单数量"} {
		if _, ok := tt.Find(label); !ok {
			t.Errorf("missing localized app/native/Onevra label %q", label)
		}
	}
	if nativeLocale != "zh-Hans" || !tt.Focused("十六进制") || draft() != "A" || color != before ||
		quantity != 1 || app.currentInstrument.Identity() != identity ||
		app.shellLabels().LeftToggle != "切换侧栏" {
		t.Fatal("host locale event lost language, focus, draft or domain state")
	}

	app.setLanguagePreference(locales.System)
	if requested != "" {
		t.Fatal("System must clear the MyGo locale override")
	}
	publish("en")
	if !tt.Focused("Hex") || !tt.HasText("Overview") || draft() != "A" {
		t.Fatal("returning to System lost localized focus or the incomplete draft")
	}
	// The system can change again while following it, without a settings action.
	publish("zh-Hans")
	if app.languagePreference != locales.System || !tt.Focused("十六进制") || !tt.HasText("概览") {
		t.Fatal("system locale change did not update presentation independently of the preference")
	}
}

func TestLocaleEventsAndDevOverridesDoNotSavePreferences(t *testing.T) {
	for _, transient := range []bool{false, true} {
		t.Run(map[bool]string{false: "host-event", true: "dev-override"}[transient], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "locale.json")
			if err := saveLocalePreference(path, locales.PreferEnglish); err != nil {
				t.Fatal(err)
			}
			app := &application{
				language: locales.English, languagePreference: locales.PreferEnglish,
				localeStore: newLocalePreferenceStore(path, nil), transientLocale: transient,
			}
			t.Cleanup(func() { _ = app.localeStore.Close() })
			requested := "not called"
			app.changeLocale = func(tag string) { requested = tag }
			if transient {
				app.applyLocale("en-XA")
				if got := app.text().Overview(); got == "Overview" || got == "" {
					t.Fatal("dev pseudo locale was lost by the app's text accessors")
				}
				app.setLanguagePreference(locales.PreferChinese)
				if requested != "zh-Hans" {
					t.Fatal("dev overrides must still allow interactive language changes")
				}
			} else {
				app.applyLocale("zh-Hans")
				if app.languagePreference != locales.PreferEnglish {
					t.Fatal("host event rewrote the settings preference")
				}
			}
			app.setLanguagePreference(locales.Preference("unsupported"))
			if err := app.localeStore.Close(); err != nil {
				t.Fatal(err)
			}
			if got, err := loadLocalePreference(path); err != nil || got != locales.PreferEnglish {
				t.Fatalf("saved preference changed: %q, %v", got, err)
			}
		})
	}
}
