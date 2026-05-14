package modals

import (
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

type Modal interface {
	Init() tea.Cmd
	Update(tea.Msg) (Modal, tea.Cmd)
	View() string

	AllowsInsertMode() bool
	AllowsSearchMode() bool

	MotionSet() meta.Trie[tea.Msg]
	CommandSet() meta.Trie[tea.Msg]

	Reload() Modal
}

type ModalManager struct {
	DB *sqlx.DB

	width, height int

	Modal Modal
}

func NewModalManager(DB *sqlx.DB) *ModalManager {
	return &ModalManager{DB: DB}
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
			mm.Modal, cmd = mm.Modal.Update(tea.WindowSizeMsg{
				// -8 for the padding
				Width:  message.Width - 8,
				Height: message.Height,
			})
		}

		return mm, cmd

	case meta.ShowTextModalMsg:
		mm.Modal = newTextModal(message.Text...)

		var cmd tea.Cmd
		mm.Modal, cmd = mm.Modal.Update(tea.WindowSizeMsg{
			Width:  mm.width - 8,
			Height: mm.height,
		})

		return mm, tea.Batch(mm.Modal.Init(), cmd)

	case meta.ShowBankImporterMsg:
		mm.Modal = newBankImporter()

		var cmd tea.Cmd
		mm.Modal, cmd = mm.Modal.Update(tea.WindowSizeMsg{
			Width:  mm.width - 8,
			Height: mm.height,
		})

		return mm, tea.Batch(mm.Modal.Init(), cmd)

	case meta.ShowNotificationsMsg:
		mm.Modal = newNotificationsModal()

		var cmd tea.Cmd
		mm.Modal, cmd = mm.Modal.Update(tea.WindowSizeMsg{
			Width:  mm.width - 8,
			Height: mm.height,
		})

		return mm, tea.Batch(mm.Modal.Init(), cmd)

	case meta.ShowGlobalSearchMsg:
		mm.Modal = newGlobalSearchModal(mm.DB)

		var cmd tea.Cmd
		mm.Modal, cmd = mm.Modal.Update(tea.WindowSizeMsg{
			Width:  mm.width - 8,
			Height: mm.height,
		})

		return mm, tea.Batch(mm.Modal.Init(), cmd)

	case meta.ReloadViewMsg:
		mm.Modal = mm.Modal.Reload()

		var windowSizeCmd tea.Cmd
		mm.Modal, windowSizeCmd = mm.Modal.Update(tea.WindowSizeMsg{
			Width:  mm.width - 8,
			Height: mm.height,
		})

		notificationCmd := meta.MessageCmd(meta.NotificationMessageMsg{Message: "Refreshed modal"})

		// There's supposedly no guarantee on the order of cmds
		// But in practice putting notificationCmd after Modal.Init makes the refreshed notificationsModal
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

func (mm *ModalManager) CurrentViewAllowsSearchMode() bool {
	return mm.Modal.AllowsSearchMode()
}

func (mm *ModalManager) CurrentMotionSet() meta.Trie[tea.Msg] {
	return mm.Modal.MotionSet()
}

func (mm *ModalManager) CurrentCommandSet() meta.Trie[tea.Msg] {
	return mm.Modal.CommandSet()
}
