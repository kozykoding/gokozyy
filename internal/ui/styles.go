package ui

import "github.com/charmbracelet/lipgloss"

const Banner = `
 ██████╗  ██████╗ ██╗  ██╗ ██████╗ ███████╗██╗   ██╗██╗   ██╗
██╔════╝ ██╔═══██╗██║ ██╔╝██╔═══██╗╚══███╔╝╚██╗ ██╔╝╚██╗ ██╔╝
██║  ███╗██║   ██║█████╔╝ ██║   ██║  ███╔╝  ╚████╔╝  ╚████╔╝ 
██║   ██║██║   ██║██╔═██╗ ██║   ██║ ███╔╝    ╚██╔╝    ╚██╔╝  
╚██████╔╝╚██████╔╝██║  ██╗╚██████╔╝███████╗   ██║      ██║   
 ╚═════╝  ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝   ╚═╝      ╚═╝   
`

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ffd7")).
			MarginBottom(1)

	QuestionStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#00ffd7")).
			Foreground(lipgloss.Color("#000000")).
			Padding(0, 1).
			MarginBottom(1)

	DescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#b19cd9")) // purple-ish
	OptionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff"))
	InactiveOptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	CursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ffd7"))

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(1)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00ffd7")).
			Padding(1, 2).
			Margin(1, 2)
)
