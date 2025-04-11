package entries

import (
	"local/bubbles/itempicker"
	"strconv"
	"strings"
	"terminaccounting/meta"
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

type CreateView struct {
	db *sqlx.DB

	nameInput    textinput.Model
	journalInput itempicker.Model
	notesInput   textarea.Model

	colours styles.AppColours
}

func NewCreateView(db *sqlx.DB, colours styles.AppColours) *CreateView {
	nameInput := textinput.New()
	nameInput.Focus()
	nameInput.Cursor.SetMode(cursor.CursorStatic)
	journalInput := itempicker.New([]itempicker.Item{})
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	result := &CreateView{
		db: db,

		nameInput:    nameInput,
		journalInput: journalInput,
		notesInput:   noteInput,

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

func (cv *CreateView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return cv, nil
}

func (cv *CreateView) View() string {
	return "todo"
}

func (cv *CreateView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

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
