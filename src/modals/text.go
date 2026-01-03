package modals

import (
	"fmt"
	"strings"
	"terminaccounting/meta"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type textModal struct {
	viewport viewport.Model

	text []string

	ready bool
}

func NewTextModal(text []string) *textModal {
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
		maxMessageWidth := 0
		for _, m := range tm.text {
			maxMessageWidth = max(maxMessageWidth, ansi.StringWidth(m))
		}

		tm.viewport.Width = min(maxMessageWidth, message.Width)
		tm.viewport.Height = min(len(tm.text), message.Height)

		// Upon receiving initial WindowSizeMsg, scroll to bottom
		if !tm.ready {
			tm.viewport.GotoBottom()
			tm.ready = true
		}

		// Refresh viewport for when Height increased
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

func (tm *textModal) View() string {
	return tm.viewport.View()
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
	return NewTextModal(tm.text)
}
