package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// RadioOption holds one selectable item.
type RadioOption struct {
	Label       string
	Description string
	Value       string
}

// RadioListModel manages a list where exactly 0 or 1 item is selected.
type RadioListModel struct {
	Title    string
	Question string
	Options  []RadioOption
	cursor   int
	selected int // index, -1 if none selected
}

// Creates a new list with no selection.
func NewRadioList(title, question string, options []RadioOption) RadioListModel {
	return RadioListModel{
		Title:    title,
		Question: question,
		Options:  options,
		cursor:   0,
		selected: -1,
	}
}

func (m RadioListModel) Init() tea.Cmd { return nil }

func (m RadioListModel) Update(msg tea.Msg) (RadioListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.Options)-1 {
				m.cursor++
			}
		case " ":
			// space toggles: set selected to cursor
			if m.selected == m.cursor {
				m.selected = -1 // unselect
			} else {
				m.selected = m.cursor
			}
		}
	}
	return m, nil
}

func (m RadioListModel) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render(m.Title))
	b.WriteString("\n")
	b.WriteString(QuestionStyle.Render(m.Question))
	b.WriteString("\n\n")

	for i, opt := range m.Options {
		cursor := "  "
		if i == m.cursor {
			cursor = CursorStyle.Render("➜ ")
		}

		check := "[ ]"
		if i == m.selected {
			check = "[x]"
		}

		// headline
		line := OptionStyle.Render(check + " " + opt.Label)
		if i == m.cursor {
			line = OptionStyle.Bold(true).Render(check + " " + opt.Label)
		}

		b.WriteString(cursor + line + "\n")
		if opt.Description != "" {
			b.WriteString("    " + DescStyle.Render(opt.Description) + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(
		HelpStyle.Render(
			"↑/↓ move • space select • y confirm • q quit",
		),
	)

	return BoxStyle.Render(b.String())
}

func (m RadioListModel) SelectedValue() (string, bool) {
	if m.selected < 0 || m.selected >= len(m.Options) {
		return "", false
	}
	return m.Options[m.selected].Value, true
}
