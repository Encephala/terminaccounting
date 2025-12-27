package view

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"terminaccounting/bubbles/itempicker"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type View interface {
	tea.Model

	AcceptedModels() map[meta.ModelType]struct{}

	MotionSet() meta.MotionSet
	CommandSet() meta.CommandSet

	Reload() View
}

// TODO is this needed still?
type viewable interface {
	View() string
}

func renderBoolean(reconciled bool) string {
	if reconciled {
		// Font Awesome checkbox because it's monospace, standard emoji is too wide
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("")
	} else {
		return "□"
	}
}

type input interface {
	Update(tea.Msg) (input, tea.Cmd)
	View() string

	Focus()
	Blur()

	Value() any
	SetValue(any) error
}

type inputAdapter[T viewable] struct {
	model T
	// Adapter for Update, since every model returns itself and not generic input
	// (or generic tea.Model or something alike)
	focusFn    func(*T)
	blurFn     func(*T)
	updateFn   func(T, tea.Msg) (T, tea.Cmd)
	valueFn    func(T) any
	setValueFn func(*T, any) error
}

func (ia inputAdapter[T]) Update(message tea.Msg) (input, tea.Cmd) {
	var cmd tea.Cmd
	ia.model, cmd = ia.updateFn(ia.model, message)

	return ia, cmd
}
func (ia inputAdapter[T]) View() string {
	return ia.model.View()
}
func (ia inputAdapter[T]) Focus() {
	ia.blurFn(&ia.model)
}
func (ia inputAdapter[T]) Blur() {
	ia.focusFn(&ia.model)
}
func (ia inputAdapter[T]) Value() any {
	return ia.valueFn(ia.model)
}
func (ia inputAdapter[T]) SetValue(value any) error {
	return ia.setValueFn(&ia.model, value)
}

func newAdapterFrom(input any) input {
	switch input := input.(type) {
	case textinput.Model:
		return inputAdapter[textinput.Model]{
			model: input,
			updateFn: func(model textinput.Model, message tea.Msg) (textinput.Model, tea.Cmd) {
				return model.Update(message)
			},
			valueFn: func(model textinput.Model) any {
				return model.Value()
			},
			setValueFn: func(model *textinput.Model, value any) error {
				model.SetValue(value.(string))
				return nil
			},
		}

	case textarea.Model:
		return inputAdapter[textarea.Model]{
			model: input,
			updateFn: func(model textarea.Model, message tea.Msg) (textarea.Model, tea.Cmd) {
				return model.Update(message)
			},
			valueFn: func(model textarea.Model) any {
				return model.Value()
			},
			setValueFn: func(model *textarea.Model, value any) error {
				model.SetValue(value.(string))
				return nil
			},
		}

	case itempicker.Model:
		return inputAdapter[itempicker.Model]{
			model: input,
			updateFn: func(model itempicker.Model, message tea.Msg) (itempicker.Model, tea.Cmd) {
				return model.Update(message)
			},
			valueFn: func(model itempicker.Model) any {
				return model.Value()
			},
			setValueFn: func(model *itempicker.Model, value any) error {
				model.SetValue(value.(itempicker.Item))
				return nil
			},
		}

	default:
		panic(fmt.Sprintf("haven't implemented adapter yet for input: %#v", input))
	}
}

type inputManager struct {
	width, height int

	activeInput int
	inputs      []input
	names       []string
}

func newInputManager(inputs []any, names []string) *inputManager {
	inputsAdapted := make([]input, len(inputs))

	for i, input := range inputs {
		inputsAdapted[i] = newAdapterFrom(input)
	}

	return &inputManager{
		inputs: inputsAdapted,
		names:  names,
	}
}

func (im *inputManager) Update(message tea.Msg) (*inputManager, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		im.width = message.Width
		im.height = message.Height

		return im, nil

	case meta.SwitchFocusMsg:
		// TODO: if on textinput, Blur() it
		switch message.Direction {
		case meta.PREVIOUS:
			im.previous()

		case meta.NEXT:
			im.next()
		}

		// TODO: If now on a textinput, Focus() it

		return im, nil

	case tea.KeyMsg:
		newInput, cmd := im.inputs[im.activeInput].Update(message)

		im.inputs[im.activeInput] = newInput

		return im, cmd

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (im *inputManager) View(highlightColour lipgloss.Color) string {
	var result strings.Builder

	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Left)
	highlightStyle := sectionStyle.Foreground(highlightColour)

	if len(im.names) != len(im.inputs) {
		panic("what in the fuck")
	}

	styles := slices.Repeat([]lipgloss.Style{sectionStyle}, len(im.names))
	styles[im.activeInput] = highlightStyle

	// +2 for padding
	maxNameColWidth := len(slices.MaxFunc(im.names, func(name string, other string) int {
		return cmp.Compare(len(name), len(other))
	})) + 2

	for i := range im.names {
		if im.names[i] == "" {
			result.WriteString(sectionStyle.Render(im.inputs[i].View()))
		} else {
			result.WriteString(lipgloss.JoinHorizontal(
				lipgloss.Top,
				sectionStyle.Width(maxNameColWidth).Render(im.names[i]),
				" ",
				styles[i].Render(im.inputs[i].View()),
			))
		}

		result.WriteString("\n")
	}

	return result.String()
}

func (im *inputManager) previous() {
	im.activeInput--

	if im.activeInput < 0 {
		im.activeInput += len(im.inputs)
	}
}

func (im *inputManager) next() {
	im.activeInput++

	im.activeInput %= len(im.inputs)
}

func (im *inputManager) getActiveInput() *input {
	return &im.inputs[im.activeInput]
}
