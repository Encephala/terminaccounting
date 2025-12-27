package modals

import (
	"fmt"
	"log/slog"
	"strings"
	"terminaccounting/meta"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type textModal struct {
	viewport viewport.Model

	message string
}

func NewTextModal(message string) *textModal {
	maxMessageWidth := -1
	for _, m := range strings.Split(message, "\n") {
		maxMessageWidth = max(maxMessageWidth, len(m))
	}

	numLines := min(len(strings.Split(message, "\n")), 20)

	vp := viewport.New(maxMessageWidth, numLines)
	vp.SetContent(message)
	vp.GotoBottom() // To show recentmost messages. Perhaps this is a bad default, but good enough for now.

	return &textModal{
		viewport: vp,
		message:  message,
	}
}

func (tm *textModal) Init() tea.Cmd {
	return nil
}

func (tm *textModal) Update(message tea.Msg) (view.View, tea.Cmd) {
	switch message := message.(type) {
	case meta.NavigateMsg:
		slog.Debug("navigating out the wazoo", "msg", message)

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

func (tm *textModal) View() string {
	return tm.viewport.View()
}

func (tm *textModal) AcceptedModels() map[meta.ModelType]struct{} {
	return make(map[meta.ModelType]struct{})
}

func (tm *textModal) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	return meta.MotionSet{Normal: normalMotions}
}

func (tm *textModal) CommandSet() meta.CommandSet {
	return meta.CommandSet{}
}

func (tm *textModal) Reload() view.View {
	return NewTextModal(tm.message)
}
