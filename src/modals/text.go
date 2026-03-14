package modals

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"terminaccounting/meta"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type textModal struct {
	width, height int

	text     []string
	viewport viewport.Model

	ready bool
}

func NewTextModal(text ...string) *textModal {
	viewport := viewport.New(0, 0)
	viewport.SetContent(strings.Join(text, "\n"))

	return &textModal{
		viewport: viewport,

		text: text,
	}
}

func (tm *textModal) Init() tea.Cmd {
	return nil
}

func (tm *textModal) Update(message tea.Msg) (view.View, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		tm.width = message.Width
		tm.height = message.Height

		tm.updateViewport()

		// Refresh viewport
		// This effectively scrolls up if a window height increase makes space for more rows,
		// otherwise it's a no-op
		tm.viewport.SetYOffset(tm.viewport.YOffset)

		return tm, nil

	case meta.NavigateMsg:
		switch message.Direction {
		case meta.DOWN:
			tm.viewport.ScrollDown(1)
		case meta.UP:
			tm.viewport.ScrollUp(1)
		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", message.Direction))
		}

		return tm, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (tm *textModal) updateViewport() {
	tm.viewport.SetContent(strings.Join(tm.text, "\n"))

	maxMessageWidth := 0
	for _, m := range tm.text {
		maxMessageWidth = max(maxMessageWidth, ansi.StringWidth(m))
	}

	tm.viewport.Width = min(maxMessageWidth, tm.width)
	tm.viewport.Height = min(len(tm.text), tm.height)
}

func (tm *textModal) View() string {
	return tm.viewport.View()
}

func (tm *textModal) Title() string {
	// TODO?
	return ""
}

func (tm *textModal) Type() meta.ViewType {
	return meta.TEXTMODALVIEWTYPE
}

func (tm *textModal) AllowsInsertMode() bool {
	return false
}

func (tm *textModal) AcceptedModels() map[meta.ModelType]struct{} {
	return make(map[meta.ModelType]struct{})
}

func (tm *textModal) MotionSet() meta.Trie[tea.Msg] {
	var motions meta.Trie[tea.Msg]

	motions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	motions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	return motions
}

func (tm *textModal) CommandSet() meta.Trie[tea.Msg] {
	return meta.Trie[tea.Msg]{}
}

func (tm *textModal) Reload() view.View {
	return NewTextModal(tm.text...)
}

type notificationsModal struct {
	*textModal
}

func NewNotificationsModal() *notificationsModal {
	return &notificationsModal{
		textModal: NewTextModal(),
	}
}

func (nm *notificationsModal) Init() tea.Cmd {
	return meta.MessageCmd(meta.FetchNotificationHistoryMsg{})
}

func (nm *notificationsModal) Update(message tea.Msg) (view.View, tea.Cmd) {
	switch message := message.(type) {
	case meta.NotificationHistoryLoadedMsg:
		slog.Debug("loaded notif history")

		if len(message.Notifications) == 0 {
			errorCmd := meta.MessageCmd(errors.New("no messages to show"))
			quitCmd := meta.MessageCmd(meta.QuitMsg{})

			return nm, tea.Batch(errorCmd, quitCmd)
		}

		var rendered []string
		for i, notification := range message.Notifications {
			newLine := fmt.Sprintf("%d: %s", i, notification.String())
			rendered = append(rendered, newLine)
		}

		nm.textModal.text = rendered
		nm.textModal.updateViewport()

		// Scroll to bottom to show most recent notifications
		nm.textModal.viewport.GotoBottom()

		return nm, nil
	}

	newTextModal, cmd := nm.textModal.Update(message)
	nm.textModal = newTextModal.(*textModal)

	return nm, cmd
}

func (nm *notificationsModal) View() string {
	slog.Debug("RENDERING OUTER", "width", nm.textModal.viewport.Width, "height", nm.textModal.viewport.Height)
	return nm.textModal.View()
}

func (nm *notificationsModal) Title() string {
	// TODO?
	return ""
}

func (nm *notificationsModal) Type() meta.ViewType {
	return meta.NOTIFICATIONSMODALVIEWTYPE
}

func (nm *notificationsModal) AllowsInsertMode() bool {
	return false
}

func (nm *notificationsModal) AcceptedModels() map[meta.ModelType]struct{} {
	return nm.textModal.AcceptedModels()
}

func (nm *notificationsModal) MotionSet() meta.Trie[tea.Msg] {
	return nm.textModal.MotionSet()
}

func (nm *notificationsModal) CommandSet() meta.Trie[tea.Msg] {
	return nm.textModal.CommandSet()
}

func (nm *notificationsModal) Reload() view.View {
	return NewNotificationsModal()
}
