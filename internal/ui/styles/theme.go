package styles

import "github.com/charmbracelet/lipgloss"

// Neutral zinc palette; the theme only tints small focus accents.
var (
	TextColor                           = lipgloss.AdaptiveColor{Light: "#27272A", Dark: "#D4D4D8"}
	BrightColor                         = lipgloss.AdaptiveColor{Light: "#09090B", Dark: "#FAFAFA"}
	SubtleColor                         = lipgloss.AdaptiveColor{Light: "#71717A", Dark: "#71717A"}
	FaintColor                          = lipgloss.AdaptiveColor{Light: "#A1A1AA", Dark: "#52525B"}
	SurfaceColor                        = lipgloss.AdaptiveColor{Light: "#E4E4E7", Dark: "#27272A"}
	BorderColor                         = lipgloss.AdaptiveColor{Light: "#E4E4E7", Dark: "#27272A"}
	FocusBorder                         = lipgloss.AdaptiveColor{Light: "#A1A1AA", Dark: "#52525B"}
	SuccessColor                        = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	WarningColor                        = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}
	ErrorColor                          = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
	AccentColor  lipgloss.TerminalColor = BrightColor

	// MethodColors are muted so the request name, not the verb, draws the eye.
	MethodColors = map[string]lipgloss.AdaptiveColor{
		"GET":    {Light: "#15803D", Dark: "#86EFAC"},
		"POST":   {Light: "#1D4ED8", Dark: "#93C5FD"},
		"PUT":    {Light: "#B45309", Dark: "#FCD34D"},
		"PATCH":  {Light: "#7E22CE", Dark: "#D8B4FE"},
		"DELETE": {Light: "#B91C1C", Dark: "#FCA5A5"},
		"HEAD":   {Light: "#52525B", Dark: "#A1A1AA"},
	}

	Title, Panel, ActivePanel, Method, ActiveMethod, SendBtn, ActiveSendBtn lipgloss.Style
	ActiveTab, FocusedTab, InactiveTab, Label, Muted, Faint, Pill, Help     lipgloss.Style
	Folder, Row, ActiveRow, ModalBox                                        lipgloss.Style
)

func init() { SetAccent(BrightColor) }

// SetAccent rebuilds the styles that carry the theme accent.
func SetAccent(accent lipgloss.TerminalColor) {
	AccentColor = accent
	Title = lipgloss.NewStyle().Bold(true).Foreground(BrightColor)
	Panel = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(BorderColor)
	ActivePanel = Panel.BorderForeground(FocusBorder)
	Method = lipgloss.NewStyle().Width(7).Foreground(FaintColor)
	ActiveMethod = lipgloss.NewStyle().Width(7).Bold(true)
	SendBtn = lipgloss.NewStyle().Padding(0, 1).Background(SurfaceColor).Foreground(TextColor)
	ActiveSendBtn = lipgloss.NewStyle().Padding(0, 1).Bold(true).Background(BrightColor).Foreground(lipgloss.AdaptiveColor{Light: "#FAFAFA", Dark: "#09090B"})
	InactiveTab = lipgloss.NewStyle().Padding(0, 1).Foreground(SubtleColor)
	ActiveTab = InactiveTab.Foreground(BrightColor).Background(SurfaceColor)
	FocusedTab = ActiveTab.Foreground(accent).Bold(true)
	Label = lipgloss.NewStyle().Foreground(SubtleColor)
	Muted = lipgloss.NewStyle().Foreground(SubtleColor)
	Faint = lipgloss.NewStyle().Foreground(FaintColor)
	Pill = lipgloss.NewStyle().Padding(0, 1).Background(SurfaceColor)
	Help = lipgloss.NewStyle().Foreground(FaintColor).MarginTop(1)
	Folder = lipgloss.NewStyle().Foreground(TextColor)
	Row = lipgloss.NewStyle().Foreground(TextColor)
	ActiveRow = lipgloss.NewStyle().Background(SurfaceColor).Foreground(accent)
	ModalBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(FocusBorder).Padding(1, 2)
}

// MethodStyle colors an HTTP verb with its muted method color.
func MethodStyle(method string) lipgloss.Style {
	c, ok := MethodColors[method]
	if !ok {
		c = MethodColors["HEAD"]
	}
	return lipgloss.NewStyle().Foreground(c)
}
