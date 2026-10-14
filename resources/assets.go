// Package resources embeds the application assets used by the native UI.
package resources

import _ "embed"

//go:embed icon.png
var IconPNG []byte

//go:embed github.svg
var GitHubSVG []byte
