package app

import (
	"context"
	"errors"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components/markdown"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/locales"
)

// A Tester can consume a worker result in content's rebuild before a click
// returns. Wait for observable UI state, rather than requiring a queued result.
func waitForUpdateState(t *testing.T, app *application, ready func() bool) {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for !ready() {
		select {
		case result := <-app.updates.results:
			app.applyUpdateResult(result)
		case <-timer.C:
			t.Fatal("timed out waiting for update UI state")
		}
	}
}

func TestInformationButtonAndCurrentVersion(t *testing.T) {
	mygo.App.SetVersion("0.0.1")
	t.Cleanup(func() { mygo.App.SetVersion("") })
	for _, language := range []locales.Language{locales.Chinese, locales.English} {
		t.Run(string(language), func(t *testing.T) {
			app, tt := newLocaleSettingsTester()
			app.language = language
			tt.Frame()
			settings, settingsOK := tt.Find(app.text().Settings)
			information, informationOK := tt.Find(app.text().Changelog)
			if !settingsOK || !informationOK || information.X < settings.X+settings.W || information.Y != settings.Y {
				t.Fatalf("information should follow settings: settings=%+v, information=%+v", settings, information)
			}
			if app.informationIcon() != lucide.Info() {
				t.Fatal("current version should use Info")
			}
			clickReplaySetting(t, tt, app.text().Changelog)
			if !app.informationOpen || !tt.HasText(applicationName) || !tt.HasText("v0.0.1") || !tt.HasText("GitHub") {
				t.Fatalf("information did not show version and GitHub: %v", tt.Texts())
			}
			if tt.HasText(app.text().InstallUpdate) {
				t.Fatal("current version should not show install")
			}
			if tt.HasText(projectGitHubURL) {
				t.Fatal("GitHub link should not show the raw URL")
			}
			assertInformationHeader(t, tt, app, 760)
			clickReplaySetting(t, tt, "GitHub")
			if urls := tt.OpenedURLs(); len(urls) != 1 || urls[0] != projectGitHubURL {
				t.Fatalf("GitHub link opened %v", urls)
			}
			tt.Key(0, ui.KeyEnter)
			if urls := tt.OpenedURLs(); len(urls) != 2 || urls[1] != projectGitHubURL {
				t.Fatalf("GitHub link did not support keyboard activation: %v", urls)
			}
		})
	}
}

func assertInformationHeader(t *testing.T, tt *ui.Tester, app *application, width float32) {
	t.Helper()
	logo, logoOK := tt.Find(fmt.Sprintf(app.text().ApplicationLogo, applicationName))
	name, nameOK := tt.Find(applicationName)
	versionText := mygo.App.Version()
	if versionText == "" {
		versionText = "dev"
	} else {
		versionText = "v" + strings.TrimPrefix(versionText, "v")
	}
	version, versionOK := tt.Find(versionText)
	github, githubOK := tt.Find("GitHub")
	near := func(a, b float32) bool { return math.Abs(float64(a-b)) < 1 }
	if !logoOK || !nameOK || !versionOK || !githubOK ||
		!near(logo.X+logo.W/2, width/2) || !near(github.X+github.W/2, width/2) ||
		!near((name.X+version.X+version.W)/2, width/2) {
		t.Fatalf("header is not centered: logo=%+v, name=%+v, version=%+v, github=%+v", logo, name, version, github)
	}
	if name.Y < logo.Y+logo.H || github.Y < name.Y+name.H || version.X <= name.X+name.W || version.H >= name.H {
		t.Fatalf("expected logo, name with small version, then GitHub: logo=%+v, name=%+v, version=%+v, github=%+v", logo, name, version, github)
	}
	// Changelog is the toolbar label without an update, or the notes heading
	// with an update. It must not also appear as a visible dialog title.
	count := 0
	for _, label := range tt.Texts() {
		if label == app.text().Changelog {
			count++
		}
	}
	if count != 1 || tt.HasText("Information") || tt.HasText("版本信息") {
		t.Fatalf("unexpected dialog title or stale naming: labels=%v", tt.Texts())
	}
}

func setTestAvailableUpdate(app *application, notes string) {
	app.updates.available = &availableUpdate{version: "0.0.2", notes: notes}
	app.updates.notes = markdown.ParseMarkdown(notes)
}

func TestInformationUpdateChangelogAndInstallPlacement(t *testing.T) {
	for _, language := range []locales.Language{locales.Chinese, locales.English} {
		t.Run(string(language), func(t *testing.T) {
			app, tt := newLocaleSettingsTester()
			app.language = language
			setTestAvailableUpdate(app, "## Fixes\n\n- Chart zoom\n- Replay controls\n\n[Release](https://github.com/Gavin-Gong/trajon/releases/tag/v0.0.2)")
			tt.Frame()
			if app.informationIcon() != lucide.Download() {
				t.Fatal("new update should use Download")
			}
			tt.SetSize(1120, 760)
			clickReplaySetting(t, tt, app.text().UpdateAvailable)
			assertInformationHeader(t, tt, app, 1120)
			for _, text := range []string{fmt.Sprintf(app.text().NewVersion, "0.0.2"), app.text().Changelog, "Fixes", "Chart zoom", "Replay controls", "Install Update"} {
				if !tt.HasText(text) {
					t.Errorf("missing update text %q: %v", text, tt.Texts())
				}
			}
			panel, _ := tt.Find(applicationName + " " + app.text().Changelog)
			button, ok := tt.Find("Install Update")
			if !ok || button.X+button.W > panel.X+panel.W || button.X < panel.X+panel.W/2 || button.Y+button.H > panel.Y+panel.H || button.Y < panel.Y+panel.H/2 {
				t.Fatalf("install should be bottom-right: panel=%+v, button=%+v", panel, button)
			}
			// Markdown links occupy inline glyphs, not the full paragraph row.
			link, _ := tt.Find("Release")
			tt.ClickAt(link.X+2, link.Y+link.H/2)
			if urls := tt.OpenedURLs(); len(urls) != 1 || !strings.HasSuffix(urls[0], "/v0.0.2") {
				t.Fatalf("release link opened %v", urls)
			}
		})
	}
}

func TestInformationDismissalRestoresStableButtonFocus(t *testing.T) {
	for _, dismissal := range []string{"close", "escape", "backdrop"} {
		t.Run(dismissal, func(t *testing.T) {
			app, tt := newLocaleSettingsTester()
			for range 2 {
				clickReplaySetting(t, tt, app.informationButtonLabel())
				// A check can finish while the dialog is open, changing the icon
				// and accessible label without replacing the opener's identity.
				setTestAvailableUpdate(app, "- Fixed replay")
				tt.Frame()
				switch dismissal {
				case "close":
					clickReplaySetting(t, tt, app.text().CloseChangelog)
				case "escape":
					tt.Key(0, ui.KeyEscape)
				case "backdrop":
					tt.ClickAt(2, 500)
				}
				for range 3 {
					tt.Frame()
				}
				if app.informationOpen || !tt.Focused(app.text().UpdateAvailable) {
					t.Fatal("dismissal did not restore focus to the updated opener")
				}
			}
		})
	}
}

func TestInformationCheckErrorAndRetry(t *testing.T) {
	app, tt := newLocaleSettingsTester()
	app.updates.check = func(context.Context) (*availableUpdate, error) { return nil, errors.New("offline") }
	t.Cleanup(app.closeUpdates)
	clickReplaySetting(t, tt, app.text().Changelog)
	waitForUpdateState(t, app, func() bool { return !app.updates.checking })
	tt.Frame()
	if !tt.HasText(app.text().UpdateCheckFailed) || !tt.HasText(app.text().Retry) || tt.HasText("Install Update") {
		t.Fatal("failed check did not offer retry")
	}
	app.updates.check = func(context.Context) (*availableUpdate, error) {
		return &availableUpdate{version: "0.0.2", notes: "- Fixed replay"}, nil
	}
	clickReplaySetting(t, tt, app.text().Retry)
	waitForUpdateState(t, app, func() bool { return !app.updates.checking })
	tt.Frame()
	if tt.HasText(app.text().UpdateCheckFailed) || !tt.HasText("Install Update") || !app.informationOpen {
		t.Fatal("successful retry did not update the open dialog")
	}
}

func TestInformationInstallDismissalAndRetry(t *testing.T) {
	app, tt := newLocaleSettingsTester()
	setTestAvailableUpdate(app, "- Fixed replay")
	started, release := make(chan struct{}, 2), make(chan error, 2)
	app.updates.available.install = func(ctx context.Context, progress func(int64, int64)) error {
		started <- struct{}{}
		progress(25, 100)
		select {
		case err := <-release:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	relaunched := false
	app.updates.relaunch = func() { relaunched = true }
	t.Cleanup(app.closeUpdates)
	tt.Frame()
	clickReplaySetting(t, tt, app.text().UpdateAvailable)
	clickReplaySetting(t, tt, "Install Update")
	<-started
	waitForUpdateState(t, app, func() bool { return app.updates.total == 100 })
	tt.Frame()
	if !tt.HasText(app.text().InstallingUpdate) {
		t.Fatal("installation progress is not visible")
	}
	clickReplaySetting(t, tt, "Install Update")
	if len(started) != 0 {
		t.Fatal("disabled install button started a second install")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if app.informationOpen || !app.updates.installing {
		t.Fatal("dismissing information canceled installation")
	}
	clickReplaySetting(t, tt, app.text().UpdateAvailable)
	release <- errors.New("download failed")
	waitForUpdateState(t, app, func() bool { return !app.updates.installing })
	tt.Frame()
	if !tt.HasText(app.text().UpdateInstallFailed) || relaunched {
		t.Fatal("failed installation did not show an error or relaunched")
	}
	clickReplaySetting(t, tt, "Install Update")
	<-started
	release <- nil
	waitForUpdateState(t, app, func() bool { return !app.updates.installing })
	if !relaunched {
		t.Fatal("successful installation did not relaunch")
	}
}

func TestInformationLongNotesKeepInstallVisible(t *testing.T) {
	app, tt := newLocaleSettingsTester()
	setTestAvailableUpdate(app, strings.Repeat("- Fixed chart interactions with a long description that wraps across the available width.\n", 40))
	tt.Frame()
	clickReplaySetting(t, tt, app.text().UpdateAvailable)
	for _, size := range [][2]int{{760, 520}, {1120, 760}} {
		tt.SetSize(size[0], size[1])
		assertInformationHeader(t, tt, app, float32(size[0]))
		panel, _ := tt.Find(applicationName + " " + app.text().Changelog)
		button, ok := tt.Find("Install Update")
		if !ok || button.Y+button.H > panel.Y+panel.H || button.Y+button.H > float32(size[1]-20) {
			t.Fatalf("long notes clipped install: size=%v, panel=%+v, button=%+v", size, panel, button)
		}
	}
}

// Opt-in screenshots follow the existing indicator evidence pattern.
func TestInformationVisualEvidence(t *testing.T) {
	dir := os.Getenv("TRAJON_INFORMATION_SHOTS")
	if dir == "" {
		t.Skip("set TRAJON_INFORMATION_SHOTS to capture rendered UI")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	mygo.App.SetVersion("0.0.1")
	t.Cleanup(func() { mygo.App.SetVersion("") })
	for _, state := range []string{"current", "available", "installing"} {
		app, tt := newLocaleSettingsTester()
		app.language = locales.Chinese
		tt.SetSize(1120, 760)
		if state != "current" {
			notes := "### 修复\n\n- 修复图表缩放与拖动行为。\n- 改进回放控制与交易操作。"
			if state == "installing" {
				notes += strings.Repeat("\n\n- 优化图表回放操作。", 20)
				app.updates.installing, app.updates.downloaded, app.updates.total = true, 30, 100
			}
			setTestAvailableUpdate(app, notes)
		}
		tt.Frame()
		clickReplaySetting(t, tt, app.informationButtonLabel())
		for _, dark := range []bool{false, true} {
			tt.SetDark(dark)
			assertInformationHeader(t, tt, app, 1120)
			file, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s-dark-%v.png", state, dark)))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(file, tt.Image()); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestChangelogDialogWidthsAndLiveExpansion(t *testing.T) {
	app, tt := newLocaleSettingsTester()
	tt.SetSize(1120, 760)
	clickReplaySetting(t, tt, app.text().Changelog)
	assertPanel := func(wantWidth float32, windowWidth, windowHeight float32) ui.Rect {
		t.Helper()
		panel, ok := tt.Find(applicationName + " " + app.text().Changelog)
		if !ok || math.Abs(float64(panel.W-wantWidth)) > 1 ||
			panel.X < 20 || panel.Y < 20 || panel.X+panel.W > windowWidth-20 || panel.Y+panel.H > windowHeight-20 {
			t.Fatalf("panel=%+v, want width=%v within %vx%v", panel, wantWidth, windowWidth, windowHeight)
		}
		assertInformationHeader(t, tt, app, windowWidth)
		return panel
	}
	assertPanel(312, 1120, 760) // 520 * (1 - 2/5)
	app.updates.checking = true
	tt.Frame()
	assertPanel(312, 1120, 760)
	app.updates.checking = false
	app.updates.checkError = errors.New("offline")
	tt.Frame()
	assertPanel(312, 1120, 760)

	// Discovering an update resizes the already-open modal without replacing
	// its centered header or losing the install/close controls.
	app.updates.checkError = nil
	var notes strings.Builder
	for i := range 40 {
		fmt.Fprintf(&notes, "- Change %02d\n", i)
	}
	setTestAvailableUpdate(app, notes.String())
	tt.Frame()
	assertPanel(780, 1120, 760) // 520 * 3/2
	firstNote, noteOK := tt.Find("Change 00")
	install, installOK := tt.Find(app.text().InstallUpdate)
	if !noteOK || !installOK || math.Abs(float64(install.Y-firstNote.Y-406)) > 1 {
		t.Fatalf("notes should get 390px plus a 16px footer gap: first=%+v, install=%+v", firstNote, install)
	}

	tt.SetSize(760, 520)
	panel := assertPanel(720, 760, 520) // Window insets cap the expanded width.
	for _, label := range []string{app.text().CloseChangelog, app.text().InstallUpdate} {
		control, ok := tt.Find(label)
		if !ok || control.X < panel.X || control.Y < panel.Y || control.X+control.W > panel.X+panel.W || control.Y+control.H > panel.Y+panel.H {
			t.Fatalf("%q falls outside the small-window panel: control=%+v, panel=%+v", label, control, panel)
		}
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if app.informationOpen || !tt.Focused(app.text().UpdateAvailable) {
		t.Fatal("resizing lost dismissal or opener focus")
	}
}

func TestInformationEmptyChangelogAndLanguageSwitch(t *testing.T) {
	app, tt := newLocaleSettingsTester()
	setTestAvailableUpdate(app, "  \n")
	tt.Frame()
	clickReplaySetting(t, tt, app.text().UpdateAvailable)
	if !tt.HasText(app.text().NoReleaseNotes) {
		t.Fatal("empty changelog did not show a fallback")
	}
	available := app.updates.available
	app.setLanguagePreference(locales.PreferChinese)
	tt.Frame()
	if !app.informationOpen || app.updates.available != available || !tt.HasText("暂无更新日志") || !tt.HasText("Install Update") {
		t.Fatal("language switch lost update state or left stale labels")
	}
}
