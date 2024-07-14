package ledgers

import (
	"fmt"
	"strings"
	"terminaccounting/meta"
	"terminaccounting/styles"

	"local/bubbles/itempicker"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (l Ledger) FilterValue() string {
	var result strings.Builder
	result.WriteString(l.Name)
	result.WriteString(strings.Join(l.Notes, ";"))
	return result.String()
}

func (l Ledger) Title() string {
	return l.Name
}

func (l Ledger) Description() string {
	return l.Name
}

type CreateView struct {
	idInput   textinput.Model
	nameInput textinput.Model
	typeInput itempicker.Model
	noteInput textarea.Model

	styles styles.CreateViewStyles
}

func NewCreateView(app meta.App, colours styles.AppColours) *CreateView {
	styles := styles.CreateViewStyles{
		Title: lipgloss.NewStyle().Background(colours.Background).Padding(0, 1),
	}

	return &CreateView{
		idInput:   textinput.New(),
		nameInput: textinput.New(),
		typeInput: itempicker.New(),
		noteInput: textarea.New(),

		styles: styles,
	}
}

func (cv *CreateView) Init() tea.Cmd {
	return nil
}

func (cv *CreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return cv, nil
}

func (cv *CreateView) View() string {
	var result strings.Builder

	result.WriteString("\n")
	result.WriteString(fmt.Sprintf(" %s", cv.styles.Title.Render("Create new Ledgers")))
	result.WriteString("\n\n")

	result.WriteString(cv.idInput.View())
	result.WriteString(" | ")
	result.WriteString(cv.nameInput.View())
	result.WriteString(" | ")
	result.WriteString(cv.typeInput.View())
	result.WriteString(" | ")
	result.WriteString(cv.noteInput.View())

	return result.String()
}

func (cv *CreateView) Type() meta.ViewType {
	return meta.CreateViewType
}
