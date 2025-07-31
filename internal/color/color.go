package color

import "github.com/charmbracelet/lipgloss"

var (
	Blue   = lipgloss.Color("12") // Bright blue
	Cyan   = lipgloss.Color("14") // Bright cyan
	Yellow = lipgloss.Color("11") // Bright yellow
	Orange = lipgloss.Color("3")  // Yellow/Orange
	Green  = lipgloss.Color("10") // Bright green
	Red    = lipgloss.Color("9")  // Bright red
	White  = lipgloss.Color("15") // Bright white
	Gray   = lipgloss.Color("8")  // Gray
	Black  = lipgloss.Color("0")  // Black

	// Additional colors used by stdout reporter
	DarkBlue  = lipgloss.Color("4")   // Dark blue
	DarkGreen = lipgloss.Color("2")   // Dark green
	DarkRed   = lipgloss.Color("1")   // Dark red
	LightGray = lipgloss.Color("252") // Light gray
	DarkGray  = lipgloss.Color("240") // Dark gray
)
