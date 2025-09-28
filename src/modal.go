package main

import (
	"log/slog"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

type modalManager struct {
	overlay   *overlay.Model
	showModal bool
	main      *terminaccounting
	modal     *modalModel
}

func newModalManager(main *terminaccounting) *modalManager {
	modal := modalModel{}

	return &modalManager{
		overlay: overlay.New(&modal,
			main,
			overlay.Center,
			overlay.Center,
			0,
			0),
		showModal: true,
		main:      main,
		modal:     &modal,
	}
}

func (mm *modalManager) Init() tea.Cmd {
	return mm.main.Init()
}

func (mm *modalManager) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.ShowModalMsg:
		mm.showModal = true
		mm.modal.message = message.Message

		return mm, nil

	case meta.CloseModalMsg:
		mm.showModal = false

		return mm, nil
	}

	new, cmd := mm.main.Update(message)
	mm.main = new.(*terminaccounting)

	return mm, cmd
}

func (mm *modalManager) View() string {
	if mm.showModal {
		return mm.overlay.View()
	}

	return mm.main.View()
}

type modalModel struct {
	message string
}

func (m *modalModel) Init() tea.Cmd {
	return nil
}

func (m *modalModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *modalModel) View() string {
	return meta.ModalStyle.Render(m.message)
}
