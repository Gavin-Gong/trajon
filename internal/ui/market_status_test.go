package ui

import (
	"errors"
	"testing"

	myui "github.com/egoist/mygo/ui"
	"trajon/internal/locales"
)

func TestRenderMarketStatusDoesNotInlineOlderLoadErrors(t *testing.T) {
	tt := myui.NewTester(func(c *myui.Context) {
		RenderMarketStatus(c, MarketStatusState{OlderError: errors.New("request failed")}, locales.In(string(locales.Chinese)), nil, nil)
	}, 900, 560)

	if tt.HasText("无法加载更早行情") || tt.HasText("request failed") || tt.HasText("重试") {
		t.Fatal("older-history failure rendered inline instead of staying silent for a toast")
	}
}
