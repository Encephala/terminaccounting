package modals

import (
	"terminaccounting/meta"
	"terminaccounting/view"

	tea "github.com/charmbracelet/bubbletea"
)

type ModalManager struct {
	width, height int

	Modal view.View
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
		mm.Modal = NewTextModal(message.Text)

		initCmd := mm.Modal.Init()
		windowSizeCmd := meta.MessageCmd(tea.WindowSizeMsg{
			Width:  mm.width,
			Height: mm.height,
		})

		return mm, tea.Batch(initCmd, windowSizeCmd)

	case meta.ShowBankImporterMsg:
		mm.Modal = NewBankStatementImporter()

		initCmd := mm.Modal.Init()
		windowSizeCmd := meta.MessageCmd(tea.WindowSizeMsg{
			Width:  mm.width,
			Height: mm.height,
		})

		return mm, tea.Batch(initCmd, windowSizeCmd)

	case meta.ReloadViewMsg:
		mm.Modal = mm.Modal.Reload()

		return mm, mm.Modal.Init()
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

	return meta.ModalStyle.Render(mm.Modal.View())
}

func (mm *ModalManager) CurrentMotionSet() meta.MotionSet {
	return mm.Modal.MotionSet()
}

func (mm *ModalManager) CurrentCommandSet() meta.CommandSet {
	return mm.Modal.CommandSet()
}
