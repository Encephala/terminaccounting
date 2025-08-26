package view

import (
	"fmt"
	"local/bubbles/itempicker"
	"strconv"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
)

const (
	JOURNALINPUT activeInput = iota
	NOTESINPUT
)

type EntryCreateView struct {
	db *sqlx.DB

	JournalInput     itempicker.Model
	NotesInput       textarea.Model
	EntryRowsManager EntryRowCreateViewManager
	activeInput      activeInput

	colours styles.AppColours
}

type EntryRowCreateViewManager struct {
	rows    []*EntryRowCreateView
	numRows int
}

type EntryRowCreateView struct {
	ledgerInput  itempicker.Model
	accountInput itempicker.Model
	// TODO: documentInput as some file selector thing
	valueInput textinput.Model
}

func NewEntryCreateView(db *sqlx.DB, colours styles.AppColours) *EntryCreateView {
	journalInput := itempicker.New([]itempicker.Item{})
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	result := &EntryCreateView{
		db: db,

		JournalInput:     journalInput,
		NotesInput:       noteInput,
		activeInput:      JOURNALINPUT,
		EntryRowsManager: NewEntryRowCreateViewManager(),

		colours: colours,
	}

	return result
}

func NewEntryRowCreateViewManager() EntryRowCreateViewManager {
	rows := make([]*EntryRowCreateView, 1)

	rows[0] = &EntryRowCreateView{
		ledgerInput:  itempicker.New([]itempicker.Item{}),
		accountInput: itempicker.New([]itempicker.Item{}),
		valueInput:   textinput.New(),
	}

	return EntryRowCreateViewManager{
		rows:    rows,
		numRows: 1,
	}
}

func (cv *EntryCreateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(cv.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(cv.CommandSet())))

	cmds = append(cmds, database.MakeSelectJournalsCmd())

	return tea.Batch(cmds...)
}

func (cv *EntryCreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.SwitchFocusMsg:
		if cv.activeInput == NOTESINPUT {
			cv.NotesInput.Blur()
		}

		// Only two inputs, previous/next is equivalent
		cv.activeInput++
		cv.activeInput %= 2

		if cv.activeInput == NOTESINPUT {
			cv.NotesInput.Focus()
		}

		return cv, nil

	case meta.NavigateMsg:
		// Don't panic, just ignore the message
		return cv, nil

	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.JOURNAL:
			journals := message.Data.([]database.Journal)

			asSlice := make([]itempicker.Item, len(journals))
			for i, journal := range journals {
				asSlice[i] = journal
			}

			cv.JournalInput.SetItems(asSlice)

			return cv, nil

		case meta.LEDGER:
			ledgers := message.Data.([]database.Ledger)

			asSlice := make([]itempicker.Item, len(ledgers))
			for i, ledger := range ledgers {
				asSlice[i] = ledger
			}

			cv.EntryRowsManager.SetLedgers(asSlice)

			return cv, nil

		case meta.ACCOUNT:
			accounts := message.Data.([]database.Account)

			asSlice := make([]itempicker.Item, len(accounts))
			for i, account := range accounts {
				asSlice[i] = account
			}

			cv.EntryRowsManager.SetAccounts(asSlice)

			return cv, nil

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}

	case meta.CommitCreateMsg:
		journal := cv.JournalInput.Value().(database.Journal)
		notes := cv.NotesInput.Value()

		_ = database.Entry{
			Journal: journal.Id,
			Notes:   strings.Split(notes, "\n"),
		}

		// TODO: Actually create the entry in db and stuff.
		// id, err := newEntry.Insert(cv.db)

		// if err != nil {
		// 	return cv, meta.MessageCmd(err)
		// }

		cmd := meta.MessageCmd(meta.SwitchViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
		})

		return cv, cmd

	case tea.KeyMsg:
		var cmd tea.Cmd
		switch cv.activeInput {
		case JOURNALINPUT:
			cv.JournalInput, cmd = cv.JournalInput.Update(message)
		case NOTESINPUT:
			cv.NotesInput, cmd = cv.NotesInput.Update(message)

		default:
			panic(fmt.Sprintf("Updating create view but active input was %d", cv.activeInput))
		}

		return cv, cmd

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (cv *EntryCreateView) View() string {
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
	cv.NotesInput.SetWidth(inputWidth)

	// TODO: Render active input with a different colour
	var typeRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Journal"),
		" ",
		style.Width(cv.JournalInput.MaxViewLength()+2).AlignHorizontal(lipgloss.Left).Render(cv.JournalInput.View()),
	)

	var notesRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Note"),
		" ",
		style.Render(cv.NotesInput.View()),
	)

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			typeRow,
			notesRow,
		),
	))

	result.WriteString(cv.EntryRowsManager.View())

	return result.String()
}

func (cv *EntryCreateView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

func (cv *EntryCreateView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitCreateMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}

func (ercvm *EntryRowCreateViewManager) View() string {
	var result strings.Builder

	for i, row := range ercvm.rows {
		result.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			strconv.Itoa(i),
			" ",
			row.ledgerInput.View(),
			" ",
			row.accountInput.View(),
			" ",
			row.valueInput.View(),
		))

		if i < len(ercvm.rows)-1 {
			result.WriteString("\n\n")
		}
	}

	return result.String()
}

func (ercvm *EntryRowCreateViewManager) SetLedgers(ledgers []itempicker.Item) {
	for _, row := range ercvm.rows {
		row.ledgerInput.SetItems(ledgers)
	}
}

func (ercvm *EntryRowCreateViewManager) SetAccounts(accounts []itempicker.Item) {

	for _, row := range ercvm.rows {
		row.accountInput.SetItems(accounts)
	}
}
