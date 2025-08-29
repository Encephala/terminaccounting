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
)

const (
	JOURNALINPUT activeInput = iota
	NOTESINPUT
	ENTRYROWINPUT
)

type EntryCreateView struct {
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
	dateInput    textinput.Model
	ledgerInput  itempicker.Model
	accountInput itempicker.Model
	// TODO: documentInput as some file selector thing
	// https://github.com/charmbracelet/bubbles/tree/master/filepicker
	debitInput  textinput.Model
	creditInput textinput.Model
}

func newEntryRowCreateView() *EntryRowCreateView {
	dateInput := textinput.New()
	dateInput.CharLimit = 10 // YYYY-MM-DD
	dateInput.Validate = func(input string) error {
		for _, char := range input {
			if !unicode.IsDigit(char) && char != '-' {
				return fmt.Errorf("invalid character %s", string(char))
			}
		}

		return nil
	}

	result := EntryRowCreateView{
		dateInput:    dateInput,
		ledgerInput:  itempicker.New([]itempicker.Item{}),
		accountInput: itempicker.New([]itempicker.Item{}),
		debitInput:   textinput.New(),
		creditInput:  textinput.New(),
	}

	return &result
}

func NewEntryCreateView(colours styles.AppColours) *EntryCreateView {
	journalInput := itempicker.New([]itempicker.Item{})
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	result := &EntryCreateView{
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
		if cv.activeInput != ENTRYROWINPUT {
			return cv, meta.MessageCmd(fmt.Errorf("hjkl navigation only works within the entryrows"))
		}

		var cmd tea.Cmd
		cv.EntryRowsManager, cmd = cv.EntryRowsManager.Update(message)

		return cv, cmd

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

	case tea.KeyMsg:
		var cmd tea.Cmd
		switch cv.activeInput {
		case JOURNALINPUT:
			cv.JournalInput, cmd = cv.JournalInput.Update(message)
		case NOTESINPUT:
			cv.NotesInput, cmd = cv.NotesInput.Update(message)
		case ENTRYROWINPUT:
			cv.EntryRowsManager, cmd = cv.EntryRowsManager.Update(message)

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

	case meta.CommitMsg:
		entryJournal := cv.JournalInput.Value().(database.Journal)
		entryNotes := cv.NotesInput.Value()

		entryRows, err := cv.EntryRowsManager.CompileRows()
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		newEntry := database.Entry{
			Journal: entryJournal.Id,
			Notes:   strings.Split(entryNotes, "\n"),
		}

		id, err := newEntry.Insert(entryRows)
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		return cv, meta.MessageCmd(meta.SwitchViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
			Data:     id,
		})

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

	// Default navigation
	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	// Create/delete rows
	normalMotions.Insert(meta.Motion{"d", "d"}, DeleteEntryRowMsg{})
	normalMotions.Insert(meta.Motion{"V", "d"}, DeleteEntryRowMsg{})
	normalMotions.Insert(meta.Motion{"V", "D"}, DeleteEntryRowMsg{})
	normalMotions.Insert(meta.Motion{"o"}, CreateEntryRowMsg{after: true})
	normalMotions.Insert(meta.Motion{"O"}, CreateEntryRowMsg{after: false})

	// hjkl navigation in entryrows
	normalMotions.Insert(meta.Motion{"h"}, meta.NavigateMsg{Direction: meta.LEFT})
	normalMotions.Insert(meta.Motion{"left"}, meta.NavigateMsg{Direction: meta.LEFT})
	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"down"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})
	normalMotions.Insert(meta.Motion{"up"}, meta.NavigateMsg{Direction: meta.UP})
	normalMotions.Insert(meta.Motion{"l"}, meta.NavigateMsg{Direction: meta.RIGHT})
	normalMotions.Insert(meta.Motion{"right"}, meta.NavigateMsg{Direction: meta.RIGHT})

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

	commands.Insert(meta.Command{"w"}, meta.CommitMsg{})

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

func (ercvm *EntryRowCreateViewManager) Update(msg tea.Msg) (*EntryRowCreateViewManager, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		highlightRow, highlightCol := ercvm.getActiveCoords()

		row := ercvm.rows[highlightRow]
		var cmd tea.Cmd
		switch highlightCol {
		case 0:
			row.dateInput, cmd = row.dateInput.Update(msg)
		case 1:
			row.ledgerInput, cmd = row.ledgerInput.Update(msg)
		case 2:
			row.accountInput, cmd = row.accountInput.Update(msg)
		case 3:
			if !validateNumberInput(msg) {
				return ercvm, meta.MessageCmd(fmt.Errorf("%s is not a valid character for a number", msg))
			}
			row.debitInput, cmd = row.debitInput.Update(msg)
			if row.creditInput.Value() != "" {
				row.creditInput.SetValue("")
			}
		case 4:
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

	case meta.NavigateMsg:
		oldRow, oldCol := ercvm.getActiveCoords()

		switch msg.Direction {
		case meta.DOWN:
			if oldRow == ercvm.numRows()-1 {
				break
			}
			ercvm.setActiveCoords(oldRow+1, oldCol)

		case meta.LEFT:
			if oldCol == 0 {
				break
			}
			ercvm.setActiveCoords(oldRow, oldCol-1)

		case meta.RIGHT:
			if oldCol == ercvm.numInputsPerRow()-1 {
				break
			}
			ercvm.setActiveCoords(oldRow, oldCol+1)

		case meta.UP:
			if oldRow == 0 {
				break
			}
			ercvm.setActiveCoords(oldRow-1, oldCol)

		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", msg.Direction))
		}
		return ercvm, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", msg))
	}
}

func (ercvm *EntryRowCreateViewManager) View(style, highlightStyle lipgloss.Style, isActive bool) string {
	// TODO?: render using the table bubble to have them fix all the alignment and stuff
	var result strings.Builder

	length := ercvm.numRows() + 1
	highlightRow, highlightCol := ercvm.getActiveCoords()

	var idCol []string = make([]string, length)
	var dateCol []string = make([]string, length)
	var ledgerCol []string = make([]string, length)
	var accountCol []string = make([]string, length)
	var debitCol []string = make([]string, length)
	var creditCol []string = make([]string, length)

	idCol[0] = "Row"
	dateCol[0] = "Date"
	ledgerCol[0] = "Ledger"
	accountCol[0] = "Account"
	debitCol[0] = "Debit"
	creditCol[0] = "Credit"

	for i, row := range ercvm.rows {
		idStyle := style
		dateStyle := style
		ledgerStyle := style
		accountStyle := style
		debitStyle := style
		creditStyle := style

		if isActive && i == highlightRow {
			switch highlightCol {
			case 0:
				dateStyle = highlightStyle
			case 1:
				ledgerStyle = highlightStyle
			case 2:
				accountStyle = highlightStyle
			case 3:
				debitStyle = highlightStyle
			case 4:
				creditStyle = highlightStyle
			default:
				panic(fmt.Sprintf("Unexpected highlighted column %d", highlightCol))
			}
		}

		idCol[i+1] = idStyle.Render(strconv.Itoa(i))
		dateCol[i+1] = dateStyle.Render(row.dateInput.View())
		ledgerCol[i+1] = ledgerStyle.Render(row.ledgerInput.View())
		accountCol[i+1] = accountStyle.Render(row.accountInput.View())
		debitCol[i+1] = debitStyle.Render(row.debitInput.View())
		creditCol[i+1] = creditStyle.Render(row.creditInput.View())
	}

	idRendered := lipgloss.JoinVertical(lipgloss.Left, idCol...)
	dateRendered := lipgloss.JoinVertical(lipgloss.Left, dateCol...)
	ledgerRendered := lipgloss.JoinVertical(lipgloss.Left, ledgerCol...)
	accountRendered := lipgloss.JoinVertical(lipgloss.Left, accountCol...)
	debitRendered := lipgloss.JoinVertical(lipgloss.Left, debitCol...)
	creditRendered := lipgloss.JoinVertical(lipgloss.Left, creditCol...)

	entryRows := style.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		idRendered,
		" ",
		dateRendered,
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
		if total == 0 {
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

func (ercvm *EntryRowCreateViewManager) CompileRows() ([]database.EntryRow, error) {
	result := make([]database.EntryRow, ercvm.numRows())

	total, err := ercvm.calculateCurrentTotal()
	if err != nil {
		return nil, err
	}

	if total != 0 {
		return nil, fmt.Errorf("entry has nonzero total value %s", total)
	}

	for i, formRow := range ercvm.rows {
		formLedger := formRow.ledgerInput.Value().(database.Ledger)
		formAccount := formRow.accountInput.Value().(database.Account)

		var accountId *int
		accountId = nil
		if formAccount.Id != -1 {
			accountId = &formAccount.Id
		}

		debitValue := formRow.debitInput.Value()
		creditValue := formRow.creditInput.Value()

		// Assert exactly one nonempty, because the createview should automatically clear the other field
		if debitValue == "" && creditValue == "" || debitValue != "" && creditValue != "" {
			panic(fmt.Sprintf(
				"expected only one of debit and credit nonzero in row %d, but got %s and %s",
				i, debitValue, creditValue))
		}

		var value database.CurrencyValue
		if debitValue != "" {
			debit, err := database.ParseDecimalValue(formRow.debitInput.Value())
			if err != nil {
				return nil, err
			}
			value = debit
		}
		if creditValue != "" {
			credit, err := database.ParseDecimalValue(formRow.creditInput.Value())
			if err != nil {
				return nil, err
			}
			value = credit
		}

		result[i] = database.EntryRow{
			Entry:      -1, // Will be inserted after entry itself has been inserted
			Ledger:     formLedger.Id,
			Account:    accountId,
			Document:   nil, // TODO
			Value:      value,
			Reconciled: false,
		}
	}

	return result, nil
}

// Returns preceeded/exceeded if the move would make the active input go "out of bounds"
func (ercvm *EntryRowCreateViewManager) switchFocus(direction meta.Sequence) (preceeded, exceeded bool) {
	activeRow, activeCol := ercvm.getActiveCoords()

	switch direction {
	case meta.PREVIOUS:
		if activeRow == 0 && activeCol == 0 {
			return true, false
		}

		ercvm.setActiveCoords(activeRow, activeCol-1)

	case meta.NEXT:
		if activeRow == ercvm.numRows()-1 && activeCol == ercvm.numInputsPerRow()-1 {
			ercvm.rows[activeRow].creditInput.Blur()
			return false, true
		}

		ercvm.setActiveCoords(activeRow, activeCol+1)

	default:
		panic(fmt.Sprintf("unexpected meta.Sequence: %#v", direction))
	}

	return false, false
}

func (ercvm *EntryRowCreateViewManager) calculateCurrentTotal() (database.CurrencyValue, error) {
	var total database.CurrencyValue

	for _, row := range ercvm.rows {
		if row.debitInput.Value() != "" {
			change, err := database.ParseDecimalValue(row.debitInput.Value())
			if err != nil {
				return 0, err
			}

			total = total.Add(change)
		}
		if row.creditInput.Value() != "" {
			change, err := database.ParseDecimalValue(row.creditInput.Value())
			if err != nil {
				return 0, err
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

func (ercvm *EntryRowCreateViewManager) numRows() int {
	return len(ercvm.rows)
}

func (ercvm *EntryRowCreateViewManager) numInputs() int {
	return ercvm.numRows() * ercvm.numInputsPerRow()
}

func (ercvm *EntryRowCreateViewManager) numInputsPerRow() int {
	return 5
}

func (ercvm *EntryRowCreateViewManager) getActiveCoords() (row, col int) {
	inputsPerRow := ercvm.numInputsPerRow()
	return ercvm.activeInput / inputsPerRow, ercvm.activeInput % inputsPerRow
}

func (ercvm *EntryRowCreateViewManager) focus(direction meta.Sequence) {
	numInputs := ercvm.numInputs()

	switch direction {
	case meta.PREVIOUS:
		ercvm.activeInput = numInputs - 1
		ercvm.rows[ercvm.numRows()-1].creditInput.Focus()

	case meta.NEXT:
		ercvm.activeInput = 0
		ercvm.rows[0].dateInput.Focus()
	}
}

// Ignores an input that would make the active input go "out of bounds"
func (ercvm *EntryRowCreateViewManager) setActiveCoords(newRow, newCol int) {
	numRow := ercvm.numRows()
	numPerRow := ercvm.numInputsPerRow()

	if newCol == -1 {
		newRow -= 1
		newCol = numPerRow - 1
	} else if newCol < -1 {
		panic("What")
	} else if newCol == numPerRow {
		newRow += 1
		newCol = 0
	} else if newCol > numPerRow {
		panic("What")
	}

	if newRow == -1 {
		return
	} else if newRow < -1 {
		panic("What")
	}
	if newRow == numRow {
		return
	} else if newRow > numRow {
		panic("What")
	}

	// Blur when leaving a textinput
	activeRow, activeCol := ercvm.getActiveCoords()
	switch activeCol {
	case 0:
		ercvm.rows[activeRow].dateInput.Blur()
	case 3:
		ercvm.rows[activeRow].debitInput.Blur()
	case 4:
		ercvm.rows[activeRow].creditInput.Blur()
	}

	ercvm.activeInput = newRow*numPerRow + newCol

	switch newCol {
	case 0:
		ercvm.rows[newRow].dateInput.Focus()
	case 3:
		ercvm.rows[newRow].debitInput.Focus()
	case 4:
		ercvm.rows[newRow].creditInput.Focus()
	}
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
	activeRow, _ := ercvm.getActiveCoords()

	// If trying to delete the last row in the entry
	// CBA handling weird edge cases here
	if ercvm.numRows() == 1 {
		return ercvm, meta.MessageCmd(fmt.Errorf("cannot delete the final entryrow"))
	}

	// If about to delete the bottom-most row, switch focus to one row above
	if activeRow == ercvm.numRows()-1 {
		ercvm.setActiveCoords(activeRow-1, 0)
	}

	ercvm.rows = append(ercvm.rows[:activeRow], ercvm.rows[activeRow+1:]...)

	return ercvm, nil
}

func (ercvm *EntryRowCreateViewManager) addRow(after bool) (*EntryRowCreateViewManager, tea.Cmd) {
	activeRow, _ := ercvm.getActiveCoords()

	newRow := newEntryRowCreateView()
	newRow.ledgerInput.Items = ercvm.rows[0].ledgerInput.Items
	newRow.accountInput.Items = ercvm.rows[0].accountInput.Items

	newRows := make([]*EntryRowCreateView, 0, ercvm.numRows()+1)

	if after {
		newRows = append(newRows, ercvm.rows[:activeRow+1]...)
		newRows = append(newRows, newRow)
		newRows = append(newRows, ercvm.rows[activeRow+1:]...)

		ercvm.rows = newRows

		ercvm.setActiveCoords(activeRow+1, 0)
	} else {
		// Blur old active input
		// Just blur them all, cba to write a switch activeCol {}
		ercvm.rows[activeRow].dateInput.Blur()
		ercvm.rows[activeRow].debitInput.Blur()
		ercvm.rows[activeRow].creditInput.Blur()
		newRows = append(newRows, ercvm.rows[:activeRow]...)
		newRows = append(newRows, newRow)
		newRows = append(newRows, ercvm.rows[activeRow:]...)

		ercvm.rows = newRows

		// Ensure new active input is focused if need be
		ercvm.setActiveCoords(activeRow, 0)
	}

	return ercvm, nil
}
