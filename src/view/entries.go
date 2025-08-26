package view

import (
	"fmt"
	"local/bubbles/itempicker"
	"log/slog"
	"strconv"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/styles"
	"unicode"

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
	ENTRYROWINPUT
)

type EntryCreateView struct {
	db *sqlx.DB

	JournalInput     itempicker.Model
	NotesInput       textarea.Model
	EntryRowsManager *EntryRowCreateViewManager
	activeInput      activeInput

	colours styles.AppColours
}

type EntryRowCreateViewManager struct {
	rows []*EntryRowCreateView

	activeInput int
}

type EntryRowCreateView struct {
	ledgerInput  itempicker.Model
	accountInput itempicker.Model
	// TODO: documentInput as some file selector thing
	// https://github.com/charmbracelet/bubbles/tree/master/filepicker
	debitInput  textinput.Model
	creditInput textinput.Model
}

func newEntryRowCreateView() *EntryRowCreateView {
	result := EntryRowCreateView{
		ledgerInput:  itempicker.New([]itempicker.Item{}),
		accountInput: itempicker.New([]itempicker.Item{}),
		debitInput:   textinput.New(),
		creditInput:  textinput.New(),
	}

	return &result
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

func (cv *EntryCreateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(cv.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(cv.CommandSet())))

	cmds = append(cmds, database.MakeSelectJournalsCmd(meta.ENTRIES))
	cmds = append(cmds, database.MakeSelectLedgersCmd(meta.ENTRIES))
	cmds = append(cmds, database.MakeSelectAccountsCmd(meta.ENTRIES))

	return tea.Batch(cmds...)
}

func (cv *EntryCreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.SwitchFocusMsg:
		if cv.activeInput == NOTESINPUT {
			cv.NotesInput.Blur()
		}

		if cv.activeInput != ENTRYROWINPUT {
			switch message.Direction {
			case meta.PREVIOUS:
				cv.activeInput--
				if cv.activeInput < 0 {
					cv.activeInput += 3
				}
			case meta.NEXT:
				cv.activeInput++
				cv.activeInput %= 3
			}

			// If it changed to entryrow input
			if cv.activeInput == ENTRYROWINPUT {
				cv.EntryRowsManager.focus(message.Direction)
			}
		} else {
			preceeded, exceeded := cv.EntryRowsManager.switchFocus(message.Direction)

			if exceeded {
				cv.activeInput = 0
			}
			if preceeded {
				cv.activeInput = 1
			}
		}

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

			cv.JournalInput.Items = asSlice

			return cv, nil

		case meta.LEDGER:
			ledgers := message.Data.([]database.Ledger)

			asSlice := make([]itempicker.Item, len(ledgers))
			for i, ledger := range ledgers {
				asSlice[i] = ledger
			}

			cv.EntryRowsManager.setLedgers(asSlice)

			return cv, nil

		case meta.ACCOUNT:
			accounts := message.Data.([]database.Account)

			asSlice := make([]itempicker.Item, len(accounts)+1)
			asSlice[0] = database.Account{Id: -1}
			for i, account := range accounts {
				asSlice[i+1] = account
			}

			cv.EntryRowsManager.setAccounts(asSlice)

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
		case ENTRYROWINPUT:
			cv.EntryRowsManager, cmd = cv.EntryRowsManager.handleKeyMsg(message)

		default:
			panic(fmt.Sprintf("Updating create view but active input was %d", cv.activeInput))
		}

		return cv, cmd

	case DeleteEntryRowMsg:
		if cv.activeInput != ENTRYROWINPUT {
			return cv, meta.MessageCmd(fmt.Errorf("no entry row highlighted while trying to delete one"))
		}

		var cmd tea.Cmd
		cv.EntryRowsManager, cmd = cv.EntryRowsManager.deleteRow()

		return cv, cmd

	case CreateEntryRowMsg:
		if cv.activeInput != ENTRYROWINPUT {
			return cv, meta.MessageCmd(fmt.Errorf("no entry row highlighted while trying to create one"))
		}

		var cmd tea.Cmd
		cv.EntryRowsManager, cmd = cv.EntryRowsManager.addRow(message.after)

		return cv, cmd

	default:
		// TODO (waiting for https://github.com/charmbracelet/bubbles/issues/834)
		// panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
		slog.Warn(fmt.Sprintf("unexpected tea.Msg: %#v", message))
		return cv, nil
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
	highlightStyle := style.Foreground(cv.colours.Foreground)

	inputWidth := cv.JournalInput.MaxViewLength()
	cv.NotesInput.SetWidth(inputWidth)
	cv.NotesInput.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(cv.colours.Foreground)
	cv.NotesInput.FocusedStyle.Text = lipgloss.NewStyle().Foreground(cv.colours.Foreground)
	cv.NotesInput.FocusedStyle.CursorLine = lipgloss.NewStyle().Foreground(cv.colours.Foreground)
	cv.NotesInput.FocusedStyle.LineNumber = lipgloss.NewStyle().Foreground(cv.colours.Foreground)

	nameCol := lipgloss.JoinVertical(
		lipgloss.Right,
		style.Render("Journal"),
		style.Render("Notes"),
	)

	var inputCol string
	if cv.activeInput == JOURNALINPUT {
		inputCol = lipgloss.JoinVertical(
			lipgloss.Left,
			highlightStyle.Width(inputWidth+2).AlignHorizontal(lipgloss.Left).Render(cv.JournalInput.View()),
			style.Render(cv.NotesInput.View()),
		)
	} else if cv.activeInput == NOTESINPUT {
		cv.NotesInput.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(cv.colours.Foreground)

		inputCol = lipgloss.JoinVertical(
			lipgloss.Left,
			style.Width(inputWidth+2).AlignHorizontal(lipgloss.Left).Render(cv.JournalInput.View()),
			style.Render(cv.NotesInput.View()),
		)
	} else {
		inputCol = lipgloss.JoinVertical(
			lipgloss.Left,
			style.Width(inputWidth+2).AlignHorizontal(lipgloss.Left).Render(cv.JournalInput.View()),
			style.Render(cv.NotesInput.View()),
		)
	}

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			nameCol,
			inputCol,
		),
	))
	result.WriteString("\n\n")

	result.WriteString(style.MarginLeft(2).Render(
		cv.EntryRowsManager.View(lipgloss.NewStyle(), lipgloss.NewStyle().Foreground(cv.colours.Foreground), cv.activeInput == ENTRYROWINPUT),
	))

	return result.String()
}

func (cv *EntryCreateView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	normalMotions.Insert(meta.Motion{"d", "d"}, DeleteEntryRowMsg{})
	normalMotions.Insert(meta.Motion{"V", "d"}, DeleteEntryRowMsg{})
	normalMotions.Insert(meta.Motion{"V", "D"}, DeleteEntryRowMsg{})

	normalMotions.Insert(meta.Motion{"o"}, CreateEntryRowMsg{after: true})
	normalMotions.Insert(meta.Motion{"O"}, CreateEntryRowMsg{after: false})

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

type DeleteEntryRowMsg struct{}
type CreateEntryRowMsg struct {
	after bool
}

func (cv *EntryCreateView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitCreateMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}

func NewEntryRowCreateViewManager() *EntryRowCreateViewManager {
	rows := make([]*EntryRowCreateView, 2)

	// Prefill with two empty rows
	rows[0] = newEntryRowCreateView()
	rows[1] = newEntryRowCreateView()

	return &EntryRowCreateViewManager{
		rows: rows,
	}
}

func (ercvm *EntryRowCreateViewManager) View(style, highlightStyle lipgloss.Style, isActive bool) string {
	// TODO?: render using the table bubble to have them fix all the alignment and stuff
	var result strings.Builder

	length := len(ercvm.rows) + 1
	highlightRow, highlightCol := ercvm.activeCoords()

	var idCol []string = make([]string, length)
	var ledgerCol []string = make([]string, length)
	var accountCol []string = make([]string, length)
	var debitCol []string = make([]string, length)
	var creditCol []string = make([]string, length)

	idCol[0] = "Row"
	ledgerCol[0] = "Ledger"
	accountCol[0] = "Account"
	debitCol[0] = "Debit"
	creditCol[0] = "Credit"

	for i, row := range ercvm.rows {
		idStyle := style
		ledgerStyle := style
		accountStyle := style
		debitStyle := style
		creditStyle := style

		if isActive && i == highlightRow {
			switch highlightCol {
			case 0:
				ledgerStyle = highlightStyle
			case 1:
				accountStyle = highlightStyle
			case 2:
				debitStyle = highlightStyle
			case 3:
				creditStyle = highlightStyle
			default:
				panic(fmt.Sprintf("Unexpected highlighted column %d", highlightCol))
			}
		}

		idCol[i+1] = idStyle.Render(strconv.Itoa(i + 1))
		ledgerCol[i+1] = ledgerStyle.Render(row.ledgerInput.View())
		accountCol[i+1] = accountStyle.Render(row.accountInput.View())
		debitCol[i+1] = debitStyle.Render(row.debitInput.View())
		creditCol[i+1] = creditStyle.Render(row.creditInput.View())
	}

	idRendered := lipgloss.JoinVertical(lipgloss.Left, idCol...)
	ledgerRendered := lipgloss.JoinVertical(lipgloss.Left, ledgerCol...)
	accountRendered := lipgloss.JoinVertical(lipgloss.Left, accountCol...)
	debitRendered := lipgloss.JoinVertical(lipgloss.Left, debitCol...)
	creditRendered := lipgloss.JoinVertical(lipgloss.Left, creditCol...)

	entryRows := style.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		idRendered,
		" ",
		ledgerRendered,
		" ",
		accountRendered,
		" ",
		debitRendered,
		" ",
		creditRendered,
	))

	total, err := ercvm.calculateCurrentTotal()
	var totalRendered string
	red := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1)).Italic(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(2))
	if err == nil {
		if total.Whole == 0 && total.Fractional == 0 {
			totalRendered = fmt.Sprintf("Total: %s", green.Render(total.String()))
		} else {
			totalRendered = fmt.Sprintf("Total: %s", total)
		}
	} else {
		totalRendered = red.Render("error")
	}

	result.WriteString(lipgloss.JoinVertical(
		lipgloss.Right,
		entryRows,
		totalRendered,
	))

	return result.String()
}

func (ercvm *EntryRowCreateViewManager) calculateCurrentTotal() (database.DecimalValue, error) {
	total := database.DecimalValue{}

	for _, row := range ercvm.rows {
		if row.debitInput.Value() != "" {
			change, err := database.ParseDecimalValue(row.debitInput.Value())
			if err != nil {
				return database.DecimalValue{}, err
			}

			total = total.Add(change)
		}
		if row.creditInput.Value() != "" {
			change, err := database.ParseDecimalValue(row.creditInput.Value())
			if err != nil {
				return database.DecimalValue{}, err
			}

			total = total.Subtract(change)
		}
	}

	return total, nil
}

func (ercvm *EntryRowCreateViewManager) setLedgers(ledgers []itempicker.Item) {
	for _, row := range ercvm.rows {
		row.ledgerInput.Items = ledgers
	}
}

func (ercvm *EntryRowCreateViewManager) setAccounts(accounts []itempicker.Item) {

	for _, row := range ercvm.rows {
		row.accountInput.Items = accounts
	}
}

func (ercvm *EntryRowCreateViewManager) numInputs() int {
	numRows := len(ercvm.rows)
	inputsPerRow := 4
	return numRows * inputsPerRow
}

func (ercvm *EntryRowCreateViewManager) activeCoords() (int, int) {
	inputsPerRow := 4
	return ercvm.activeInput / inputsPerRow, ercvm.activeInput % inputsPerRow
}

func (ercvm *EntryRowCreateViewManager) focus(direction meta.Sequence) {
	numInputs := ercvm.numInputs()

	switch direction {
	case meta.PREVIOUS:
		ercvm.activeInput = numInputs - 1

	case meta.NEXT:
		ercvm.activeInput = 0
	}
}

func (ercvm *EntryRowCreateViewManager) switchFocus(direction meta.Sequence) (preceeded, exceeded bool) {
	numInputs := ercvm.numInputs()

	// Blur when leaving a textinput
	highlightRow, highlightCol := ercvm.activeCoords()
	switch highlightCol {
	case 2:
		ercvm.rows[highlightRow].debitInput.Blur()
	case 3:
		ercvm.rows[highlightRow].creditInput.Blur()
	}

	switch direction {
	case meta.PREVIOUS:
		ercvm.activeInput--
		if ercvm.activeInput < 0 {
			return true, false
		}

	case meta.NEXT:
		ercvm.activeInput++
		if ercvm.activeInput >= numInputs {
			return false, true
		}
	}

	// Focus when entering a textinput
	highlightRow, highlightCol = ercvm.activeCoords()
	switch highlightCol {
	case 2:
		ercvm.rows[highlightRow].debitInput.Focus()
	case 3:
		ercvm.rows[highlightRow].creditInput.Focus()
	}

	return false, false
}

func (ercvm *EntryRowCreateViewManager) handleKeyMsg(msg tea.KeyMsg) (*EntryRowCreateViewManager, tea.Cmd) {
	highlightRow, highlightCol := ercvm.activeCoords()

	row := ercvm.rows[highlightRow]
	var cmd tea.Cmd
	switch highlightCol {
	case 0:
		row.ledgerInput, cmd = row.ledgerInput.Update(msg)
	case 1:
		row.accountInput, cmd = row.accountInput.Update(msg)
	case 2:
		if !validateNumberInput(msg) {
			return ercvm, meta.MessageCmd(fmt.Errorf("%s is not a valid character for a number", msg))
		}
		row.debitInput, cmd = row.debitInput.Update(msg)
		if row.creditInput.Value() != "" {
			row.creditInput.SetValue("")
		}
	case 3:
		if !validateNumberInput(msg) {
			return ercvm, meta.MessageCmd(fmt.Errorf("%s is not a valid character for a number", msg))
		}
		row.creditInput, cmd = row.creditInput.Update(msg)
		if row.debitInput.Value() != "" {
			row.debitInput.SetValue("")
		}
	}

	ercvm.rows[highlightRow] = row

	return ercvm, cmd
}

// Checks if input is a digit or a period.
// NOTE: don't allow -, a negative debit is just a positive credit
func validateNumberInput(msg tea.KeyMsg) bool {
	// These are (likely) control flow stuff, allow it
	if len(msg.Runes) > 1 || len(msg.Runes) == 0 {
		return true
	}

	character := msg.Runes[0]

	if unicode.IsDigit(character) {
		return true
	}

	if character == '.' {
		return true
	}

	return false
}

func (ercvm *EntryRowCreateViewManager) deleteRow() (*EntryRowCreateViewManager, tea.Cmd) {
	highlightRow, _ := ercvm.activeCoords()

	// If trying to delete the last row in the entry
	// CBA handling weird edge cases here
	if len(ercvm.rows) == 1 {
		return ercvm, meta.MessageCmd(fmt.Errorf("cannot delete the final entryrow"))
	}

	// If about to delete the bottom-most row, switch focus to one row above
	if highlightRow == len(ercvm.rows)-1 {
		// Bit inefficient, but it handles blur/focus and stuff properly
		for i := 0; i < 4; i++ {
			preceeded, exceeded := ercvm.switchFocus(meta.PREVIOUS)
			if preceeded || exceeded {
				panic("How did this ever happen")
			}
		}
	}

	ercvm.rows = append(ercvm.rows[:highlightRow], ercvm.rows[highlightRow+1:]...)

	return ercvm, nil
}

func (ercvm *EntryRowCreateViewManager) addRow(after bool) (*EntryRowCreateViewManager, tea.Cmd) {
	highlightRow, _ := ercvm.activeCoords()

	newRow := newEntryRowCreateView()
	newRow.ledgerInput.Items = ercvm.rows[0].ledgerInput.Items
	newRow.accountInput.Items = ercvm.rows[0].accountInput.Items

	newRows := make([]*EntryRowCreateView, 0, len(ercvm.rows)+1)

	if after {
		newRows = append(newRows, ercvm.rows[:highlightRow+1]...)
		newRows = append(newRows, newRow)
		newRows = append(newRows, ercvm.rows[highlightRow+1:]...)

		ercvm.rows = newRows

		for i := 0; i < 4; i++ {
			preceeded, exceeded := ercvm.switchFocus(meta.NEXT)
			if preceeded || exceeded {
				panic("How did this ever happen v2")
			}
		}
	} else {
		// Blur old active input
		ercvm.rows[highlightRow].debitInput.Blur()
		ercvm.rows[highlightRow].creditInput.Blur()
		newRows = append(newRows, ercvm.rows[:highlightRow]...)
		newRows = append(newRows, newRow)
		newRows = append(newRows, ercvm.rows[highlightRow:]...)

		ercvm.rows = newRows

		// Ensure new active input is focused if need be
		ercvm.switchFocus(meta.NEXT)
		ercvm.switchFocus(meta.PREVIOUS)
	}

	return ercvm, nil
}
