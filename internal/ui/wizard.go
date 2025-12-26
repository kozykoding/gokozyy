package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Steps
const (
	stepName = iota
	stepFramework
	stepDB
	stepDocker // NEW
	stepFrontend
	stepSummary
	stepDone
)

// Result is what the wizard will return.
type Result struct {
	ProjectName string
	Framework   string // backend: std|chi|gin
	DBDriver    string // none|postgres|sqlite
	Frontend    string // "vite-react-tailwind" or "vite-react-tailwind-shadcn"
	Runtime     string // set to "bun"
	UseDocker   bool
	Confirmed   bool
}

type WizardModel struct {
	step          int
	nameInput     textinput.Model
	frameworkList RadioListModel
	dbList        RadioListModel
	frontendList  RadioListModel
	result        Result
	quit          bool
}

func NewWizardModel() WizardModel {
	// name input
	ti := textinput.New()
	ti.Placeholder = "my-project"
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 30

	frontendOpts := []RadioOption{
		{
			Label: "Vite + React + Tailwind + Bun",
			Description: "Basic Vite React app styled with Tailwind CSS, " +
				"using Bun as the runtime",
			Value: "vite-react-tailwind",
		},
		{
			Label: "Vite + React + Tailwind + shadcn/ui + Bun",
			Description: "Vite React app with Tailwind and shadcn/ui components " +
				"(with Bun runtime)",
			Value: "vite-react-tailwind-shadcn",
		},
	}

	frameworkOpts := []RadioOption{
		{
			Label:       "Standard-library",
			Description: "The built-in Go standard library HTTP package",
			Value:       "std",
		},
		{
			Label:       "Chi",
			Description: "Lightweight, idiomatic router for Go HTTP services",
			Value:       "chi",
		},
		{
			Label:       "Gin",
			Description: "Martini-like API, high performance HTTP framework",
			Value:       "gin",
		},
	}

	dbOpts := []RadioOption{
		{
			Label:       "None",
			Description: "No DB driver will be installed",
			Value:       "none",
		},
		{
			Label:       "Postgres",
			Description: "pgx postgres driver for Go",
			Value:       "postgres",
		},
		{
			Label:       "Sqlite",
			Description: "sqlite3 driver for Go's database/sql interface",
			Value:       "sqlite",
		},
	}

	return WizardModel{
		step:      stepName,
		nameInput: ti,
		frameworkList: NewRadioList(
			"What framework do you want to use in your Go project?",
			"Press y to confirm choice.",
			frameworkOpts,
		),
		dbList: NewRadioList(
			"What database driver do you want to use in your Go project?",
			"Press y to confirm choice.",
			dbOpts,
		),
		frontendList: NewRadioList(
			"What frontend stack do you want?",
			"Press y to confirm choice.",
			frontendOpts,
		),
	}
}

func (m WizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// global keys
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.quit = true
			return m, tea.Quit
		}

		switch m.step {
		case stepName:
			return m.updateName(msg)
		case stepFramework:
			return m.updateFramework(msg)
		case stepDB:
			return m.updateDB(msg)
		case stepDocker:
			return m.updateDocker(msg)
		case stepFrontend:
			return m.updateFrontend(msg)
		case stepSummary:
			return m.updateSummary(msg)
		case stepDone:
			return m, tea.Quit
		}
	}

	// Let components update (e.g. blinking cursor)
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m WizardModel) updateName(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if strings.TrimSpace(m.nameInput.Value()) == "" {
			// keep focus but maybe set default
			m.nameInput.SetValue("my-project")
		}
		m.step = stepFramework
		return m, nil
	}
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m WizardModel) updateFramework(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if v, ok := m.frameworkList.SelectedValue(); ok {
			m.result.Framework = v
			m.step = stepDB
			return m, nil
		}
		// if nothing selected, ignore
	}
	var cmd tea.Cmd
	m.frameworkList, cmd = m.frameworkList.Update(msg)
	return m, cmd
}

func (m WizardModel) updateFrontend(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if v, ok := m.frontendList.SelectedValue(); ok {
			m.result.Frontend = v
			m.result.Runtime = "bun"
			m.step = stepSummary
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.frontendList, cmd = m.frontendList.Update(msg)
	return m, cmd
}

func (m WizardModel) updateDB(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if v, ok := m.dbList.SelectedValue(); ok {
			m.result.DBDriver = v
			m.step = stepDocker
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.dbList, cmd = m.dbList.Update(msg)
	return m, cmd
}

func (m WizardModel) updateDocker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.result.UseDocker = true
		m.step = stepFrontend
		return m, nil
	case "n":
		m.result.UseDocker = false
		m.step = stepFrontend
		return m, nil
	case "h", "left":
		m.step = stepDB
		return m, nil
	}
	return m, nil
}

func (m WizardModel) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "y":
		m.result.ProjectName = strings.TrimSpace(m.nameInput.Value())
		if m.result.ProjectName == "" {
			m.result.ProjectName = "my-project"
		}
		// runtime is already set when frontend was chosen
		m.result.Confirmed = true
		m.step = stepDone
		return m, tea.Quit
	case "h", "left":
		m.step = stepFrontend
	}
	return m, nil
}

func (m WizardModel) View() string {
	switch m.step {
	case stepName:
		return m.viewName()
	case stepFramework:
		return m.frameworkList.View()
	case stepDB:
		return m.dbList.View()
	case stepDocker:
		return m.viewDocker()
	case stepFrontend:
		return m.frontendList.View()
	case stepSummary:
		return m.viewSummary()
	case stepDone:
		return ""
	default:
		return ""
	}
}

func (m WizardModel) viewDocker() string {
	body := fmt.Sprintf(
		"%s\n%s\n\n%s\n\n%s",
		TitleStyle.Render("Docker support"),
		QuestionStyle.Render("Do you want Docker/docker-compose files for your backend/DB?"),
		"Press y for Yes, n for No.",
		HelpStyle.Render("y = yes • n = no • h = back • q = quit"),
	)
	return BoxStyle.Render(body)
}

func (m WizardModel) viewName() string {
	body := fmt.Sprintf(
		"%s\n%s\n\n%s\n\n%s",
		TitleStyle.Render(Banner),
		QuestionStyle.Render("What is the name of your project?"),
		m.nameInput.View(),
		HelpStyle.Render("Type a name and press Enter • q to quit"),
	)
	return BoxStyle.Render(body)
}

func (m WizardModel) viewSummary() string {
	name := m.nameInput.Value()
	if name == "" {
		name = "my-project"
	}
	fw, _ := m.frameworkList.SelectedValue()
	db, _ := m.dbList.SelectedValue()
	fe, _ := m.frontendList.SelectedValue()

	var b strings.Builder
	b.WriteString(TitleStyle.Render("Summary"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Project:    %s\n", OptionStyle.Render(name)))
	b.WriteString(fmt.Sprintf("Backend:    %s\n", OptionStyle.Render(fw)))
	b.WriteString(fmt.Sprintf("Database:   %s\n", OptionStyle.Render(db)))
	b.WriteString(fmt.Sprintf("Frontend:   %s\n", OptionStyle.Render(fe)))
	b.WriteString(fmt.Sprintf("Runtime:    %s\n", OptionStyle.Render("bun")))
	docker := "no"
	if m.result.UseDocker {
		docker = "yes"
	}
	b.WriteString(fmt.Sprintf("Docker:     %s\n\n", OptionStyle.Render(docker)))

	return BoxStyle.Render(b.String())
}

// Result exposes the collected configuration.
func (m WizardModel) Result() Result {
	return m.result
}
