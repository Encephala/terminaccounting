package entries

import (
	"fmt"
	"local/bubbles/itempicker"
	"strconv"
	"strings"
	"terminaccounting/apps/journals"
	"terminaccounting/meta"
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
)

func (e Entry) FilterValue() string {
	var result strings.Builder
	result.WriteString(strconv.Itoa(e.Id))
	result.WriteString(strconv.Itoa(e.Journal))
	result.WriteString(strings.Join(e.Notes, ";"))
	return result.String()
}

func (e Entry) Title() string {
	return strconv.Itoa(e.Id)
}

func (e Entry) Description() string {
	return strings.Join(e.Notes, "; ")
}

func (er EntryRow) FilterValue() string {
	var result strings.Builder

	result.WriteString(strconv.Itoa(er.Id))

	// TODO: Get entry name, ledger name, account name etc.
	// Maybe I do want to maintain a `[]Ledger` array in ledgers app etc.,
	// for this. Makes sense maybe.
	// Then again, import loops and all. Maybe the main program needs a way to query these things?
	// Or a just a bunch of DB queries.
	// I mean I guess they're just lookups by primary key, that's fiiiine?
	// Probably runs every time the search box updates, maybe it's not "fiiiine".
	result.WriteString(strconv.Itoa(er.Entry))
	result.WriteString(strconv.Itoa(er.Ledger))
	result.WriteString(strconv.Itoa(*er.Account))

	result.WriteString(strconv.Itoa(int(er.Value.Whole)))
	result.WriteString(strconv.Itoa(int(er.Value.Fractional)))

	return result.String()
}

func (er EntryRow) Title() string {
	return strconv.Itoa(er.Id)
}

func (er EntryRow) Description() string {
	return strconv.Itoa(er.Id)
}

type activeInput int

const (
	JOURNALINPUT activeInput = iota
	NOTESINPUT
)

type CreateView struct {
	db *sqlx.DB

	journalInput itempicker.Model
	notesInput   textarea.Model
	activeInput  activeInput

	colours styles.AppColours
}

func NewCreateView(db *sqlx.DB, colours styles.AppColours) *CreateView {
	journalInput := itempicker.New([]itempicker.Item{})
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	result := &CreateView{
		db: db,

		journalInput: journalInput,
		notesInput:   noteInput,
		activeInput:  JOURNALINPUT,

		colours: colours,
	}

	return result
}

func (cv *CreateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(cv.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(cv.CommandSet())))

	cmds = append(cmds, makeSelectJournalsCmd(cv.db))

	return tea.Batch(cmds...)
}

func (cv *CreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.SwitchFocusMsg:
		if cv.activeInput == NOTESINPUT {
			cv.notesInput.Blur()
		}

		// Only two inputs, previous/next is equivalent
		cv.activeInput++
		cv.activeInput %= 2

		if cv.activeInput == NOTESINPUT {
			cv.notesInput.Focus()
		}

		return cv, nil

	case meta.NavigateMsg:
		// Don't panic, just ignore the message
		return cv, nil

	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.JOURNAL:
			journals := message.Data.([]journals.Journal)

			asSlice := make([]itempicker.Item, len(journals))
			for i, journal := range journals {
				asSlice[i] = journal
			}

			cv.journalInput.SetItems(asSlice)

			return cv, nil

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}

	case tea.KeyMsg:
		var cmd tea.Cmd
		switch cv.activeInput {
		case JOURNALINPUT:
			cv.journalInput, cmd = cv.journalInput.Update(message)
		case NOTESINPUT:
			cv.notesInput, cmd = cv.notesInput.Update(message)

		default:
			panic(fmt.Sprintf("Updating create view but active input was %d", cv.activeInput))
		}

		return cv, cmd

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (cv *CreateView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(cv.colours.Background).Padding(0, 1)

	result.WriteString(fmt.Sprintf("  %s", titleStyle.Render("Create new Entry")))
	result.WriteString("\n\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)

	const inputWidth = 26
	cv.notesInput.SetWidth(inputWidth)

	// TODO: Render active input with a different colour
	var typeRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Journal"),
		" ",
		style.Width(cv.journalInput.MaxViewLength()+2).AlignHorizontal(lipgloss.Left).Render(cv.journalInput.View()),
	)

	var notesRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Note"),
		" ",
		style.Render(cv.notesInput.View()),
	)

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			typeRow,
			notesRow,
		),
	))

	return result.String()
}

func (cv *CreateView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

func (cv *CreateView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitCreateMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}
