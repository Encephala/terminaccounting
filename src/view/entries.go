package view

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"terminaccounting/bubbles/itempicker"
	"terminaccounting/database"
	"terminaccounting/meta"
	"unicode"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TODO: Prefix entries to prevent clashes
const (
	JOURNALINPUT activeInput = iota
	NOTESINPUT
	ENTRYROWINPUT
)

type EntryCreateView struct {
	journalInput     itempicker.Model
	notesInput       textarea.Model
	entryRowsManager *EntryRowViewManager
	activeInput      activeInput

	colours meta.AppColours
}

func NewEntryCreateView() *EntryCreateView {
	journalInput := itempicker.New(database.AvailableJournalsAsItempickerItems())
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	result := &EntryCreateView{
		journalInput:     journalInput,
		notesInput:       noteInput,
		activeInput:      JOURNALINPUT,
		entryRowsManager: NewEntryRowViewManager(),

		colours: meta.ENTRIESCOLOURS,
	}

	return result
}

type EntryPrefillData struct {
	Journal database.Journal
	Rows    []database.EntryRow
	Notes   meta.Notes
}

// Make an EntryCreateView with the provided journal, rows prefilled into forms
func NewEntryCreateViewPrefilled(data EntryPrefillData) (*EntryCreateView, error) {
	result := NewEntryCreateView()

	result.journalInput.SetValue(data.Journal)
	result.notesInput.SetValue(data.Notes.Collapse())

	entryRowCreateView, err := decompileRows(data.Rows)
	if err != nil {
		return nil, err
	}

	result.entryRowsManager.rows = entryRowCreateView

	return result, nil
}

type entryRowCreator struct {
	dateInput        textinput.Model
	ledgerInput      itempicker.Model
	accountInput     itempicker.Model
	descriptionInput textinput.Model
	// TODO: documentInput as some file selector thing
	// https://github.com/charmbracelet/bubbles/tree/master/filepicker
	debitInput  textinput.Model
	creditInput textinput.Model
}

func newEntryRowCreator(startDate *database.Date) *entryRowCreator {
	dateInput := textinput.New()
	dateInput.Placeholder = "yyyy-MM-dd"
	dateInput.CharLimit = 10
	dateInput.Width = 10
	if startDate != nil {
		dateInput.SetValue(startDate.String())
	}

	ledgerInput := itempicker.New(database.AvailableLedgersAsItempickerItems())
	accountInput := itempicker.New(database.AvailableAccountsAsItempickerItems())

	result := entryRowCreator{
		dateInput:        dateInput,
		ledgerInput:      ledgerInput,
		accountInput:     accountInput,
		descriptionInput: textinput.New(),
		debitInput:       textinput.New(),
		creditInput:      textinput.New(),
	}

	return &result
}

func (cv *EntryCreateView) Init() tea.Cmd {
	return nil
}

func (cv *EntryCreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message.(type) {
	case meta.CommitMsg:
		entryJournal := cv.journalInput.Value()
		if entryJournal == nil {
			return cv, meta.MessageCmd(errors.New("no journal selected (none available)"))
		}
		entryNotes := cv.notesInput.Value()

		entryRows, err := cv.entryRowsManager.compileRows()
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		newEntry := database.Entry{
			Journal: entryJournal.(database.Journal).Id,
			Notes:   meta.CompileNotes(entryNotes),
		}

		id, err := newEntry.Insert(entryRows)
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully created Entry \"%d\"", id,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
			Data:     id,
		}))

		return cv, tea.Batch(cmds...)
	}

	return entriesCreateUpdateViewUpdate(cv, message)
}

func (cv *EntryCreateView) View() string {
	return entriesCreateUpdateViewView(cv)
}

func (cv *EntryCreateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL:  {},
		meta.ACCOUNTMODEL: {},
		meta.JOURNALMODEL: {},
	}
}

type DeleteEntryRowMsg struct{}
type CreateEntryRowMsg struct {
	after bool
}

func (cv *EntryCreateView) MotionSet() meta.MotionSet {
	return entriesCreateUpdateViewMotionSet()
}

func (cv *EntryCreateView) CommandSet() meta.CommandSet {
	return entriesCreateUpdateViewCommandSet()
}

func (cv *EntryCreateView) Reload() View {
	return NewEntryCreateView()
}

func (cv *EntryCreateView) getJournalInput() *itempicker.Model {
	return &cv.journalInput
}

func (cv *EntryCreateView) getNotesInput() *textarea.Model {
	return &cv.notesInput
}

func (cv *EntryCreateView) getManager() *EntryRowViewManager {
	return cv.entryRowsManager
}

func (cv *EntryCreateView) getActiveInput() *activeInput {
	return &cv.activeInput
}

func (cv *EntryCreateView) getColours() meta.AppColours {
	return cv.colours
}

func (cv *EntryCreateView) title() string {
	return "Creating new Entry"
}

type EntryUpdateView struct {
	journalInput     itempicker.Model
	notesInput       textarea.Model
	entryRowsManager *EntryRowViewManager
	activeInput      activeInput

	modelId           int
	startingEntry     database.Entry
	startingEntryRows []database.EntryRow

	colours meta.AppColours
}

func NewEntryUpdateView(modelId int) *EntryUpdateView {
	journalInput := itempicker.New(database.AvailableJournalsAsItempickerItems())

	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	result := &EntryUpdateView{
		journalInput:     journalInput,
		notesInput:       noteInput,
		activeInput:      JOURNALINPUT,
		entryRowsManager: NewEntryRowViewManager(),

		modelId: modelId,

		colours: meta.ENTRIESCOLOURS,
	}

	return result
}

func (uv *EntryUpdateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, database.MakeSelectEntryCmd(uv.modelId))
	cmds = append(cmds, database.MakeSelectEntryRowsCmd(uv.modelId))

	return tea.Batch(cmds...)
}

func (uv *EntryUpdateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.CommitMsg:
		entryJournal := uv.journalInput.Value()
		if entryJournal == nil {
			return uv, meta.MessageCmd(errors.New("no journal selected (none available)"))
		}
		entryNotes := uv.notesInput.Value()

		entryRows, err := uv.entryRowsManager.compileRows()
		if err != nil {
			return uv, meta.MessageCmd(err)
		}

		if uv.startingEntry.Id == 0 {
			panic("Updating entry but its starting value was not set")
		}

		newEntry := database.Entry{
			Id:      uv.startingEntry.Id,
			Journal: entryJournal.(database.Journal).Id,
			Notes:   meta.CompileNotes(entryNotes),
		}

		err = newEntry.Update(entryRows)
		if err != nil {
			return uv, meta.MessageCmd(err)
		}

		return uv, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully updated Entry \"%d\"", uv.modelId,
		)})

	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYMODEL:
			// NOTE: I assume a valid state of the database cache (ledgers/accounts/journals)

			entry := message.Data.(database.Entry)
			uv.startingEntry = entry

			journal, err := database.SelectJournal(entry.Journal)
			if err != nil {
				return uv, meta.MessageCmd(err)
			}

			err = uv.journalInput.SetValue(itempicker.Item(journal))
			if err != nil {
				return uv, meta.MessageCmd(err)
			}

			uv.notesInput.SetValue(entry.Notes.Collapse())

			return uv, nil

		case meta.ENTRYROWMODEL:
			// NOTE: I assume a valid state of the database cache (ledgers/accounts/journals)

			rows := message.Data.([]database.EntryRow)
			if len(rows) == 0 {
				panic(fmt.Sprintf("How did entry %d end up being empty?", uv.modelId))
			}

			uv.startingEntryRows = rows

			formRows, err := decompileRows(rows)
			if err != nil {
				return uv, meta.MessageCmd(err)
			}

			uv.getManager().rows = formRows

			return uv, nil
		}

	case meta.ResetInputFieldMsg:
		switch uv.activeInput {
		case JOURNALINPUT:
			availableJournalIndex := slices.IndexFunc(database.AvailableJournals, func(journal database.Journal) bool {
				return journal.Id == uv.startingEntry.Journal
			})

			if availableJournalIndex == -1 {
				panic("This won't happen, surely")
			}

			err := uv.journalInput.SetValue(database.AvailableJournals[availableJournalIndex])
			if err != nil {
				panic("This can't happen")
			}

		case NOTESINPUT:
			uv.notesInput.SetValue(uv.startingEntry.Notes.Collapse())

		case ENTRYROWINPUT:
			var err error
			uv.entryRowsManager.rows, err = decompileRows(uv.startingEntryRows)

			if err != nil {
				return uv, meta.MessageCmd(err)
			}

		default:
			panic(fmt.Sprintf("unexpected view.activeInput: %#v", uv.activeInput))
		}

		return uv, nil
	}

	return entriesCreateUpdateViewUpdate(uv, message)
}

func (uv *EntryUpdateView) View() string {
	return entriesCreateUpdateViewView(uv)
}

func (uv *EntryUpdateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL:   {},
		meta.ENTRYMODEL:    {},
		meta.ENTRYROWMODEL: {},
		meta.ACCOUNTMODEL:  {},
		meta.JOURNALMODEL:  {},
	}
}

func (uv *EntryUpdateView) MotionSet() meta.MotionSet {
	result := entriesCreateUpdateViewMotionSet()

	result.Normal.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	result.Normal.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return result
}

func (uv *EntryUpdateView) CommandSet() meta.CommandSet {
	return entriesCreateUpdateViewCommandSet()
}

func (uv *EntryUpdateView) Reload() View {
	return NewEntryUpdateView(uv.modelId)
}

func (uv *EntryUpdateView) getJournalInput() *itempicker.Model {
	return &uv.journalInput
}

func (uv *EntryUpdateView) getNotesInput() *textarea.Model {
	return &uv.notesInput
}

func (uv *EntryUpdateView) getManager() *EntryRowViewManager {
	return uv.entryRowsManager
}

func (uv *EntryUpdateView) getActiveInput() *activeInput {
	return &uv.activeInput
}

func (uv *EntryUpdateView) getColours() meta.AppColours {
	return uv.colours
}

func (uv *EntryUpdateView) title() string {
	// TODO get some name/id/whatever for the entry here?
	return fmt.Sprintf("Update Entry: %s", "TODO")
}

type EntryRowViewManager struct {
	rows []*entryRowCreator

	activeInput int
}

func NewEntryRowViewManager() *EntryRowViewManager {
	// Prefill with two empty rows
	rows := make([]*entryRowCreator, 2)

	rows[0] = newEntryRowCreator(database.Today())
	rows[1] = newEntryRowCreator(database.Today())

	return &EntryRowViewManager{
		rows: rows,
	}
}

func (ervm *EntryRowViewManager) update(msg tea.Msg) (*EntryRowViewManager, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// TODO
		return ervm, nil

	case tea.KeyMsg:
		highlightRow, highlightCol := ervm.getActiveCoords()

		row := ervm.rows[highlightRow]
		var cmd tea.Cmd
		switch highlightCol {
		case 0:
			if !validateDateInput(msg) {
				return ervm, meta.MessageCmd(fmt.Errorf("%q is not a valid character for a date", msg))
			}
			row.dateInput, cmd = row.dateInput.Update(msg)

		case 1:
			row.ledgerInput, cmd = row.ledgerInput.Update(msg)
		case 2:
			row.accountInput, cmd = row.accountInput.Update(msg)
		case 3:
			row.descriptionInput, cmd = row.descriptionInput.Update(msg)
		case 4:
			if !validateNumberInput(msg) {
				return ervm, meta.MessageCmd(fmt.Errorf("%q is not a valid character for a number", msg))
			}
			row.debitInput, cmd = row.debitInput.Update(msg)
			if row.creditInput.Value() != "" {
				row.creditInput.SetValue("")
			}
		case 5:
			if !validateNumberInput(msg) {
				return ervm, meta.MessageCmd(fmt.Errorf("%q is not a valid character for a number", msg))
			}
			row.creditInput, cmd = row.creditInput.Update(msg)
			if row.debitInput.Value() != "" {
				row.debitInput.SetValue("")
			}
		}

		ervm.rows[highlightRow] = row

		return ervm, cmd

	case meta.NavigateMsg:
		oldRow, oldCol := ervm.getActiveCoords()

		switch msg.Direction {
		case meta.LEFT:
			if oldCol == 0 {
				break
			}
			ervm.setActiveCoords(oldRow, oldCol-1)

		case meta.DOWN:
			if oldRow == ervm.numRows()-1 {
				break
			}
			ervm.setActiveCoords(oldRow+1, oldCol)

		case meta.UP:
			if oldRow == 0 {
				break
			}
			ervm.setActiveCoords(oldRow-1, oldCol)

		case meta.RIGHT:
			if oldCol == ervm.numInputsPerRow()-1 {
				break
			}
			ervm.setActiveCoords(oldRow, oldCol+1)

		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", msg.Direction))
		}
		return ervm, nil

	case meta.JumpHorizontalMsg:
		oldRow, _ := ervm.getActiveCoords()

		if msg.ToEnd {
			ervm.setActiveCoords(oldRow, ervm.numInputsPerRow()-1)
		} else {
			ervm.setActiveCoords(oldRow, 0)
		}

		return ervm, nil

	case meta.JumpVerticalMsg:
		_, oldCol := ervm.getActiveCoords()

		if msg.ToEnd {
			ervm.setActiveCoords(ervm.numRows()-1, oldCol)
		} else {
			ervm.setActiveCoords(0, oldCol)
		}

		return ervm, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", msg))
	}
}

func (ervm *EntryRowViewManager) view(isActive bool, highlightColour lipgloss.Color) string {
	columnStyle := lipgloss.NewStyle().MaxWidth(40)
	baseStyle := lipgloss.NewStyle()
	highlightStyle := baseStyle.Foreground(highlightColour)

	// TODO?: render using the table bubble to have that fix all the alignment and stuff
	var result strings.Builder

	length := ervm.numRows() + 1
	highlightRow, highlightCol := ervm.getActiveCoords()

	var idCol []string = make([]string, length)
	var dateCol []string = make([]string, length)
	var ledgerCol []string = make([]string, length)
	var accountCol []string = make([]string, length)
	var descriptionCol []string = make([]string, length)
	var debitCol []string = make([]string, length)
	var creditCol []string = make([]string, length)

	idCol[0] = "Row"
	dateCol[0] = "Date"
	ledgerCol[0] = "Ledger"
	accountCol[0] = "Account"
	descriptionCol[0] = "Description"
	debitCol[0] = "Debit"
	creditCol[0] = "Credit"

	for i, row := range ervm.rows {
		idStyle := baseStyle
		dateStyle := baseStyle
		ledgerStyle := baseStyle
		accountStyle := baseStyle
		descriptionStyle := baseStyle
		debitStyle := baseStyle
		creditStyle := baseStyle

		if isActive && i == highlightRow {
			switch highlightCol {
			case 0:
				dateStyle = highlightStyle
			case 1:
				ledgerStyle = highlightStyle
			case 2:
				accountStyle = highlightStyle
			case 3:
				descriptionStyle = highlightStyle
			case 4:
				debitStyle = highlightStyle
			case 5:
				creditStyle = highlightStyle
			default:
				panic(fmt.Sprintf("Unexpected highlighted column %d", highlightCol))
			}
		}

		idCol[i+1] = idStyle.Render(strconv.Itoa(i))
		dateCol[i+1] = dateStyle.Render(row.dateInput.View())
		ledgerCol[i+1] = ledgerStyle.Render(row.ledgerInput.View())
		accountCol[i+1] = accountStyle.Render(row.accountInput.View())
		descriptionCol[i+1] = descriptionStyle.Render(row.descriptionInput.View())
		debitCol[i+1] = debitStyle.Render(row.debitInput.View())
		creditCol[i+1] = creditStyle.Render(row.creditInput.View())
	}

	idRendered := columnStyle.Render(lipgloss.JoinVertical(lipgloss.Left, idCol...))
	dateRendered := columnStyle.Render(lipgloss.JoinVertical(lipgloss.Left, dateCol...))
	ledgerRendered := columnStyle.Render(lipgloss.JoinVertical(lipgloss.Left, ledgerCol...))
	accountRendered := columnStyle.Render(lipgloss.JoinVertical(lipgloss.Left, accountCol...))
	descriptionRendered := columnStyle.Render(lipgloss.JoinVertical(lipgloss.Left, descriptionCol...))
	debitRendered := columnStyle.Render(lipgloss.JoinVertical(lipgloss.Left, debitCol...))
	creditRendered := columnStyle.Render(lipgloss.JoinVertical(lipgloss.Left, creditCol...))

	entryRows := lipgloss.JoinHorizontal(
		lipgloss.Top,
		idRendered, " ",
		dateRendered, " ",
		ledgerRendered, " ",
		accountRendered, " ",
		descriptionRendered, " ",
		debitRendered, " ",
		creditRendered,
	)

	total, err := ervm.calculateCurrentTotal()
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

// Converts a slice of EntryRow "forms" to a slice of EntryRow
func (ervm *EntryRowViewManager) compileRows() ([]database.EntryRow, error) {
	result := make([]database.EntryRow, ervm.numRows())

	total, err := ervm.calculateCurrentTotal()
	if err != nil {
		return nil, err
	}

	if total != 0 {
		return nil, fmt.Errorf("entry has nonzero total value %s", total)
	}

	for i, formRow := range ervm.rows {
		formLedger := formRow.ledgerInput.Value()
		if formLedger == nil {
			return nil, fmt.Errorf("invalid ledger selected in row %d (none available)", i)
		}

		formAccount := formRow.accountInput.Value()
		if formAccount == nil {
			return nil, fmt.Errorf("invalid account selected in row %d (none available)", i)
		}
		var accountId *int
		if formAccount.(*database.Account) != nil {
			accountId = &formAccount.(*database.Account).Id
		}

		// TODO: Validate the date thingy
		date, err := database.ToDate(formRow.dateInput.Value())
		if err != nil {
			return nil, fmt.Errorf("row %d had date %q which isn't in yyyy-MM-dd:\n%#v", i, formRow.dateInput.Value(), err)
		}

		formDescription := formRow.descriptionInput.Value()
		formDebit := formRow.debitInput.Value()
		formCredit := formRow.creditInput.Value()

		// Assert not both nonempty, because the createview should automatically clear the other field
		if formDebit != "" && formCredit != "" {
			panic(fmt.Sprintf(
				"expected only one of debit and credit nonempty in row %d, but got %q and %q",
				i, formDebit, formCredit))
		}

		if formDebit == "" && formCredit == "" {
			return nil, fmt.Errorf("row %d had no value for both debit and credit", i)
		}

		var value database.CurrencyValue
		if formDebit != "" {
			debit, err := database.ParseCurrencyValue(formRow.debitInput.Value())
			if err != nil {
				return nil, err
			}
			if debit == 0 {
				return nil, fmt.Errorf("row %d had 0 as debit value, only nonzero allowed", i)
			}

			value = debit
		}
		if formCredit != "" {
			credit, err := database.ParseCurrencyValue(formRow.creditInput.Value())
			if err != nil {
				return nil, err
			}
			if credit == 0 {
				return nil, fmt.Errorf("row %d had 0 as credit value, only nonzero allowed", i)
			}

			value = -credit
		}

		result[i] = database.EntryRow{
			Entry:       -1, // Will be inserted into the struct after entry itself has been inserted into db
			Date:        date,
			Ledger:      formLedger.(database.Ledger).Id,
			Account:     accountId,
			Description: formDescription,
			Document:    nil, // TODO
			Value:       value,
			Reconciled:  false,
		}
	}

	return result, nil
}

func (uv *EntryUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingEntry}
	}
}

// Returns preceeded/exceeded if the move would make the active input go "out of bounds"
func (ervm *EntryRowViewManager) switchFocus(direction meta.Sequence) (preceeded, exceeded bool) {
	oldRow, oldCol := ervm.getActiveCoords()

	switch direction {
	case meta.PREVIOUS:
		if oldRow == 0 && oldCol == 0 {
			ervm.rows[0].dateInput.Blur()
			return true, false
		}

		ervm.setActiveCoords(oldRow, oldCol-1)

	case meta.NEXT:
		if oldRow == ervm.numRows()-1 && oldCol == ervm.numInputsPerRow()-1 {
			ervm.rows[oldRow].creditInput.Blur()
			return false, true
		}

		ervm.setActiveCoords(oldRow, oldCol+1)

	default:
		panic(fmt.Sprintf("unexpected meta.Sequence: %#v", direction))
	}

	return false, false
}

func (ervm *EntryRowViewManager) calculateCurrentTotal() (database.CurrencyValue, error) {
	var total database.CurrencyValue

	for _, row := range ervm.rows {
		if row.debitInput.Value() != "" {
			change, err := database.ParseCurrencyValue(row.debitInput.Value())
			if err != nil {
				return 0, err
			}

			total = total.Add(change)
		}
		if row.creditInput.Value() != "" {
			change, err := database.ParseCurrencyValue(row.creditInput.Value())
			if err != nil {
				return 0, err
			}

			total = total.Subtract(change)
		}
	}

	return total, nil
}

func (ervm *EntryRowViewManager) numRows() int {
	return len(ervm.rows)
}

func (ervm *EntryRowViewManager) numInputs() int {
	return ervm.numRows() * ervm.numInputsPerRow()
}

func (ervm *EntryRowViewManager) numInputsPerRow() int {
	return 6
}

func (ervm *EntryRowViewManager) getActiveCoords() (row, col int) {
	inputsPerRow := ervm.numInputsPerRow()
	return ervm.activeInput / inputsPerRow, ervm.activeInput % inputsPerRow
}

func (ervm *EntryRowViewManager) focus(direction meta.Sequence) {
	numInputs := ervm.numInputs()

	switch direction {
	case meta.PREVIOUS:
		ervm.activeInput = numInputs - 1
		ervm.rows[ervm.numRows()-1].creditInput.Focus()

	case meta.NEXT:
		ervm.activeInput = 0
		ervm.rows[0].dateInput.Focus()
	}
}

// Ignores an input that would make the active input go "out of bounds"
func (ervm *EntryRowViewManager) setActiveCoords(newRow, newCol int) {
	numRow := ervm.numRows()
	numPerRow := ervm.numInputsPerRow()

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
	// Have to do all this instead of leaving them all focussed, because then the cursor renders permanently
	oldRow, oldCol := ervm.getActiveCoords()
	switch oldCol {
	case 0:
		ervm.rows[oldRow].dateInput.Blur()
	case 3:
		ervm.rows[oldRow].descriptionInput.Blur()
	case 4:
		ervm.rows[oldRow].debitInput.Blur()
	case 5:
		ervm.rows[oldRow].creditInput.Blur()
	}

	ervm.activeInput = newRow*numPerRow + newCol

	switch newCol {
	case 0:
		ervm.rows[newRow].dateInput.Focus()
	case 3:
		ervm.rows[newRow].descriptionInput.Focus()
	case 4:
		ervm.rows[newRow].debitInput.Focus()
	case 5:
		ervm.rows[newRow].creditInput.Focus()
	}
}

// Converts a slice of EntryRow to a slice of EntryRowCreateView
func decompileRows(rows []database.EntryRow) ([]*entryRowCreator, error) {
	result := make([]*entryRowCreator, len(rows))

	for i, row := range rows {
		availableLedgerIndex := slices.IndexFunc(database.AvailableLedgers, func(ledger database.Ledger) bool {
			return ledger.Id == row.Ledger
		})
		if availableLedgerIndex == -1 {
			panic(fmt.Sprintf("Ledger not found for %#v", row))
		}

		ledger := database.AvailableLedgers[availableLedgerIndex]

		var account *database.Account
		if row.Account != nil {
			availableAccountIndex := slices.IndexFunc(database.AvailableAccounts, func(account database.Account) bool {
				return account.Id == *row.Account
			})
			if availableAccountIndex == -1 {
				panic(fmt.Sprintf("Account not found for %#v", row))
			}

			account = &database.AvailableAccounts[availableAccountIndex]
		}

		formRow := newEntryRowCreator(&row.Date)

		err := formRow.ledgerInput.SetValue(ledger)
		if err != nil {
			return nil, err
		}

		err = formRow.accountInput.SetValue(account)
		if err != nil {
			return nil, err
		}

		formRow.descriptionInput.SetValue(row.Description)

		if row.Value > 0 {
			formRow.debitInput.SetValue(row.Value.String())
		} else if row.Value < 0 {
			formRow.creditInput.SetValue((-row.Value).String())
		}

		result[i] = formRow
	}

	return result, nil
}

// Checks if input is a digit or a hyphen
func validateDateInput(msg tea.KeyMsg) bool {
	// These are (likely) control flow stuff, allow it
	if len(msg.Runes) > 1 || len(msg.Runes) == 0 {
		return true
	}

	character := msg.Runes[0]

	if unicode.IsDigit(character) {
		return true
	}

	if character == '-' {
		return true
	}

	return false
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

func (ervm *EntryRowViewManager) deleteRow() (*EntryRowViewManager, tea.Cmd) {
	activeRow, activeCol := ervm.getActiveCoords()

	// If trying to delete the last row in the entry
	// CBA handling weird edge cases here
	if ervm.numRows() == 1 {
		return ervm, meta.MessageCmd(errors.New("cannot delete the final entryrow"))
	}

	// If about to delete the bottom-most row
	newRow, newCol := activeRow, activeCol
	if activeRow == ervm.numRows()-1 {
		newRow -= 1

		// Switch focus first to avoid index out of bounds panic when unblurring oldRow
		ervm.setActiveCoords(newRow, newCol)

		ervm.rows = append(ervm.rows[:activeRow], ervm.rows[activeRow+1:]...)
	} else {
		// Switch focus after because otherwise the to-be-deleted row gets highlighted
		ervm.rows = append(ervm.rows[:activeRow], ervm.rows[activeRow+1:]...)

		ervm.setActiveCoords(newRow, newCol)
	}

	return ervm, nil
}

func (ervm *EntryRowViewManager) addRow(after bool) (*EntryRowViewManager, tea.Cmd) {
	activeRow, _ := ervm.getActiveCoords()

	var newRow *entryRowCreator

	// If the row that the new-row-creation was triggered from had a valid date,
	// prefill it in the new row. Otherwise, just leave new row empty
	prefillDate, parseErr := database.ToDate(ervm.rows[activeRow].dateInput.Value())
	if parseErr == nil {
		newRow = newEntryRowCreator(&prefillDate)
	} else {
		newRow = newEntryRowCreator(nil)
	}

	newRows := make([]*entryRowCreator, 0, ervm.numRows()+1)

	if after {
		// Insert after activeRow
		newRows = append(newRows, ervm.rows[:activeRow+1]...)
		newRows = append(newRows, newRow)
		newRows = append(newRows, ervm.rows[activeRow+1:]...)

		ervm.rows = newRows

		ervm.setActiveCoords(activeRow+1, 0)
	} else {
		// Insert before activeRow
		newRows = append(newRows, ervm.rows[:activeRow]...)
		newRows = append(newRows, newRow)
		newRows = append(newRows, ervm.rows[activeRow:]...)

		ervm.rows = newRows

		// Ensure new active input is focused if need be
		// Have to add 1 row to activeInput after inserting row above active
		// Fixing activeInput makes it correctly get blurred
		ervm.activeInput += ervm.numInputsPerRow()
		ervm.setActiveCoords(activeRow, 0)
	}

	return ervm, nil
}

type entryCreateOrUpdateView interface {
	View

	getJournalInput() *itempicker.Model
	getNotesInput() *textarea.Model
	getManager() *EntryRowViewManager

	getActiveInput() *activeInput

	getColours() meta.AppColours

	title() string
}

func entriesCreateUpdateViewUpdate(view entryCreateOrUpdateView, message tea.Msg) (tea.Model, tea.Cmd) {
	activeInput := view.getActiveInput()
	journalInput := view.getJournalInput()
	notesInput := view.getNotesInput()
	entryRowsManager := view.getManager()

	switch message := message.(type) {
	case meta.SwitchFocusMsg:
		if *activeInput == NOTESINPUT {
			notesInput.Blur()
		}

		if *activeInput != ENTRYROWINPUT {
			switch message.Direction {
			case meta.PREVIOUS:
				activeInput.previous(3)

			case meta.NEXT:
				activeInput.next(3)
			}

			// If it changed to entryrow input
			if *activeInput == ENTRYROWINPUT {
				entryRowsManager.focus(message.Direction)
			}
		} else {
			preceeded, exceeded := entryRowsManager.switchFocus(message.Direction)

			if exceeded {
				*activeInput = 0
			}
			if preceeded {
				*activeInput = 1
			}
		}

		if *activeInput == NOTESINPUT {
			notesInput.Focus()
		}

		return view, nil

	case meta.NavigateMsg:
		if *activeInput != ENTRYROWINPUT {
			return view, meta.MessageCmd(errors.New("hjkl navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.update(message)
		*entryRowsManager = *manager

		return view, cmd

	case meta.JumpHorizontalMsg:
		if *activeInput != ENTRYROWINPUT {
			return view, meta.MessageCmd(errors.New("$/_ navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.update(message)
		*entryRowsManager = *manager

		return view, cmd

	case meta.JumpVerticalMsg:
		if *activeInput != ENTRYROWINPUT {
			return view, meta.MessageCmd(errors.New("'gg'/'G' navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.update(message)
		*entryRowsManager = *manager

		return view, cmd

	case tea.WindowSizeMsg:
		// TODO

		return view, nil

	case tea.KeyMsg:
		var cmd tea.Cmd
		switch *activeInput {
		case JOURNALINPUT:
			*journalInput, cmd = journalInput.Update(message)
		case NOTESINPUT:
			*notesInput, cmd = notesInput.Update(message)
		case ENTRYROWINPUT:
			var manager *EntryRowViewManager
			manager, cmd = entryRowsManager.update(message)
			*entryRowsManager = *manager

		default:
			panic(fmt.Sprintf("Updating create view but active input was %d", *activeInput))
		}

		return view, cmd

	case DeleteEntryRowMsg:
		if *activeInput != ENTRYROWINPUT {
			return view, meta.MessageCmd(errors.New("no entry row highlighted while trying to delete one"))
		}

		manager, cmd := entryRowsManager.deleteRow()
		*entryRowsManager = *manager

		return view, cmd

	case CreateEntryRowMsg:
		if *activeInput != ENTRYROWINPUT {
			return view, meta.MessageCmd(errors.New("no entry row highlighted while trying to create one"))
		}

		var cmds []tea.Cmd

		manager, cmd := entryRowsManager.addRow(message.after)
		*entryRowsManager = *manager
		cmds = append(cmds, cmd)

		cmds = append(cmds, meta.MessageCmd(meta.SwitchModeMsg{
			InputMode: meta.INSERTMODE,
		}))

		return view, tea.Batch(cmds...)

	default:
		// TODO (waiting for https://github.com/charmbracelet/bubbles/issues/834)
		// panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
		slog.Warn("Unexpected tea.Msg", "message", message)
		return view, nil
	}
}

func entriesCreateUpdateViewView(view entryCreateOrUpdateView) string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(view.getColours().Background).Padding(0, 1).Margin(0, 0, 0, 2)

	result.WriteString(titleStyle.Render(view.title()))
	result.WriteString("\n\n")

	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)
	highlightStyle := sectionStyle.Foreground(view.getColours().Foreground)

	journalInput := view.getJournalInput()
	notesInput := view.getNotesInput()
	entryRowsManager := view.getManager()

	inputWidth := journalInput.MaxViewLength()

	notesInput.SetWidth(inputWidth)
	notesFocusStyle := lipgloss.NewStyle().Foreground(view.getColours().Foreground)
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	nameCol := lipgloss.JoinVertical(
		lipgloss.Right,
		sectionStyle.Render("Journal"),
		sectionStyle.Render("Notes"),
	)

	var inputCol string
	if *view.getActiveInput() == JOURNALINPUT {
		inputCol = lipgloss.JoinVertical(
			lipgloss.Left,
			highlightStyle.Width(inputWidth+2).AlignHorizontal(lipgloss.Left).Render(view.getJournalInput().View()),
			sectionStyle.Render(notesInput.View()),
		)
	} else if *view.getActiveInput() == NOTESINPUT {
		notesInput.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(view.getColours().Foreground)

		inputCol = lipgloss.JoinVertical(
			lipgloss.Left,
			sectionStyle.Width(inputWidth+2).AlignHorizontal(lipgloss.Left).Render(journalInput.View()),
			sectionStyle.Render(notesInput.View()),
		)
	} else {
		inputCol = lipgloss.JoinVertical(
			lipgloss.Left,
			sectionStyle.Width(inputWidth+2).AlignHorizontal(lipgloss.Left).Render(journalInput.View()),
			sectionStyle.Render(notesInput.View()),
		)
	}

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			nameCol,
			" ",
			inputCol,
		),
	))
	result.WriteString("\n\n")

	result.WriteString(sectionStyle.MarginLeft(2).Render(
		entryRowsManager.view(
			*view.getActiveInput() == ENTRYROWINPUT,
			view.getColours().Foreground,
		),
	))

	return result.String()
}

func entriesCreateUpdateViewMotionSet() meta.MotionSet {
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
	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})
	normalMotions.Insert(meta.Motion{"l"}, meta.NavigateMsg{Direction: meta.RIGHT})

	// Extra horizontal navigation
	normalMotions.Insert(meta.Motion{"$"}, meta.JumpHorizontalMsg{ToEnd: true})
	normalMotions.Insert(meta.Motion{"_"}, meta.JumpHorizontalMsg{ToEnd: false})

	// Extra vertical navigation
	normalMotions.Insert(meta.Motion{"g", "g"}, meta.JumpVerticalMsg{ToEnd: false})
	normalMotions.Insert(meta.Motion{"G"}, meta.JumpVerticalMsg{ToEnd: true})

	return meta.MotionSet{
		Normal: normalMotions,
	}
}

func entriesCreateUpdateViewCommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

type EntryDeleteView struct {
	modelId int // only for retrieving the model itself initially
	model   database.Entry

	colours meta.AppColours
}

func NewEntryDeleteView(modelId int) *EntryDeleteView {
	return &EntryDeleteView{
		modelId: modelId,

		colours: meta.ENTRIESCOLOURS,
	}
}

func (dv *EntryDeleteView) Init() tea.Cmd {
	return database.MakeLoadEntryDetailCmd(dv.modelId)
}

func (dv *EntryDeleteView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		entry := message.Data.(database.Entry)

		dv.model = entry

		return dv, nil

	case meta.CommitMsg:
		err := database.DeleteEntry(dv.model.Id)
		if err != nil {
			return dv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully deleted entry \"%d\"", dv.modelId,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE}))

		return dv, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		// TODO

		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *EntryDeleteView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(dv.colours.Background).Padding(0, 1).MarginLeft(2)
	style := lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder())
	rightStyle := style.Margin(0, 0, 0, 1)

	result.WriteString(titleStyle.Render(fmt.Sprintf("Delete Entry: %d", dv.model.Id)))
	result.WriteString("\n\n")

	journalRendered := lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Name"),
		rightStyle.Render(strconv.Itoa(dv.model.Journal)),
	)

	notesRendered := lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Notes"),
		rightStyle.Render(strings.Join(dv.model.Notes, "\n")),
	)

	var confirmRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Italic(true).Render("Run the `:w` command to confirm"),
	)

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			journalRendered,
			notesRendered,
			"",
			confirmRow,
		),
	))

	return result.String()
}

func (dv *EntryDeleteView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRYMODEL: {},
	}
}

func (dv *EntryDeleteView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *EntryDeleteView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (dv *EntryDeleteView) Reload() View {
	return NewEntryDeleteView(dv.modelId)
}

func (dv *EntryDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
