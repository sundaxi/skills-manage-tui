package styles

import "github.com/charmbracelet/lipgloss"

var Mocha = struct {
	Rosewater, Flamingo, Pink, Mauve, Red, Maroon, Peach string
	Yellow, Green, Teal, Sky, Sapphire, Blue, Lavender  string
	Text, Subtext1, Subtext0                             string
	Overlay2, Overlay1, Overlay0                         string
	Surface2, Surface1, Surface0                         string
	Base, Mantle, Crust                                  string
}{
	Rosewater: "#F5E0DC", Flamingo: "#F2CDCD", Pink: "#F5C2E7", Mauve: "#CBA6F7",
	Red: "#F38BA8", Maroon: "#EBA0AC", Peach: "#FAB387", Yellow: "#F9E2AF",
	Green: "#A6E3A1", Teal: "#94E2D5", Sky: "#89DCEB", Sapphire: "#74C7EC",
	Blue: "#89B4FA", Lavender: "#B4BEFE", Text: "#CDD6F4", Subtext1: "#BAC2DE",
	Subtext0: "#A6ADC8", Overlay2: "#9399B2", Overlay1: "#7F849C", Overlay0: "#6C7086",
	Surface2: "#585B70", Surface1: "#45475A", Surface0: "#313244", Base: "#1E1E2E",
	Mantle: "#181825", Crust: "#11111B",
}

var Latte = struct {
	Rosewater, Flamingo, Pink, Mauve, Red, Maroon, Peach string
	Yellow, Green, Teal, Sky, Sapphire, Blue, Lavender  string
	Text, Subtext1, Subtext0                             string
	Overlay2, Overlay1, Overlay0                         string
	Surface2, Surface1, Surface0                         string
	Base, Mantle, Crust                                  string
}{
	Rosewater: "#DC8A78", Flamingo: "#DD7878", Pink: "#EA76CB", Mauve: "#8839EF",
	Red: "#D20F39", Maroon: "#E64553", Peach: "#FE640B", Yellow: "#DF8E1D",
	Green: "#40A02B", Teal: "#179299", Sky: "#04A5E5", Sapphire: "#209FB5",
	Blue: "#1E66F5", Lavender: "#7287FD", Text: "#4C4F69", Subtext1: "#5C5F77",
	Subtext0: "#6C6F85", Overlay2: "#7C7F93", Overlay1: "#8C8FA1", Overlay0: "#9CA0B0",
	Surface2: "#ACB0BE", Surface1: "#BCC0CC", Surface0: "#CCD0DA", Base: "#EFF1F5",
	Mantle: "#E6E9EF", Crust: "#DCE0E8",
}

type Theme struct {
	Title, Subtitle, ActiveTab, InactiveTab, TabGap lipgloss.Style
	Selected, Cursor, Normal, Dimmed, Accent        lipgloss.Style
	Success, Warning, Error                         lipgloss.Style
	StatusBar, StatusText, StatusAccent             lipgloss.Style
	CheckboxOn, CheckboxOff                         string
	Border                                          lipgloss.Style
}

func NewTheme(name string) Theme {
	return NewThemeWithAccent(name, "")
}

var AccentColors = []string{
	"mauve", "pink", "red", "peach", "yellow",
	"green", "teal", "sky", "blue", "lavender",
}

func accentHex(palette string, accent string) string {
	colors := map[string]map[string]string{
		"mocha": {
			"rosewater": Mocha.Rosewater, "flamingo": Mocha.Flamingo, "pink": Mocha.Pink,
			"mauve": Mocha.Mauve, "red": Mocha.Red, "maroon": Mocha.Maroon,
			"peach": Mocha.Peach, "yellow": Mocha.Yellow, "green": Mocha.Green,
			"teal": Mocha.Teal, "sky": Mocha.Sky, "sapphire": Mocha.Sapphire,
			"blue": Mocha.Blue, "lavender": Mocha.Lavender,
		},
		"latte": {
			"rosewater": Latte.Rosewater, "flamingo": Latte.Flamingo, "pink": Latte.Pink,
			"mauve": Latte.Mauve, "red": Latte.Red, "maroon": Latte.Maroon,
			"peach": Latte.Peach, "yellow": Latte.Yellow, "green": Latte.Green,
			"teal": Latte.Teal, "sky": Latte.Sky, "sapphire": Latte.Sapphire,
			"blue": Latte.Blue, "lavender": Latte.Lavender,
		},
	}
	if p, ok := colors[palette]; ok {
		if hex, ok := p[accent]; ok {
			return hex
		}
	}
	return ""
}

func NewThemeWithAccent(name, accent string) Theme {
	var t Theme
	if name == "latte" {
		t = newLatteTheme()
	} else {
		t = newMochaTheme()
	}

	hex := accentHex(name, accent)
	if hex == "" {
		return t
	}

	c := lipgloss.Color(hex)
	t.Title = t.Title.Foreground(c)
	t.Subtitle = t.Subtitle.Foreground(c)
	t.ActiveTab = t.ActiveTab.Background(c)
	t.Cursor = t.Cursor.Foreground(c)
	t.Accent = t.Accent.Foreground(c)
	t.StatusAccent = t.StatusAccent.Foreground(c)
	return t
}

func newMochaTheme() Theme {
	return Theme{
		Title:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Mocha.Pink)).MarginBottom(1),
		Subtitle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Mocha.Mauve)),
		ActiveTab:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Mocha.Crust)).Background(lipgloss.Color(Mocha.Mauve)).Padding(0, 2),
		InactiveTab:  lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Subtext1)).Padding(0, 2),
		TabGap:       lipgloss.NewStyle().Background(lipgloss.Color(Mocha.Base)),
		Selected:     lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Green)).Bold(true),
		Cursor:       lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Mauve)).Bold(true),
		Normal:       lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Text)),
		Dimmed:       lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Subtext0)),
		Accent:       lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Mauve)),
		Success:      lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Green)),
		Warning:      lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Yellow)),
		Error:        lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Red)),
		StatusBar:    lipgloss.NewStyle().Background(lipgloss.Color(Mocha.Surface0)).Foreground(lipgloss.Color(Mocha.Text)).Padding(0, 1),
		StatusText:   lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Text)),
		StatusAccent: lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Mauve)).Bold(true),
		CheckboxOn:   lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Green)).Render("◉"),
		CheckboxOff:  lipgloss.NewStyle().Foreground(lipgloss.Color(Mocha.Overlay2)).Render("○"),
		Border:       lipgloss.NewStyle().BorderForeground(lipgloss.Color(Mocha.Surface1)),
	}
}

func newLatteTheme() Theme {
	return Theme{
		Title:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Latte.Mauve)).MarginBottom(1),
		Subtitle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Latte.Pink)),
		ActiveTab:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Latte.Crust)).Background(lipgloss.Color(Latte.Mauve)).Padding(0, 2),
		InactiveTab:  lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Overlay1)).Padding(0, 2),
		TabGap:       lipgloss.NewStyle().Background(lipgloss.Color(Latte.Base)),
		Selected:     lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Green)).Bold(true),
		Cursor:       lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Mauve)).Bold(true),
		Normal:       lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Text)),
		Dimmed:       lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Overlay0)),
		Accent:       lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Mauve)),
		Success:      lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Green)),
		Warning:      lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Yellow)),
		Error:        lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Red)),
		StatusBar:    lipgloss.NewStyle().Background(lipgloss.Color(Latte.Surface0)).Foreground(lipgloss.Color(Latte.Text)).Padding(0, 1),
		StatusText:   lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Subtext1)),
		StatusAccent: lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Mauve)).Bold(true),
		CheckboxOn:   lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Green)).Render("◉"),
		CheckboxOff:  lipgloss.NewStyle().Foreground(lipgloss.Color(Latte.Overlay0)).Render("○"),
		Border:       lipgloss.NewStyle().BorderForeground(lipgloss.Color(Latte.Surface1)),
	}
}
