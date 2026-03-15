package view

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/jmoiron/sqlx"
)

type genericDetailView interface {
	View

	title() string
	metadata() metadata
	getWidth() int

	getDB() *sqlx.DB

	getCanReconcile() bool

	getViewer() *entryRowViewer
}

func genericDetailViewUpdate(gdv genericDetailView, message tea.Msg) (View, tea.Cmd) {
	viewer := gdv.getViewer()

	switch message := message.(type) {
	case tea.WindowSizeMsg:
		newViewer, cmd := viewer.Update(tea.WindowSizeMsg{
			Width: message.Width,
			// -1 for the title, reconcilableness info
			Height: message.Height - 1,
		})
		*viewer = *newViewer

		viewer.calculateColumnWidths()

		return gdv, cmd

	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYROWMODEL:
			data := message.Data.([]database.EntryRow)

			viewer.originalRows = make([]database.EntryRow, len(data))
			viewer.rows = make([]*database.EntryRow, len(data))

			for i, row := range data {
				viewer.originalRows[i] = row
				viewer.rows[i] = &row
			}

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}

		viewer.updateViewRows()

		return gdv, nil

	case meta.NavigateMsg:
		newViewer, cmd := viewer.Update(message)
		*viewer = *newViewer

		return gdv, cmd

	case meta.JumpVerticalMsg:
		if message.Down {
			viewer.activeRow = len(viewer.shownRows) - 1
		} else {
			viewer.activeRow = 0
		}

		viewer.scrollViewport()

		return gdv, nil

	case meta.UpdateSearchMsg:
		newViewer, cmd := viewer.Update(message)
		*viewer = *newViewer

		return gdv, cmd

	case meta.ToggleShowReconciledMsg:
		oldShownRows := make([]*database.EntryRow, len(viewer.shownRows))
		oldActiveIndex := viewer.activeRow
		copy(oldShownRows, viewer.shownRows)

		viewer.showReconciled = !viewer.showReconciled
		viewer.updateViewRows()

		if len(viewer.shownRows) == 0 || len(oldShownRows) == 0 {
			return gdv, nil
		}

		// If there was previously an active row, set a new row active that is reasonably close to the old one
		// Find (or try to) the closest row equal to or before the previous active row that is still being shown
		for i := oldActiveIndex; i >= 0; i-- {
			found := viewer.setActiveRow(oldShownRows[i])

			if found {
				return gdv, nil
			}
		}

		// If no closest row before found (i.e. active row was one of the first rows and is now hidden)
		for i := oldActiveIndex + 1; i < len(viewer.shownRows); i++ {
			found := viewer.setActiveRow(oldShownRows[i])

			if found {
				return gdv, nil
			}
		}

		panic("this never happens due to the above check of len(viewer.shownRows) == 0")

	case meta.ReconcileMsg:
		if !gdv.getCanReconcile() {
			return gdv, meta.MessageCmd(errors.New("reconciling is disabled in this view"))
		}

		activeEntryRow := viewer.getActiveRow()

		if activeEntryRow == nil {
			return gdv, meta.MessageCmd(errors.New("there are no rows to reconcile"))
		}

		activeEntryRow.Reconciled = !activeEntryRow.Reconciled
		// If the last row was just reconciled and it is now hidden, set activeRow to the new last row
		if !viewer.showReconciled && viewer.activeRow == len(viewer.viewRows)-1 {
			viewer.activeRow = max(0, viewer.activeRow-1)
		}

		viewer.updateViewRows()

		return gdv, nil

	case meta.CommitMsg:
		if !viewer.rowsAreChanged() {
			return gdv, meta.MessageCmd(meta.NotificationMessageMsg{Message: "there are no changes in reconciliation to commit"})
		}

		total := database.CalculateTotal(viewer.getReconciledRows())
		if total != 0 {
			return gdv, meta.MessageCmd(fmt.Errorf("total of reconciled rows not 0 but %s", total))
		}

		changed, err := database.SetReconciled(gdv.getDB(), viewer.rows)
		if err != nil {
			return gdv, meta.MessageCmd(err)
		}

		notification := meta.NotificationMessageMsg{Message: fmt.Sprintf("set reconciled status, updated %d rows", changed)}

		// Reset dv.originalRows
		for i, row := range viewer.rows {
			viewer.originalRows[i] = *row
		}

		return gdv, meta.MessageCmd(notification)

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func genericDetailViewView(gdv genericDetailView) string {
	var result strings.Builder

	result.WriteString(renderHeader(gdv.title(), gdv.metadata(), gdv.getWidth()))
	result.WriteString("\n")

	result.WriteString(fmt.Sprintf("Showing reconciled rows: %s", renderBoolean(gdv.getViewer().showReconciled)))
	result.WriteString("\n")

	result.WriteString(fmt.Sprintf("Reconciling enabled: %s", renderBoolean(gdv.getCanReconcile())))
	result.WriteString("\n\n")

	result.WriteString(gdv.getViewer().View())

	return result.String()
}

func genericDetailViewMotionSet() meta.Trie[tea.Msg] {
	var motions meta.Trie[tea.Msg]

	motions.Insert(meta.Motion{"/"}, meta.SwitchModeMsg{InputMode: meta.COMMANDMODE, Data: true}) // true -> yes search mode

	motions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	motions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	motions.Insert(meta.Motion{"g", "g"}, meta.JumpVerticalMsg{Down: false})
	motions.Insert(meta.Motion{"G"}, meta.JumpVerticalMsg{Down: true})

	motions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	motions.Insert(meta.Motion{"s", "r"}, meta.ToggleShowReconciledMsg{}) // [S]how [R]econciled
	motions.Insert(meta.Motion{"enter"}, meta.ReconcileMsg{})

	return motions
}

func genericDetailViewCommandSet() meta.Trie[tea.Msg] {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return result
}

func makeGoToEntryDetailViewCmd(DB *sqlx.DB, activeEntryRow *database.EntryRow) tea.Cmd {
	return func() tea.Msg {
		if activeEntryRow == nil {
			return meta.MessageCmd(errors.New("there is no row to view details of"))
		}

		entryId := activeEntryRow.Entry

		// Do the database query for the entry here, because it is a command and thus asynchronous
		entry, err := database.SelectEntry(DB, entryId)

		if err != nil {
			return meta.MessageCmd(err)
		}

		// Stupid go not allowing to reference a const
		targetApp := meta.ENTRIESAPP
		return meta.SwitchAppViewMsg{App: &targetApp, ViewType: meta.DETAILVIEWTYPE, Data: entry}
	}
}

type entryRowViewer struct {
	width, height   int
	highlightColour lipgloss.Color

	viewport viewport.Model

	// All the rows in this viewer
	rows []*database.EntryRow
	// The rows being shown in the view right now
	shownRows []*database.EntryRow
	// THe shownRows, rendered to strings slices
	viewRows [][]string
	// The original rows in the viewer (for comparing reconciled-state)
	originalRows []database.EntryRow

	activeRow int

	// The query to filter shownRows by
	// If nil, no filter
	filter *string

	showReconciled bool

	headers   []string
	colWidths []int
}

func newEntryRowViewer(colour lipgloss.Color) *entryRowViewer {
	result := &entryRowViewer{
		highlightColour: colour,

		viewport: viewport.New(0, 0),

		headers: []string{"Date", "Ledger", "Account", "Description", "Debit", "Credit", "Reconciled"},
	}

	return result
}

func (erv *entryRowViewer) Update(message tea.Msg) (*entryRowViewer, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		erv.width = message.Width
		erv.height = message.Height

		erv.viewport.Width = message.Width
		// -8 for the total rows and their vertical margin
		erv.viewport.Height = message.Height - 8

		return erv, nil

	case meta.NavigateMsg:
		switch message.Direction {
		case meta.DOWN:
			if erv.activeRow != len(erv.shownRows)-1 {
				erv.activeRow++
			}

		case meta.UP:
			if erv.activeRow != 0 {
				erv.activeRow--
			}

		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", message.Direction))
		}

		erv.scrollViewport()

		return erv, nil

	case meta.UpdateSearchMsg:
		if message.Query == "" {
			erv.filter = nil
		} else {
			erv.filter = &message.Query
		}

		erv.updateViewRows()

		return erv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (erv *entryRowViewer) View() string {
	var result strings.Builder

	erv.setViewportContent()

	result.WriteString(erv.renderRow(erv.headers, true, false))

	result.WriteString("\n")

	result.WriteString(erv.viewport.View())

	result.WriteString("\n\n")

	result.WriteString(fmt.Sprintf("Total: %s", database.CalculateTotal(erv.rows)))

	result.WriteString("\n")

	if erv.rowsAreChanged() {
		totalReconciled := database.CalculateTotal(erv.getReconciledRows())

		var totalReconciledRendered string
		if totalReconciled == 0 {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
			totalReconciledRendered = style.Render(fmt.Sprintf("%s", totalReconciled))
		} else {
			totalReconciledRendered = fmt.Sprintf("%s", totalReconciled)
		}

		result.WriteString(fmt.Sprintf("Reconciled total: %s", totalReconciledRendered))

		result.WriteString("\n")
	}

	result.WriteString(fmt.Sprintf("Size: %s", database.CalculateSize(erv.rows)))

	result.WriteString("\n")

	result.WriteString(fmt.Sprintf("# rows: %d", len(erv.rows)))

	return result.String()
}

func (erv *entryRowViewer) calculateColumnWidths() {
	dateWidth := 10 // This is simply the width of a date field
	reconciledWidth := len("Reconciled")

	var colWidths []int
	remainingWidth := erv.width - dateWidth - reconciledWidth
	descriptionWidth := remainingWidth / 3

	// -12 because of the 2-wide padding between columns, 6x
	// /4 because there are four other columns
	othersWidth := (remainingWidth - descriptionWidth - 12) / 4
	colWidths = []int{dateWidth, othersWidth, othersWidth, descriptionWidth, othersWidth, othersWidth, reconciledWidth}

	erv.colWidths = colWidths
}

// Takes the rows, and depending on state, updates the shownRows and viewRows based off of them
func (erv *entryRowViewer) updateViewRows() {
	if erv.showReconciled {
		erv.shownRows = filterRows(erv.rows, erv.filter)
	} else {
		erv.shownRows = filterRows(erv.getUnreconciledRows(), erv.filter)
	}

	availableLedgers := database.AvailableLedgers()
	availableAccounts := database.AvailableAccounts()

	var viewRows [][]string
	for _, row := range erv.shownRows {
		var viewRow []string
		viewRow = append(viewRow, row.Date.String())

		ledger, account := getRowLedgerAndAccount(row, availableLedgers, availableAccounts)

		viewRow = append(viewRow, ledger.String(), account.String(), row.Description)

		if row.Value == 0 {
			panic(fmt.Sprintf("row %#v had zero debit and credit?", row))
		}
		if row.Value > 0 {
			viewRow = append(viewRow, row.Value.String(), "")
		} else {
			viewRow = append(viewRow, "", (-row.Value).String())
		}

		viewRow = append(viewRow, renderBoolean(row.Reconciled))

		viewRows = append(viewRows, viewRow)
	}

	erv.viewRows = viewRows
}

func getRowLedgerAndAccount(row *database.EntryRow,
	availableLedgers []database.Ledger,
	availableAccounts []database.Account) (database.Ledger, *database.Account) {
	var ledger database.Ledger
	availableLedgerIndex := slices.IndexFunc(availableLedgers, func(ledger database.Ledger) bool {
		return ledger.Id == row.Ledger
	})
	if availableLedgerIndex == -1 {
		panic(fmt.Sprintf("ledger for row %#v wasn't found in cache", row))
	}
	ledger = availableLedgers[availableLedgerIndex]

	var account *database.Account
	if row.Account == nil {
		account = nil
	} else {
		availableAccountIndex := slices.IndexFunc(availableAccounts, func(account database.Account) bool {
			return account.Id == *row.Account
		})

		if availableAccountIndex == -1 {
			panic(fmt.Sprintf("account for row %#v wasn't found in cache", row))
		}

		account = &availableAccounts[availableAccountIndex]
	}

	return ledger, account
}

func filterRows(rows []*database.EntryRow, filter *string) []*database.EntryRow {
	if filter == nil {
		return rows
	}

	availableLedgers := database.AvailableLedgers()
	availableAccounts := database.AvailableAccounts()

	var result []*database.EntryRow
	for _, row := range rows {
		ledger, account := getRowLedgerAndAccount(row, availableLedgers, availableAccounts)

		if strings.Contains(row.Date.String(), *filter) {
			result = append(result, row)
			continue
		}

		if strings.Contains(ledger.String(), *filter) {
			result = append(result, row)
			continue
		}

		if strings.Contains(account.String(), *filter) {
			result = append(result, row)
			continue
		}

		if strings.Contains(row.Description, *filter) {
			result = append(result, row)
			continue
		}

		if strings.Contains(row.Value.Abs().String(), *filter) {
			result = append(result, row)
			continue
		}
	}

	return result
}

func (erv *entryRowViewer) scrollViewport() {
	if erv.activeRow >= erv.viewport.YOffset+erv.viewport.Height {
		erv.viewport.ScrollDown(erv.activeRow - erv.viewport.YOffset - erv.viewport.Height + 1)
	}

	if erv.activeRow < erv.viewport.YOffset {
		erv.viewport.ScrollUp(erv.viewport.YOffset - erv.activeRow)
	}
}

func (erv *entryRowViewer) getActiveRow() *database.EntryRow {
	if len(erv.viewRows) == 0 {
		return nil
	}

	return erv.shownRows[erv.activeRow]
}

func (erv *entryRowViewer) setActiveRow(entryRow *database.EntryRow) (found bool) {
	index := slices.IndexFunc(erv.shownRows, func(row *database.EntryRow) bool { return row == entryRow })

	if index == -1 {
		// This means that the row that was highlighted is reconciled and is now hidden, all good
		return false
	}

	erv.activeRow = index
	erv.scrollViewport()

	return true
}

func (erv *entryRowViewer) setViewportContent() {
	var content []string

	for i, row := range erv.viewRows {
		doHighlight := i == erv.activeRow
		content = append(content, erv.renderRow(row, false, doHighlight))
	}

	erv.viewport.SetContent(strings.Join(content, "\n"))
}

func (erv *entryRowViewer) renderRow(values []string, isHeader, highlight bool) string {
	if len(erv.colWidths) != len(values) {
		panic("You absolute dingus")
	}

	var result strings.Builder

	for i := range values {
		style := lipgloss.NewStyle().Width(erv.colWidths[i])

		if highlight {
			style = style.Foreground(erv.highlightColour)
		}

		if !isHeader && i == len(values)-1 {
			style = style.AlignHorizontal(lipgloss.Center)
		}

		if i != len(values)-1 {
			style = style.MarginRight(2)
		}

		result.WriteString(style.Render(ansi.Truncate(values[i], erv.colWidths[i], "…")))
	}

	return result.String()
}

func (erv *entryRowViewer) getReconciledRows() []*database.EntryRow {
	var result []*database.EntryRow

	for _, row := range erv.rows {
		if row.Reconciled {
			result = append(result, row)
		}
	}

	return result
}

func (erv *entryRowViewer) getUnreconciledRows() []*database.EntryRow {
	var result []*database.EntryRow

	for _, row := range erv.rows {
		if !row.Reconciled {
			result = append(result, row)
		}
	}

	return result
}

func (erv *entryRowViewer) rowsAreChanged() bool {
	return !slices.EqualFunc(
		erv.rows,
		erv.originalRows,
		func(row *database.EntryRow, originalRow database.EntryRow) bool { return *row == originalRow },
	)
}
