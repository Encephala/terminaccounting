package view

import (
	"cmp"
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
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type entriesDetailView struct {
	// The entries whose rows are being shown
	modelId int
	model   database.Entry

	viewer *entryRowViewer

	showReconciledRows  bool
	showReconciledTotal bool
}

func NewEntriesDetailView(modelId int) *entriesDetailView {
	return &entriesDetailView{
		modelId: modelId,

		viewer: newEntryRowViewer(meta.ENTRIESCOLOURS),
	}
}

func (dv *entriesDetailView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, database.MakeLoadEntriesDetailCmd(dv.modelId))
	cmds = append(cmds, database.MakeLoadEntriesRowsCmd(dv.modelId))

	return tea.Batch(cmds...)
}

func (dv *entriesDetailView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYMODEL:
			dv.model = message.Data.(database.Entry)

			return dv, nil

		case meta.ENTRYROWMODEL:
			newView, cmd := genericDetailViewUpdate(dv, message)

			return newView.(View), cmd

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}
	}

	newView, cmd := genericDetailViewUpdate(dv, message)

	return newView.(View), cmd
}

func (dv *entriesDetailView) View() string {
	return genericDetailViewView(dv)
}

func (dv *entriesDetailView) title() string {
	return fmt.Sprintf("Entry %d details", dv.model.Id)
}

func (dv *entriesDetailView) AllowsInsertMode() bool {
	return false
}

func (dv *entriesDetailView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRYMODEL:    {},
		meta.ENTRYROWMODEL: {},
	}
}

func (dv *entriesDetailView) MotionSet() meta.MotionSet {
	result := genericDetailViewMotionSet()

	result.Normal.Insert(meta.Motion{"g", "x"}, meta.SwitchViewMsg{ViewType: meta.DELETEVIEWTYPE, Data: dv.modelId})
	result.Normal.Insert(meta.Motion{"g", "e"}, meta.SwitchViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: dv.modelId})

	return result
}

func (dv *entriesDetailView) CommandSet() meta.CommandSet {
	return genericDetailViewCommandSet()
}

func (dv *entriesDetailView) Reload() View {
	return NewEntriesDetailView(dv.modelId)
}

func (dv *entriesDetailView) getViewer() *entryRowViewer {
	return dv.viewer
}

func (dv *entriesDetailView) getShowReconciledRows() *bool {
	return &dv.showReconciledRows
}

func (dv *entriesDetailView) getShowReconciledTotal() *bool {
	return &dv.showReconciledTotal
}

func (dv *entriesDetailView) getColours() meta.AppColours {
	return meta.ENTRIESCOLOURS
}

// NOTE: entries doesn't use the genericMutateView, because with the row creating it's too idiosyncratic

const (
	ENTRIESJOURNALINPUT int = iota
	ENTRIESNOTESINPUT
	ENTRIESROWINPUT
)

type entryCreateView struct {
	journalInput     itempicker.Model
	notesInput       textarea.Model
	entryRowsManager *rowsViewManager
	activeInput      int

	colours meta.AppColours
}

func NewEntryCreateView() *entryCreateView {
	colours := meta.ENTRIESCOLOURS

	journalInput := itempicker.New(database.AvailableJournalsAsItempickerItems())
	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	notesFocusStyle := lipgloss.NewStyle().Foreground(colours.Foreground)
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	result := &entryCreateView{
		journalInput:     journalInput,
		notesInput:       notesInput,
		activeInput:      ENTRIESJOURNALINPUT,
		entryRowsManager: newRowsViewManager(),

		colours: colours,
	}

	return result
}

type EntryPrefillData struct {
	Journal database.Journal
	Rows    []database.EntryRow
	Notes   meta.Notes
}

// Make an EntryCreateView with the provided journal, rows prefilled into forms
func NewEntryCreateViewPrefilled(data EntryPrefillData) (*entryCreateView, error) {
	result := NewEntryCreateView()

	result.journalInput.SetValue(data.Journal)
	result.notesInput.SetValue(data.Notes.Collapse())

	entryRowCreateView, err := decompileRows(data.Rows, result.entryRowsManager.colWidths)
	if err != nil {
		return nil, err
	}

	result.entryRowsManager.rows = entryRowCreateView

	return result, nil
}

type rowsCreator struct {
	dateInput        textinput.Model
	ledgerInput      itempicker.Model
	accountInput     itempicker.Model
	descriptionInput textinput.Model
	// TODO: documentInput as some file selector thing
	// https://github.com/charmbracelet/bubbles/tree/master/filepicker
	debitInput  textinput.Model
	creditInput textinput.Model
}

func newRowCreator(startDate *database.Date, colWidths []int) *rowsCreator {
	dateInput := textinput.New()
	dateInput.Placeholder = "yyyy-MM-dd"
	dateInput.CharLimit = 10
	dateInput.Width = 10
	if startDate != nil {
		dateInput.SetValue(startDate.String())
	}

	ledgerInput := itempicker.New(database.AvailableLedgersAsItempickerItems())
	accountInput := itempicker.New(database.AvailableAccountsAsItempickerItems())

	// -3 is for prompt and cursor
	descriptionInput := textinput.New()
	descriptionInput.Width = colWidths[4] - 3
	debitInput := textinput.New()
	debitInput.Width = colWidths[5] - 3
	creditInput := textinput.New()
	creditInput.Width = colWidths[6] - 3

	result := rowsCreator{
		dateInput:        dateInput,
		ledgerInput:      ledgerInput,
		accountInput:     accountInput,
		descriptionInput: descriptionInput,
		debitInput:       debitInput,
		creditInput:      creditInput,
	}

	return &result
}

func (cv *entryCreateView) Init() tea.Cmd {
	return nil
}

func (cv *entryCreateView) Update(message tea.Msg) (View, tea.Cmd) {
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

	return entriesMutateViewUpdate(cv, message)
}

func (cv *entryCreateView) View() string {
	return entriesMutateViewView(cv)
}

func (cv *entryCreateView) AllowsInsertMode() bool {
	return true
}

func (cv *entryCreateView) AcceptedModels() map[meta.ModelType]struct{} {
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

func (cv *entryCreateView) MotionSet() meta.MotionSet {
	return entriesMutateViewMotionSet()
}

func (cv *entryCreateView) CommandSet() meta.CommandSet {
	return entriesMutateViewCommandSet()
}

func (cv *entryCreateView) Reload() View {
	return NewEntryCreateView()
}

func (cv *entryCreateView) getJournalInput() *itempicker.Model {
	return &cv.journalInput
}

func (cv *entryCreateView) getNotesInput() *textarea.Model {
	return &cv.notesInput
}

func (cv *entryCreateView) getManager() *rowsViewManager {
	return cv.entryRowsManager
}

func (cv *entryCreateView) getActiveInput() *int {
	return &cv.activeInput
}

func (cv *entryCreateView) getColours() meta.AppColours {
	return cv.colours
}

func (cv *entryCreateView) title() string {
	return "Creating new Entry"
}

type entryUpdateView struct {
	journalInput     itempicker.Model
	notesInput       textarea.Model
	entryRowsManager *rowsViewManager
	activeInput      int

	modelId           int
	startingEntry     database.Entry
	startingEntryRows []database.EntryRow

	colours meta.AppColours
}

func NewEntryUpdateView(modelId int) *entryUpdateView {
	colours := meta.ENTRIESCOLOURS

	journalInput := itempicker.New(database.AvailableJournalsAsItempickerItems())

	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	notesFocusStyle := lipgloss.NewStyle().Foreground(colours.Foreground)
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	result := &entryUpdateView{
		journalInput:     journalInput,
		notesInput:       notesInput,
		activeInput:      ENTRIESJOURNALINPUT,
		entryRowsManager: newRowsViewManager(),

		modelId: modelId,

		colours: colours,
	}

	return result
}

func (uv *entryUpdateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, database.MakeSelectEntryCmd(uv.modelId))
	cmds = append(cmds, database.MakeSelectEntryRowsCmd(uv.modelId))

	return tea.Batch(cmds...)
}

func (uv *entryUpdateView) Update(message tea.Msg) (View, tea.Cmd) {
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

			formRows, err := decompileRows(rows, uv.entryRowsManager.colWidths)
			if err != nil {
				return uv, meta.MessageCmd(err)
			}

			uv.getManager().rows = formRows

			return uv, nil
		}

	case meta.ResetInputFieldMsg:
		switch uv.activeInput {
		case ENTRIESJOURNALINPUT:
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

		case ENTRIESNOTESINPUT:
			uv.notesInput.SetValue(uv.startingEntry.Notes.Collapse())

		case ENTRIESROWINPUT:
			var err error
			uv.entryRowsManager.rows, err = decompileRows(uv.startingEntryRows, uv.entryRowsManager.colWidths)

			if err != nil {
				return uv, meta.MessageCmd(err)
			}

		default:
			panic(fmt.Sprintf("unexpected view.activeInput: %#v", uv.activeInput))
		}

		return uv, nil
	}

	return entriesMutateViewUpdate(uv, message)
}

func (uv *entryUpdateView) View() string {
	return entriesMutateViewView(uv)
}

func (uv *entryUpdateView) AllowsInsertMode() bool {
	return true
}

func (uv *entryUpdateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL:   {},
		meta.ENTRYMODEL:    {},
		meta.ENTRYROWMODEL: {},
		meta.ACCOUNTMODEL:  {},
		meta.JOURNALMODEL:  {},
	}
}

func (uv *entryUpdateView) MotionSet() meta.MotionSet {
	result := entriesMutateViewMotionSet()

	result.Normal.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	result.Normal.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return result
}

func (uv *entryUpdateView) CommandSet() meta.CommandSet {
	return entriesMutateViewCommandSet()
}

func (uv *entryUpdateView) Reload() View {
	return NewEntryUpdateView(uv.modelId)
}

func (uv *entryUpdateView) getJournalInput() *itempicker.Model {
	return &uv.journalInput
}

func (uv *entryUpdateView) getNotesInput() *textarea.Model {
	return &uv.notesInput
}

func (uv *entryUpdateView) getManager() *rowsViewManager {
	return uv.entryRowsManager
}

func (uv *entryUpdateView) getActiveInput() *int {
	return &uv.activeInput
}

func (uv *entryUpdateView) getColours() meta.AppColours {
	return uv.colours
}

func (uv *entryUpdateView) title() string {
	// TODO get some name/id/whatever for the entry here?
	return fmt.Sprintf("Update Entry: %s", "TODO")
}

type rowsViewManager struct {
	rows []*rowsCreator

	isActive    bool
	activeInput int

	colWidths []int
	viewport  viewport.Model
}

func newRowsViewManager() *rowsViewManager {
	// Prefill with two empty rows
	rows := make([]*rowsCreator, 2)

	colWidths := []int{3, 13, 20, 20, 30, 15, 15}
	rows[0] = newRowCreator(database.Today(), colWidths)
	rows[1] = newRowCreator(database.Today(), colWidths)

	totalWidth := 130

	return &rowsViewManager{
		rows: rows,

		colWidths: colWidths,
		viewport:  viewport.New(totalWidth, 16),
	}
}

func (ervm *rowsViewManager) Update(msg tea.Msg) (*rowsViewManager, tea.Cmd) {
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

func (ervm *rowsViewManager) View() string {
	ervm.updateContent()

	var result strings.Builder

	headers := []string{"Row", "Date", "Ledger", "Account", "Description", "Debit", "Credit"}
	result.WriteString(ervm.renderRow(headers))

	result.WriteString("\n")

	result.WriteString(ervm.viewport.View())

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

	result.WriteString("\n")

	result.WriteString(totalRendered)

	return result.String()
}

func (ervm *rowsViewManager) updateContent() {
	baseStyle := lipgloss.NewStyle()
	highlightStyle := baseStyle.Foreground(meta.ENTRIESCOLOURS.Foreground)

	var result strings.Builder

	highlightRow, highlightCol := ervm.getActiveCoords()

	for i, row := range ervm.rows {
		var currentRow []string
		idStyle := baseStyle
		dateStyle := baseStyle
		ledgerStyle := baseStyle
		accountStyle := baseStyle
		descriptionStyle := baseStyle
		debitStyle := baseStyle
		creditStyle := baseStyle

		if ervm.isActive && i == highlightRow {
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

		currentRow = append(currentRow, idStyle.Render(strconv.Itoa(i)))
		currentRow = append(currentRow, dateStyle.Render(row.dateInput.View()))
		currentRow = append(currentRow, ledgerStyle.Render(row.ledgerInput.View()))
		currentRow = append(currentRow, accountStyle.Render(row.accountInput.View()))
		currentRow = append(currentRow, descriptionStyle.Render(row.descriptionInput.View()))
		currentRow = append(currentRow, debitStyle.Render(row.debitInput.View()))
		currentRow = append(currentRow, creditStyle.Render(row.creditInput.View()))

		result.WriteString(ervm.renderRow(currentRow) + "\n")
	}

	ervm.viewport.SetContent(result.String())
	ervm.scrollViewport()
}

func (ervm *rowsViewManager) renderRow(values []string) string {
	if len(values) != len(ervm.colWidths) {
		panic("you absolute dingus")
	}

	newStyle := lipgloss.NewStyle()

	var result strings.Builder
	for i := range values {
		style := newStyle.Width(ervm.colWidths[i])
		if i != len(values)-1 {
			style = style.MarginRight(2)
		}

		result.WriteString(style.Render(values[i]))
	}

	return result.String()
}

func (ervm *rowsViewManager) scrollViewport() {
	activeRow, _ := ervm.getActiveCoords()

	if activeRow >= ervm.viewport.YOffset+ervm.viewport.Height {
		ervm.viewport.ScrollDown(activeRow - ervm.viewport.YOffset - ervm.viewport.Height + 1)
	}

	if activeRow < ervm.viewport.YOffset {
		ervm.viewport.ScrollUp(ervm.viewport.YOffset - activeRow)
	}
}

// Converts a slice of EntryRow "forms" to a slice of EntryRow
func (ervm *rowsViewManager) compileRows() ([]database.EntryRow, error) {
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

func (uv *entryUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingEntry}
	}
}

// Returns preceeded/exceeded if the move would make the active input go "out of bounds"
func (ervm *rowsViewManager) switchFocus(direction meta.Sequence) (preceeded, exceeded bool) {
	oldRow, oldCol := ervm.getActiveCoords()

	switch direction {
	case meta.PREVIOUS:
		if oldRow == 0 && oldCol == 0 {
			ervm.rows[0].dateInput.Blur()
			ervm.isActive = false
			return true, false
		}

		ervm.setActiveCoords(oldRow, oldCol-1)

	case meta.NEXT:
		if oldRow == ervm.numRows()-1 && oldCol == ervm.numInputsPerRow()-1 {
			ervm.rows[oldRow].creditInput.Blur()
			ervm.isActive = false
			return false, true
		}

		ervm.setActiveCoords(oldRow, oldCol+1)

	default:
		panic(fmt.Sprintf("unexpected meta.Sequence: %#v", direction))
	}

	return false, false
}

func (ervm *rowsViewManager) calculateCurrentTotal() (database.CurrencyValue, error) {
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

func (ervm *rowsViewManager) numRows() int {
	return len(ervm.rows)
}

func (ervm *rowsViewManager) numInputs() int {
	return ervm.numRows() * ervm.numInputsPerRow()
}

func (ervm *rowsViewManager) numInputsPerRow() int {
	return 6
}

func (ervm *rowsViewManager) getActiveCoords() (row, col int) {
	inputsPerRow := ervm.numInputsPerRow()
	return ervm.activeInput / inputsPerRow, ervm.activeInput % inputsPerRow
}

func (ervm *rowsViewManager) focus(direction meta.Sequence) {
	ervm.isActive = true
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
func (ervm *rowsViewManager) setActiveCoords(newRow, newCol int) {
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
func decompileRows(rows []database.EntryRow, colWidths []int) ([]*rowsCreator, error) {
	result := make([]*rowsCreator, len(rows))

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

		formRow := newRowCreator(&row.Date, colWidths)

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

func (ervm *rowsViewManager) deleteRow() (*rowsViewManager, tea.Cmd) {
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

func (ervm *rowsViewManager) addRow(after bool) (*rowsViewManager, tea.Cmd) {
	activeRow, _ := ervm.getActiveCoords()

	var newRow *rowsCreator

	// If the row that the new-row-creation was triggered from had a valid date,
	// prefill it in the new row. Otherwise, just leave new row empty
	prefillDate, parseErr := database.ToDate(ervm.rows[activeRow].dateInput.Value())
	if parseErr == nil {
		newRow = newRowCreator(&prefillDate, ervm.colWidths)
	} else {
		newRow = newRowCreator(nil, ervm.colWidths)
	}

	newRows := make([]*rowsCreator, 0, ervm.numRows()+1)

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

type entryMutateView interface {
	View

	getJournalInput() *itempicker.Model
	getNotesInput() *textarea.Model
	getManager() *rowsViewManager

	getActiveInput() *int

	getColours() meta.AppColours

	title() string
}

func entriesMutateViewUpdate(view entryMutateView, message tea.Msg) (View, tea.Cmd) {
	activeInput := view.getActiveInput()
	journalInput := view.getJournalInput()
	notesInput := view.getNotesInput()
	entryRowsManager := view.getManager()

	switch message := message.(type) {
	case meta.SwitchFocusMsg:
		if *activeInput == ENTRIESNOTESINPUT {
			notesInput.Blur()
		}

		if *activeInput != ENTRIESROWINPUT {
			switch message.Direction {
			case meta.PREVIOUS:
				*activeInput--

				if *activeInput < 0 {
					*activeInput += 3
				}

			case meta.NEXT:
				*activeInput++

				*activeInput %= 3
			}

			// If it changed to entryrow input
			if *activeInput == ENTRIESROWINPUT {
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

		if *activeInput == ENTRIESNOTESINPUT {
			notesInput.Focus()
		}

		return view, nil

	case meta.NavigateMsg:
		if *activeInput != ENTRIESROWINPUT {
			return view, meta.MessageCmd(errors.New("hjkl navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.Update(message)
		*entryRowsManager = *manager

		return view, cmd

	case meta.JumpHorizontalMsg:
		if *activeInput != ENTRIESROWINPUT {
			return view, meta.MessageCmd(errors.New("$/_ navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.Update(message)
		*entryRowsManager = *manager

		return view, cmd

	case meta.JumpVerticalMsg:
		if *activeInput != ENTRIESROWINPUT {
			return view, meta.MessageCmd(errors.New("'gg'/'G' navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.Update(message)
		*entryRowsManager = *manager

		return view, cmd

	case tea.WindowSizeMsg:
		// TODO

		return view, nil

	case tea.KeyMsg:
		var cmd tea.Cmd
		switch *activeInput {
		case ENTRIESJOURNALINPUT:
			*journalInput, cmd = journalInput.Update(message)
		case ENTRIESNOTESINPUT:
			*notesInput, cmd = notesInput.Update(message)
		case ENTRIESROWINPUT:
			var manager *rowsViewManager
			manager, cmd = entryRowsManager.Update(message)
			*entryRowsManager = *manager

		default:
			panic(fmt.Sprintf("Updating create view but active input was %d", *activeInput))
		}

		return view, cmd

	case DeleteEntryRowMsg:
		if *activeInput != ENTRIESROWINPUT {
			return view, meta.MessageCmd(errors.New("no entry row highlighted while trying to delete one"))
		}

		manager, cmd := entryRowsManager.deleteRow()
		*entryRowsManager = *manager

		return view, cmd

	case CreateEntryRowMsg:
		if *activeInput != ENTRIESROWINPUT {
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
		slog.Warn("Unexpected tea.Msg", "message", fmt.Sprintf("%#v", message))
		return view, nil
	}
}

func entriesMutateViewView(view entryMutateView) string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(view.getColours().Background).Padding(0, 1)
	result.WriteString(titleStyle.Render(view.title()))

	result.WriteString("\n\n")

	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Align(lipgloss.Left)
	highlightStyle := sectionStyle.Foreground(view.getColours().Foreground)

	inputs := []viewable{view.getJournalInput(), view.getNotesInput(), view.getManager()}
	names := []string{"Journal", "Notes", ""}

	styles := slices.Repeat([]lipgloss.Style{sectionStyle}, len(names))
	styles[*view.getActiveInput()] = highlightStyle

	// +2 for padding
	maxNameColWidth := len(slices.MaxFunc(names, func(name string, other string) int {
		return cmp.Compare(len(name), len(other))
	})) + 2

	for i := range names {
		if names[i] == "" {
			result.WriteString(sectionStyle.Render(inputs[i].View()))
		} else {
			result.WriteString(lipgloss.JoinHorizontal(
				lipgloss.Top,
				sectionStyle.Width(maxNameColWidth).Render(names[i]),
				" ",
				styles[i].Render(inputs[i].View()),
			))
		}

		result.WriteString("\n")
	}

	return lipgloss.NewStyle().MarginLeft(2).Render(result.String())
}

func entriesMutateViewMotionSet() meta.MotionSet {
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

func entriesMutateViewCommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

type entryDeleteView struct {
	modelId int // only for retrieving the model itself initially
	model   database.Entry

	rows    []*database.EntryRow
	journal *database.Journal

	colours meta.AppColours
}

func NewEntryDeleteView(modelId int) *entryDeleteView {
	return &entryDeleteView{
		modelId: modelId,

		colours: meta.ENTRIESCOLOURS,
	}
}

func (dv *entryDeleteView) Init() tea.Cmd {
	// Can't load journal yet, we only know journal ID when entry is loaded
	entryCmd := database.MakeLoadEntriesDetailCmd(dv.modelId)
	rowsCmd := database.MakeSelectEntryRowsCmd(dv.modelId)

	return tea.Batch(entryCmd, rowsCmd)
}

func (dv *entryDeleteView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYMODEL:
			dv.model = message.Data.(database.Entry)

			journalIndex := slices.IndexFunc(database.AvailableJournals, func(j database.Journal) bool {
				return j.Id == dv.model.Journal
			})
			if journalIndex == -1 {
				return dv, meta.MessageCmd(fmt.Errorf("couldn't find journal %d in cache", dv.model.Journal))
			}

			// Don't reference original journal directly to ensure cache is never mutated
			journal := database.AvailableJournals[journalIndex]
			dv.journal = &journal

		case meta.ENTRYROWMODEL:
			rows := message.Data.([]database.EntryRow)

			dv.rows = make([]*database.EntryRow, len(rows))
			for i, row := range rows {
				dv.rows[i] = &row
			}
		}

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

func (dv *entryDeleteView) View() string {
	return genericDeleteViewView(dv)
}

func (dv *entryDeleteView) AllowsInsertMode() bool {
	return false
}

func (dv *entryDeleteView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRYMODEL:    {},
		meta.ENTRYROWMODEL: {},
	}
}

func (dv *entryDeleteView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *entryDeleteView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (dv *entryDeleteView) Reload() View {
	return NewEntryDeleteView(dv.modelId)
}

func (dv *entryDeleteView) title() string {
	return fmt.Sprintf("Delete entry %s", dv.model.String())
}

func (dv *entryDeleteView) inputValues() []string {
	var result []string

	if dv.journal == nil {
		result = append(result, strconv.Itoa(dv.model.Journal))
	} else {
		result = append(result, dv.journal.Name)
	}

	result = append(result, dv.model.Notes.Collapse())

	result = append(result, fmt.Sprintf("%d", len(dv.rows)))

	result = append(result, database.CalculateSize(dv.rows).String())

	return result
}

func (dv *entryDeleteView) inputNames() []string {
	return []string{"Journal", "Notes", "# rows", "Entry size"}
}

func (dv *entryDeleteView) getColours() meta.AppColours {
	return dv.colours
}

func (dv *entryDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
