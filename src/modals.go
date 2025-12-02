package main

import (
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

func newOverlay(main *terminaccounting) *overlay.Model {
	return overlay.New(
		main.modal,
		main.appManager,
		overlay.Center,
		overlay.Center,
		0,
		1,
	)
}

type textModal struct {
	message string
}

func (mm textModal) Init() tea.Cmd {
	return nil
}

func (mm textModal) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return mm, nil
}

func (mm textModal) View() string {
	return meta.ModalStyle.Render(mm.message)
}

type notificationMsg struct {
	text    string
	isError bool
}

func (nm notificationMsg) String() string {
	if nm.isError {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(nm.text)
	}

	return nm.text
}
