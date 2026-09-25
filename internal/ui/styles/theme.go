package styles

import "github.com/charmbracelet/lipgloss"

var (
	SubtleColor    = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}
	PrimaryColor   = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7D56F4"}
	AccentColor    = lipgloss.AdaptiveColor{Light: "#02BA83", Dark: "#02BF87"}
	WarningColor   = lipgloss.AdaptiveColor{Light: "#FFB000", Dark: "#FFA500"}
	ActiveBorder   = lipgloss.Color("#7D56F4")
	InactiveBorder = lipgloss.Color("#3C3C3C")

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(PrimaryColor).
		Padding(0, 1)

	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(InactiveBorder)

	ActivePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ActiveBorder)

	Method = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Background(lipgloss.Color("#2A2A38")).
		Foreground(lipgloss.Color("#00D7D7"))

	ActiveMethod = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			Background(PrimaryColor).
			Foreground(lipgloss.Color("#FFFFFF"))

	SendBtn = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		Background(lipgloss.Color("#2C7A4D")).
		Foreground(lipgloss.Color("#FFFFFF"))

	ActiveSendBtn = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 2).
			Background(lipgloss.Color("#00E676")).
			Foreground(lipgloss.Color("#000000"))

	ActiveTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#44475A")).
			Padding(0, 1)

	InactiveTab = lipgloss.NewStyle().
			Foreground(SubtleColor).
			Padding(0, 1)

	Label = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0A0B0")).
		Bold(true)

	Status2xx = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#2E7D32")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	Status3xx = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#F57F17")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	StatusErr = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#C62828")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	MetaBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ECEFF4")).
			Background(lipgloss.Color("#3B4252")).
			Padding(0, 1)

	Help = lipgloss.NewStyle().
		Foreground(SubtleColor).
		MarginTop(1)

	Folder = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFA500"))

	ActiveRow = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#44475A")).
			Foreground(lipgloss.Color("#FFFFFF"))

	ModalBox = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(PrimaryColor).
			Background(lipgloss.Color("#1E1E2E")).
			Padding(1, 2)
)
