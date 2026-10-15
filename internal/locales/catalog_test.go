package locales

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
)

func TestCatalogsDeclareAppLanguagesAndGeneratedMessages(t *testing.T) {
	want := []string{"en", "zh-Hans"}
	if got := mygo.App.Languages(); !slices.Equal(got, want) {
		t.Fatalf("app languages = %v, want %v", got, want)
	}
	source := readCatalog(t, "en")
	for _, tag := range want {
		catalog := readCatalog(t, tag)
		if catalog["@@locale"] != tag {
			t.Errorf("%s: incorrect locale metadata", tag)
		}
		messages := reflect.TypeFor[Messages]()
		count := 0
		for key, message := range source {
			if strings.HasPrefix(key, "@") {
				continue
			}
			count++
			translation, ok := catalog[key].(string)
			if !ok || strings.TrimSpace(translation) == "" {
				t.Errorf("%s: missing %s", tag, key)
			}
			if _, err := i18n.ParseMessage(translation); err != nil {
				t.Errorf("%s/%s: %v", tag, key, err)
			}
			if _, ok := messages.MethodByName(key); !ok {
				t.Errorf("%s: no generated method; run mygo generate", key)
			}
			if _, ok := message.(string); !ok {
				t.Errorf("%s: source is not a message", key)
			}
		}
		if count != messages.NumMethod() {
			t.Errorf("%d source messages, %d generated methods; run mygo generate", count, messages.NumMethod())
		}
		for key := range catalog {
			if !strings.HasPrefix(key, "@") {
				if _, ok := source[key]; !ok {
					t.Errorf("%s: unknown message %s", tag, key)
				}
			}
		}
	}
}

func readCatalog(t *testing.T, tag string) map[string]any {
	t.Helper()
	data, err := files.ReadFile(tag + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog map[string]any
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestCatalogFallbackAndTypedPlaceholders(t *testing.T) {
	for _, tag := range []string{"", "fr", "zh-Hant", "zh-TW"} {
		if got := In(tag).Overview(); got != "Overview" {
			t.Errorf("In(%q).Overview() = %q", tag, got)
		}
	}
	for _, tag := range []string{"zh", "zh-CN", "zh-Hans-HK"} {
		if got := In(tag).Overview(); got != "概览" {
			t.Errorf("In(%q).Overview() = %q", tag, got)
		}
	}
	for _, tc := range []struct {
		tag, chart, failed, period string
	}{
		{"en", "BTC/USDT candlestick chart", "Could not load OKX · Spot · BTC/USDT", "EMA 1234 period"},
		{"zh-Hans", "BTC/USDT K 线图", "无法加载 OKX · Spot · BTC/USDT", "EMA 1234 周期"},
	} {
		messages := In(tc.tag)
		if got := messages.CandleChart("BTC/USDT"); got != tc.chart {
			t.Errorf("chart = %q, want %q", got, tc.chart)
		}
		if got := messages.InstrumentLoadFailed("OKX · Spot", "BTC/USDT"); got != tc.failed {
			t.Errorf("load error = %q, want %q", got, tc.failed)
		}
		if got := messages.EMAPeriod(1234); got != tc.period {
			t.Errorf("EMA period = %q, want %q", got, tc.period)
		}
	}
}

func TestPseudoLocalesKeepPlaceholderValues(t *testing.T) {
	const symbol = "BTC/USDT"
	for _, tag := range []string{"en-XA", "ar-XB"} {
		got := In(tag).ChooseInstrumentSymbol(symbol)
		if got == In("en").ChooseInstrumentSymbol(symbol) || !strings.Contains(got, symbol) {
			t.Errorf("%s: pseudo message did not transform text and preserve the symbol: %q", tag, got)
		}
	}
}
