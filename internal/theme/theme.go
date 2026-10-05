package theme

import (
	"fmt"
	"github.com/reathloo/shellux/internal/themepack"

	"github.com/charmbracelet/x/ansi"
)

// Theme contains semantic ANSI colors used by the renderer.
type Theme struct {
	Secondary string
	Accent    string
	Muted     string
	Reset     string
	Rainbow   bool

	// The UI uses these same colors without parsing the renderer's ANSI strings.
	SecondaryColor, AccentColor, MutedColor ansi.Color
}

// newTheme derives terminal sequences and UI colors from one shared palette.
func newTheme(secondary, accent, muted ansi.Color, rainbow bool) Theme {
	return Theme{
		Secondary: ansi.NewStyle().ForegroundColor(secondary).String(),
		Accent:    ansi.NewStyle().ForegroundColor(accent).String(),
		Muted:     ansi.NewStyle().ForegroundColor(muted).String(),
		Reset:     "\033[0m", Rainbow: rainbow,
		SecondaryColor: secondary, AccentColor: accent, MutedColor: muted,
	}
}

func DarkLuxury() Theme {
	return newTheme(ansi.BasicColor(8), ansi.IndexedColor(180), ansi.BasicColor(8), false)
}

// Neon renders the dashboard with cyan and magenta terminal colors.
func Neon() Theme {
	return newTheme(ansi.IndexedColor(51), ansi.IndexedColor(201), ansi.IndexedColor(45), false)
}

// Aurora renders the dashboard with calm turquoise and violet accents.
func Aurora() Theme {
	return newTheme(ansi.IndexedColor(141), ansi.IndexedColor(87), ansi.IndexedColor(103), false)
}

// Rainbow renders the dashboard with a vivid multi-color palette.
func Rainbow() Theme {
	return newTheme(ansi.IndexedColor(226), ansi.IndexedColor(201), ansi.IndexedColor(51), true)
}

// ShelluxDefault matches the gold wordmark on a black terminal background.
func ShelluxDefault() Theme {
	return newTheme(ansi.RGBColor{R: 217, G: 164, B: 65}, ansi.RGBColor{R: 255, G: 240, B: 166}, ansi.RGBColor{R: 159, G: 132, B: 82}, false)
}

// RainbowColors is shared by the header and the settings headings.
func RainbowColors() []ansi.Color {
	return []ansi.Color{ansi.IndexedColor(196), ansi.IndexedColor(214), ansi.IndexedColor(226), ansi.IndexedColor(46), ansi.IndexedColor(51), ansi.IndexedColor(27), ansi.IndexedColor(201)}
}

// ByName returns a supported visual style.
func ByName(name string) (Theme, error) {
	switch name {
	case "dark-luxury", "luxury":
		return DarkLuxury(), nil
	case "neon":
		return Neon(), nil
	case "aurora":
		return Aurora(), nil
	case "rainbow":
		return Rainbow(), nil
	case "shelluxdefault":
		return ShelluxDefault(), nil
	default:
		if e, ok := themepack.Lookup(name); ok {
			return newTheme(rgb(e.Style.Secondary), rgb(e.Style.Accent), rgb(e.Style.Muted), false), nil
		}
		return Theme{}, fmt.Errorf("unsupported style %q", name)
	}
}

func rgb(value string) ansi.Color {
	var r, g, b uint8
	fmt.Sscanf(value, "#%02x%02x%02x", &r, &g, &b)
	return ansi.RGBColor{R: r, G: g, B: b}
}
