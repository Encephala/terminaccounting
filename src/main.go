package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	log *log.Logger

	viewWidth, viewHeight int

	currentView int

	rows    []Row
	inputs  [2]textinput.Model
	focused int

	inCommandMode bool
	commandInput  textinput.Model
}

type Row struct {
	id          int
	description string
	value       MonetaryValue
}

type MonetaryValue struct {
	whole      int
	fractional int
}

func main() {
	m := model{}

	logFile, err := os.OpenFile("debug.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		log.Fatalf("Couldn't open log file %q", "debug.log")
	}
	defer logFile.Close()
	m.log = log.New(logFile, "", log.Ldate|log.Ltime)

	m.log.Println("Program started")

	for i := range m.inputs {
		m.inputs[i] = textinput.New()
	}
	m.inputs[0].Focus()

	m.commandInput = textinput.New()

	program := tea.NewProgram(m)

	_, err = program.Run()
	if err != nil {
		return
	}
}

func (m model) Init() tea.Cmd {
	// No startup I/O for now
	return textinput.Blink
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	commands := []tea.Cmd{}
	var command tea.Cmd

	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = message.Width
		m.viewHeight = message.Height

	case tea.KeyMsg:
		switch message.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.log.Println("Program exited")
			return m, tea.Quit

		case tea.KeyEnter:
			valueParts := strings.Split(m.inputs[1].Value(), ".")

			whole, err := strconv.Atoi(valueParts[0])
			if err != nil {
				m.log.Printf("Invalid integer %s\n", m.inputs[1].Value())
				return m, nil
			}

			fractional, err := strconv.Atoi(valueParts[1])
			if err != nil {
				m.log.Printf("Invalid integer %s\n", m.inputs[1].Value())
				return m, nil
			}

			value := MonetaryValue{whole, fractional}

			row := Row{
				id:          len(m.rows),
				description: m.inputs[0].Value(),
				value:       value,
			}

			m.rows = append(m.rows, row)

		case tea.KeyTab, tea.KeyShiftTab:
			m.focused = 1 - m.focused

			commands = append(commands, m.inputs[m.focused].Focus())

			// Blur the other one
			m.inputs[1-m.focused].Blur()

		case tea.KeyPgDown:
			m.currentView = 1

		case tea.KeyPgUp:
			m.currentView = 0

		case tea.KeyRunes:
			if message.String() == ":" {
				m.inCommandMode = true

				for i := range m.inputs {
					m.inputs[i].Blur()
				}

				return m, m.commandInput.Focus()
			}
		}
	}

	for i, input := range m.inputs {
		m.inputs[i], command = input.Update(message)

		commands = append(commands, command)
	}

	m.commandInput, command = m.commandInput.Update(message)
	commands = append(commands, command)

	return m, tea.Batch(commands...)
}

func (m model) View() string {
	views := []string{}

	if m.currentView == 0 {
		views = append(views, m.inputView())
	}

	if m.inCommandMode {
		views = append(views, m.commandView())
	}

	views = append(views, strconv.Itoa(m.currentView))

	return lipgloss.JoinHorizontal(lipgloss.Bottom, views...)
}

func (m model) inputView() string {
	var builder strings.Builder

	for i := range m.inputs {
		builder.WriteString(m.inputs[i].View())

		builder.WriteString("\t")
	}

	builder.WriteString("\n\n")

	builder.WriteString("Rows:\n")

	for _, row := range m.rows {
		builder.WriteString(strconv.Itoa(row.id) + "\t")

		builder.WriteString(row.description + "\t")

		builder.WriteString("€ " + strconv.Itoa(row.value.whole) + "." + strconv.Itoa(row.value.fractional) + "\t")

		builder.WriteString("\n")
	}

	builder.WriteString("\n\n")

	return builder.String()
}

func (m model) commandView() string {
	var builder strings.Builder

	builder.WriteString("Command: ")

	builder.WriteString(m.commandInput.View())

	builder.WriteString("\n\n")

	return lipgloss.PlaceVertical(m.viewHeight, lipgloss.Bottom, builder.String())
}
