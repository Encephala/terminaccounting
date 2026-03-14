package modals

import (
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
)

type Modal interface {
	Init() tea.Cmd
	Update(tea.Msg) (Modal, tea.Cmd)
	View() string

	AllowsInsertMode() bool

	MotionSet() meta.Trie[tea.Msg]
	CommandSet() meta.Trie[tea.Msg]

	Reload() Modal
}

type ModalManager struct {
	width, height int

	Modal Modal
}

func NewModalManager() *ModalManager {
	return &ModalManager{}
}

func (mm *ModalManager) Init() tea.Cmd {
	return nil
}

func (mm *ModalManager) Update(message tea.Msg) (*ModalManager, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		mm.width = message.Width
		mm.height = message.Height

		var cmd tea.Cmd
		if mm.Modal != nil {
			mm.Modal, cmd = mm.Modal.Update(message)
		}

		return mm, cmd

	case meta.ShowTextModalMsg:
		mm.Modal = NewTextModal(message.Text...)

		var cmd tea.Cmd
		mm.Modal, cmd = mm.Modal.Update(tea.WindowSizeMsg{
			Width:  mm.width,
			Height: mm.height,
		})

		return mm, tea.Batch(mm.Modal.Init(), cmd)

	case meta.ShowBankImporterMsg:
		mm.Modal = NewBankImporter()

		var cmd tea.Cmd
		mm.Modal, cmd = mm.Modal.Update(tea.WindowSizeMsg{
			Width:  mm.width,
			Height: mm.height,
		})

		return mm, tea.Batch(mm.Modal.Init(), cmd)

	case meta.ShowNotificationsMsg:
		mm.Modal = NewNotificationsModal()

		var cmd tea.Cmd
		mm.Modal, cmd = mm.Modal.Update(tea.WindowSizeMsg{
			Width:  mm.width,
			Height: mm.height,
		})

		return mm, tea.Batch(mm.Modal.Init(), cmd)

	case meta.ReloadViewMsg:
		mm.Modal = mm.Modal.Reload()

		var windowSizeCmd tea.Cmd
		mm.Modal, windowSizeCmd = mm.Modal.Update(tea.WindowSizeMsg{
			Width:  mm.width,
			Height: mm.height,
		})

		notificationCmd := meta.MessageCmd(meta.NotificationMessageMsg{Message: "Refreshed modal"})

		// There's supposedly no guarantee on the order of cmds
		// But in practice putting notificationCmd after Modal.Init makes the refreshed modal
		// always contain the "Refreshed modal" message
		return mm, tea.Batch(mm.Modal.Init(), notificationCmd, windowSizeCmd)
	}

	var cmd tea.Cmd
	mm.Modal, cmd = mm.Modal.Update(message)

	return mm, cmd
}

func (mm *ModalManager) View() string {
	view := mm.Modal.View()

	if view == "" {
		return ""
	}

	return meta.ModalStyle(mm.width, mm.height).Render(mm.Modal.View())
}

func (mm *ModalManager) CurrentViewAllowsInsertMode() bool {
	return mm.Modal.AllowsInsertMode()
}

func (mm *ModalManager) CurrentMotionSet() meta.Trie[tea.Msg] {
	return mm.Modal.MotionSet()
}

func (mm *ModalManager) CurrentCommandSet() meta.Trie[tea.Msg] {
	return mm.Modal.CommandSet()
}
