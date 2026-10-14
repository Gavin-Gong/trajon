package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"trajon/resources"
)

const applicationName = "Trajon"

// Decode bundled assets before the UI starts, and reuse them across builds.
var applicationLogo = func() *ui.Bitmap {
	logo, err := ui.DecodeBitmap(resources.IconPNG)
	if err != nil {
		panic(fmt.Errorf("decode application logo: %w", err))
	}
	return logo
}()

var gitHubIcon = ui.MustParseSVG(resources.GitHubSVG)
