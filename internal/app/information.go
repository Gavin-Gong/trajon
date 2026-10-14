package app

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/components/markdown"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"github.com/oneo-me/onevra/source/shells/sidebar"
)

func (a *application) informationIcon() *ui.SVG {
	if a.updates.available != nil {
		return lucide.Download()
	}
	return lucide.Info()
}

func (a *application) informationButtonLabel() string {
	if a.updates.available != nil {
		return a.text().UpdateAvailable
	}
	return a.text().Changelog
}

func (a *application) informationToolbarButton(c *ui.Context) {
	sidebar.IconButton(c.Key("information"), a.informationIcon(), a.informationButtonLabel()).
		Bind(&a.informationButton).OnClick(func() {
		a.informationOpen = true
		a.checkForUpdates()
	})
}

func (a *application) closeInformation() {
	a.informationOpen = false
	a.informationButton.Focus()
}

func (a *application) informationDialog(c *ui.Context) {
	text, s := a.text(), &a.updates
	wasOpen := a.informationOpen
	services := c.Services()
	components.Dialog(c.Key("information-dialog"), &a.informationOpen, components.DialogOptions{Label: applicationName + " " + text.Changelog}, func(panel ui.Element) {
		width := float32(312)
		if s.available != nil {
			width = 780
		}
		panel.Width(width).Padding(18).Gap(16)
		components.Button(c.Key("information-close"), "", components.ButtonGhost).
			Absolute().Top(12).Right(12).Size(28, 28).Padding(0).Radius(14).Center().
			Label(text.CloseChangelog).AutoFocus().OnClick(a.closeInformation).Children(func() {
			components.Icon(c, lucide.X).Size(15, 15)
		})
		version := mygo.App.Version()
		if version == "" {
			version = "dev"
		} else {
			version = "v" + strings.TrimPrefix(version, "v")
		}
		ui.Column(c.Key("information-identity")).FillWidth().AlignItems(ui.Center).Gap(10).Shrink(0).Children(func() {
			components.Image(c.Key("application-logo"), applicationLogo).Size(80, 80).
				Background(ui.Hex("#ffffff")).Radius(16).Label(fmt.Sprintf(text.ApplicationLogo, applicationName))
			ui.Row(c).FillWidth().Justify(ui.Center).AlignItems(ui.Center).Gap(6).Children(func() {
				components.Text(c, applicationName).FontSize(22).Bold().SingleLine()
				components.Text(c, version).FontSize(12).TextColor(c.Theme().TextMuted).SingleLine()
			})
			components.Button(c.Key("information-github"), "", components.ButtonGhost).
				Size(32, 32).Padding(0).Center().Role(ui.RoleLink).Label("GitHub").Tooltip("GitHub").
				OnClick(func() { services.OpenURL(projectGitHubURL) }).Children(func() {
				components.Icon(c, gitHubIcon).Size(20, 20)
			})
		})
		if s.available != nil {
			components.Text(c, fmt.Sprintf(text.NewVersion, s.available.version)).Bold().SingleLine().Shrink(0)
			components.Text(c, text.Changelog).Bold().SingleLine().Shrink(0)
			components.ScrollView(c.Key("update-notes"), components.ScrollViewOptions{}).
				FillWidth().MaxHeight(390).MinHeight(0).Shrink(1).Children(func() {
				if strings.TrimSpace(s.available.notes) == "" {
					components.Text(c, text.NoReleaseNotes).TextColor(c.Theme().TextMuted).Wrap()
				} else {
					s.notes.Render(c, markdown.MarkdownOptions{OnLink: func(destination string) {
						// Release notes are remote content; only open web URLs.
						if link, err := url.Parse(destination); err == nil && (link.Scheme == "https" || link.Scheme == "http") {
							services.OpenURL(link.String())
						}
					}}).FillWidth()
				}
			})
		}
		if s.checking {
			components.Text(c, text.CheckingUpdates).TextColor(c.Theme().TextMuted).SingleLine().Shrink(0)
		}
		if s.checkError != nil {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
				components.Text(c, text.UpdateCheckFailed).TextColor(c.Theme().TextMuted).
					Tooltip(s.checkError.Error()).Wrap().Grow(1).MinWidth(0)
				components.Button(c.Key("update-check-retry"), text.Retry, components.ButtonSecondary).OnClick(a.checkForUpdates)
			})
		}
		if s.installing || s.installed {
			components.Text(c, text.InstallingUpdate).TextColor(c.Theme().TextMuted).SingleLine().Shrink(0)
			if s.total > 0 {
				components.Progress(c, float64(s.downloaded)/float64(s.total)).FillWidth().Shrink(0)
			}
		}
		if s.installError != nil {
			components.Text(c, text.UpdateInstallFailed).TextColor(c.Theme().TextMuted).
				Tooltip(s.installError.Error()).Wrap().Shrink(0)
		}
		if s.available != nil {
			ui.Row(c).FillWidth().Justify(ui.End).Shrink(0).Children(func() {
				components.Button(c.Key("install-update"), text.InstallUpdate, components.ButtonPrimary).
					Disabled(s.installing || s.installed).OnClick(a.installUpdate)
			})
		}
	})
	if wasOpen && !a.informationOpen {
		a.closeInformation()
	}
}
