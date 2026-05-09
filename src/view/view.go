package view

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"terminaccounting/bubbles/booleaninput"
	"terminaccounting/bubbles/itempicker"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type View interface {
	Init() tea.Cmd
	Update(tea.Msg) (View, tea.Cmd)
	View() string

	Type() meta.ViewType

	AllowsInsertMode() bool
	AcceptedModels() map[meta.ModelType]struct{}

	MotionSet() meta.Trie[tea.Msg]
	CommandSet() meta.Trie[tea.Msg]

	Reload() View
}

type viewable interface {
	View() string
}

func renderBoolean(reconciled bool) string {
	if reconciled {
		// Font Awesome checkbox because it's monospace, standard emoji character is too wide
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("")
	}

	return "□"
}

type metadata struct {
	names  []string
	values []string
}

func renderHeader(title string, metadata metadata, width int) string {
	if len(metadata.names) != len(metadata.values) {
		panic("que")
	}

	// Metadata
	var metadataBuilder strings.Builder
	for i := range metadata.names {
		metadataBuilder.WriteString(fmt.Sprintf("%s: %s", metadata.names[i], metadata.values[i]))

		if i != len(metadata.names)-1 {
			metadataBuilder.WriteString("\n")
		}
	}

	metadataRendered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Render(metadataBuilder.String())
	metadataWidth := ansi.StringWidth(strings.Split(metadataRendered, "\n")[0])

	// Title
	maxTitleWidth := max(width-metadataWidth, 30)

	titleRendered := lipgloss.NewStyle().
		Margin(1, 0).
		// TODO: width scaled by with space needed for metadata
		Render(ansi.Truncate(title, maxTitleWidth, "…"))
	titleWidth := ansi.StringWidth(strings.Split(titleRendered, "\n")[0])

	if len(metadata.names) == 0 {
		return titleRendered
	}

	// Middle empty space
	numEmptyCells := width - titleWidth - metadataWidth
	titleFill := strings.Repeat(" ", max(0, numEmptyCells))

	return lipgloss.JoinHorizontal(lipgloss.Top, titleRendered, titleFill, metadataRendered)
}

// This `input` interface is needed, even though inputAdapter[T] is the only implementation of it.
// This is to cover up the generic nature of inputAdapter, as input no longer has to be generic.
type input interface {
	update(tea.Msg) (input, tea.Cmd)
	view() string

	focus() tea.Cmd
	blur()
	setWidth(int)
	setTextColour(lipgloss.Color)

	value() any
	setValue(any) error
}

// Adapts an arbitrary input (like a `textinput`, `textarea`, `itempicker`) to be used as an `input` (interface above)
type inputAdapter[T viewable] struct {
	model    T
	updateFn func(T, tea.Msg) (T, tea.Cmd)
	// No adapter for View, because T is constrained viewable so we can use a blanket implementation
	focusFn         func(*T) tea.Cmd
	blurFn          func(*T)
	setWidthFn      func(*T, int)
	setTextColourFn func(*T, lipgloss.Color)
	valueFn         func(T) any
	setValueFn      func(*T, any) error
}

func (ia *inputAdapter[T]) update(message tea.Msg) (input, tea.Cmd) {
	var cmd tea.Cmd
	ia.model, cmd = ia.updateFn(ia.model, message)

	return ia, cmd
}

func (ia *inputAdapter[T]) view() string {
	return ia.model.View()
}

func (ia *inputAdapter[T]) focus() tea.Cmd {
	return ia.focusFn(&ia.model)
}

func (ia *inputAdapter[T]) blur() {
	ia.blurFn(&ia.model)
}

func (ia *inputAdapter[T]) setWidth(width int) {
	ia.setWidthFn(&ia.model, width)
}

func (ia *inputAdapter[T]) setTextColour(colour lipgloss.Color) {
	ia.setTextColourFn(&ia.model, colour)
}

func (ia *inputAdapter[T]) value() any {
	return ia.valueFn(ia.model)
}

func (ia *inputAdapter[T]) setValue(value any) error {
	return ia.setValueFn(&ia.model, value)
}

func newAdapterFrom(input any) input {
	switch input := input.(type) {
	case textinput.Model:
		return &inputAdapter[textinput.Model]{
			model: input,
			updateFn: func(model textinput.Model, message tea.Msg) (textinput.Model, tea.Cmd) {
				return model.Update(message)
			},
			focusFn: func(model *textinput.Model) tea.Cmd {
				return model.Focus()
			},
			blurFn: func(model *textinput.Model) {
				model.Blur()
			},
			setWidthFn: func(model *textinput.Model, width int) {
				// -3 for the prompt + cursor (I think), because textinput's definition of Width is weird af
				model.Width = width - 3
				// Redraw the model to handle overflow
				model.SetCursor(model.Position())
			},
			setTextColourFn: func(model *textinput.Model, colour lipgloss.Color) {
				style := lipgloss.NewStyle().Foreground(colour)
				model.TextStyle = style
				model.PromptStyle = style
				model.Cursor.Style = style
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
		return &inputAdapter[textarea.Model]{
			model: input,
			updateFn: func(model textarea.Model, message tea.Msg) (textarea.Model, tea.Cmd) {
				return model.Update(message)
			},
			focusFn: func(model *textarea.Model) tea.Cmd {
				return model.Focus()
			},
			blurFn: func(model *textarea.Model) {
				model.Blur()
			},
			setWidthFn: func(model *textarea.Model, width int) {
				model.SetWidth(width)
			},
			setTextColourFn: func(model *textarea.Model, colour lipgloss.Color) {},
			valueFn: func(model textarea.Model) any {
				return model.Value()
			},
			setValueFn: func(model *textarea.Model, value any) error {
				model.SetValue(value.(string))
				return nil
			},
		}

	case itempicker.Model:
		return &inputAdapter[itempicker.Model]{
			model: input,
			updateFn: func(model itempicker.Model, message tea.Msg) (itempicker.Model, tea.Cmd) {
				return model.Update(message)
			},
			focusFn:    func(model *itempicker.Model) tea.Cmd { return nil },
			blurFn:     func(model *itempicker.Model) {},
			setWidthFn: func(model *itempicker.Model, width int) {},
			setTextColourFn: func(model *itempicker.Model, colour lipgloss.Color) {
				model.Colour = colour
			},
			valueFn: func(model itempicker.Model) any {
				return model.Value()
			},
			setValueFn: func(model *itempicker.Model, value any) error {
				return model.SetValue(value.(itempicker.Item))
			},
		}

	case booleaninput.Model:
		return &inputAdapter[booleaninput.Model]{
			model: input,
			updateFn: func(model booleaninput.Model, message tea.Msg) (booleaninput.Model, tea.Cmd) {
				return model.Update(message)
			},
			focusFn:         func(*booleaninput.Model) tea.Cmd { return nil },
			blurFn:          func(*booleaninput.Model) {},
			setWidthFn:      func(model *booleaninput.Model, width int) {},
			setTextColourFn: func(model *booleaninput.Model, colour lipgloss.Color) {},
			valueFn: func(model booleaninput.Model) any {
				return model.Value()
			},
			setValueFn: func(model *booleaninput.Model, value any) error {
				model.SetValue(value.(bool))
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
	disabled    []bool
}

func newInputManager(inputs []any, names []string) *inputManager {
	inputsAdapted := make([]input, len(inputs))

	for i, input := range inputs {
		inputsAdapted[i] = newAdapterFrom(input)
	}

	disabled := slices.Repeat([]bool{false}, len(inputs))

	return &inputManager{
		inputs:   inputsAdapted,
		names:    names,
		disabled: disabled,
	}
}

func (im *inputManager) Update(message tea.Msg) (*inputManager, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		im.width = message.Width
		im.height = message.Height

		maxNameWidth := len(slices.MaxFunc(im.names, func(a, b string) int { return len(a) - len(b) }))
		// 4 is for padding + borders, 1 is for margin between name and input
		reservedWidth := maxNameWidth + 2*4 + 1

		for _, input := range im.inputs {
			input.setWidth(message.Width - reservedWidth)
		}

		return im, nil

	case meta.SwitchFocusMsg:
		im.inputs[im.activeInput].blur()

		var subCmd tea.Cmd
		// If new input is disabled, move again
		// TODO: this doesn't handle the edge case where all inputs are disabled, would give infinite recursion
		switch message.Direction {
		case meta.PREVIOUS:
			im.previous()
			if im.disabled[im.activeInput] {
				im.previous()
			}

		case meta.NEXT:
			im.next()
			if im.disabled[im.activeInput] {
				im.next()
			}
		}

		cmd := im.inputs[im.activeInput].focus()

		return im, tea.Batch(cmd, subCmd)

	case tea.KeyMsg:
		newInput, cmd := im.inputs[im.activeInput].update(message)

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
		Align(lipgloss.Left)

	if len(im.names) != len(im.inputs) {
		panic("what in the fuck")
	}

	// +2 for padding
	maxNameColWidth := len(slices.MaxFunc(im.names, func(name string, other string) int {
		return cmp.Compare(len(name), len(other))
	})) + 2

	nameStyle := sectionStyle.Width(maxNameColWidth)

	for i := range im.names {
		if i == im.activeInput {
			im.inputs[i].setTextColour(highlightColour)
		} else {
			im.inputs[i].setTextColour(lipgloss.Color(""))
		}

		if im.names[i] == "" {
			result.WriteString(sectionStyle.Render(im.inputs[i].view()))
		} else {
			var name, input string

			if im.disabled[i] {
				name = nameStyle.Foreground(lipgloss.ANSIColor(8)).Render(im.names[i])
				input = sectionStyle.Foreground(lipgloss.ANSIColor(8)).Render(im.inputs[i].view())
			} else {
				name = nameStyle.Render(im.names[i])
				input = sectionStyle.Render(im.inputs[i].view())
			}

			result.WriteString(lipgloss.JoinHorizontal(
				lipgloss.Top,
				name,
				" ",
				input,
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
