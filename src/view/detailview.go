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
)

type genericDetailView interface {
	View

	title() string

	getViewer() *entryRowViewer

	canReconcile() bool
	getShowReconciledRows() *bool
	getShowReconciledTotal() *bool

	getColours() meta.AppColours
}

func genericDetailViewUpdate(gdv genericDetailView, message tea.Msg) (View, tea.Cmd) {
	viewer := gdv.getViewer()

	switch message := message.(type) {
	case tea.WindowSizeMsg:
		newViewer, cmd := viewer.Update(tea.WindowSizeMsg{
			// -4 for padding on each side
			Width: message.Width - 4,
			// -4 for the title and table header (header is not considered for table width)
			// -6 to for the total rows their vertical margin
			Height: message.Height - 4 - 6,
		})
		*viewer = *newViewer

		viewer.calculateColumnWidths(gdv.canReconcile())

		return gdv, cmd

	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYROWMODEL:
			data := message.Data.([]database.EntryRow)

			viewer.originalRows = make([]database.EntryRow, len(data))
			viewer.rows = make([]*database.EntryRow, len(data))

			for i, row := range message.Data.([]database.EntryRow) {
				viewer.originalRows[i] = row
				viewer.rows[i] = &row
			}

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}

		viewer.updateViewRows(gdv.canReconcile(), *gdv.getShowReconciledRows())

		return gdv, nil

	case meta.NavigateMsg:
		newViewer, cmd := viewer.Update(message)
		*viewer = *newViewer

		return gdv, cmd

	case meta.ToggleShowReconciledMsg:
		if !gdv.canReconcile() {
			return gdv, meta.MessageCmd(errors.New("reconciling is disabled, no reconciled rows to show"))
		}

		showReconciledRows := gdv.getShowReconciledRows()
		*showReconciledRows = !*showReconciledRows

		viewer.updateViewRows(gdv.canReconcile(), *showReconciledRows)

		if activeEntryRow := viewer.activeEntryRow(); activeEntryRow != nil {
			viewer.activateEntryRow(activeEntryRow)
		}

		return gdv, nil

	case meta.ReconcileMsg:
		if !gdv.canReconcile() {
			return gdv, meta.MessageCmd(errors.New("reconciling is disabled"))
		}

		activeEntryRow := viewer.activeEntryRow()

		if activeEntryRow == nil {
			return gdv, meta.MessageCmd(errors.New("there are no rows to reconcile"))
		}

		activeEntryRow.Reconciled = !activeEntryRow.Reconciled
		if !*gdv.getShowReconciledRows() && viewer.activeRow == len(viewer.viewRows)-1 {
			viewer.activeRow = max(0, viewer.activeRow-1)
		}
		viewer.updateViewRows(gdv.canReconcile(), *gdv.getShowReconciledRows())

		if viewer.rowsAreChanged() {
			*gdv.getShowReconciledTotal() = true
		} else {
			*gdv.getShowReconciledTotal() = false
		}

		return gdv, nil

	case meta.CommitMsg:
		if !viewer.rowsAreChanged() {
			return gdv, meta.MessageCmd(meta.NotificationMessageMsg{Message: "there are no changes in reconciliation to commit"})
		}

		total := database.CalculateTotal(viewer.getReconciledRows())
		if total != 0 {
			return gdv, meta.MessageCmd(fmt.Errorf("total of reconciled rows not 0 but %s", total))
		}

		changed, err := database.SetReconciled(viewer.rows)
		if err != nil {
			return gdv, meta.MessageCmd(err)
		}

		notification := meta.NotificationMessageMsg{Message: fmt.Sprintf("set reconciled status, updated %d rows", changed)}

		// Reset dv.originalRows
		for i, row := range viewer.rows {
			viewer.originalRows[i] = *row
			*gdv.getShowReconciledTotal() = false
		}

		return gdv, meta.MessageCmd(notification)

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func genericDetailViewView(gdv genericDetailView) string {
	colours := gdv.getColours()

	var result strings.Builder

	marginLeftStyle := lipgloss.NewStyle().MarginLeft(2)

	titleStyle := marginLeftStyle.Background(colours.Background).Padding(0, 1)
	result.WriteString(titleStyle.Render(gdv.title()))

	result.WriteString("\n")

	if gdv.canReconcile() {
		result.WriteString(marginLeftStyle.Render(fmt.Sprintf("Showing reconciled rows: %s", renderBoolean(*gdv.getShowReconciledRows()))))
	} else {
		result.WriteString(lipgloss.NewStyle().Italic(true).MarginLeft(2).Render("Reconciling disabled"))
	}

	result.WriteString("\n\n")

	result.WriteString(marginLeftStyle.Render(gdv.getViewer().View(gdv.canReconcile())))

	result.WriteString("\n\n")

	result.WriteString(marginLeftStyle.Render(fmt.Sprintf("Total: %s", database.CalculateTotal(gdv.getViewer().rows))))

	result.WriteString("\n")

	if gdv.canReconcile() && *gdv.getShowReconciledTotal() {
		totalReconciled := database.CalculateTotal(gdv.getViewer().getReconciledRows())
		var totalReconciledRendered string

		if totalReconciled == 0 {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
			totalReconciledRendered = style.Render(fmt.Sprintf("%s", totalReconciled))
		} else {
			totalReconciledRendered = fmt.Sprintf("%s", totalReconciled)
		}

		result.WriteString(marginLeftStyle.Render(fmt.Sprintf("Reconciled total: %s", totalReconciledRendered)))

		result.WriteString("\n")
	}

	result.WriteString(marginLeftStyle.Render(fmt.Sprintf("Size: %s", database.CalculateSize(gdv.getViewer().rows))))

	result.WriteString("\n")

	result.WriteString(marginLeftStyle.Render(fmt.Sprintf("# rows: %d", len(gdv.getViewer().rows))))

	return result.String()
}

func genericDetailViewMotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"s", "r"}, meta.ToggleShowReconciledMsg{}) // [S]how [R]econciled
	normalMotions.Insert(meta.Motion{"enter"}, meta.ReconcileMsg{})

	return meta.MotionSet{Normal: normalMotions}
}

func genericDetailViewCommandSet() meta.CommandSet {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(result)
}

func makeGoToEntryDetailViewCmd(activeEntryRow *database.EntryRow) tea.Cmd {
	return func() tea.Msg {
		if activeEntryRow == nil {
			return meta.MessageCmd(errors.New("there is no row to view details of"))
		}

		entryId := activeEntryRow.Entry

		// Do the database query for the entry here, because it is a command and thus asynchronous
		entry, err := database.SelectEntry(entryId)

		if err != nil {
			return meta.MessageCmd(err)
		}

		// Stupid go not allowing to reference a const
		targetApp := meta.ENTRIESAPP
		return meta.SwitchViewMsg{App: &targetApp, ViewType: meta.DETAILVIEWTYPE, Data: entry}
	}
}

type entryRowViewer struct {
	width, height int

	viewport viewport.Model
	viewRows [][]string

	rows         []*database.EntryRow
	originalRows []database.EntryRow

	activeRow      int
	highlightStyle lipgloss.Style

	headers   []string
	colWidths []int
}

func newEntryRowViewer(colours meta.AppColours) *entryRowViewer {
	return &entryRowViewer{
		viewport: viewport.New(0, 0),

		highlightStyle: lipgloss.NewStyle().Foreground(colours.Foreground),

		headers: []string{"Date", "Ledger", "Account", "Description", "Debit", "Credit", "Reconciled"},
	}
}

func (erv *entryRowViewer) Update(message tea.Msg) (*entryRowViewer, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		erv.width = message.Width
		erv.height = message.Height

		erv.viewport.Width = message.Width
		erv.viewport.Height = message.Height

		return erv, nil

	case meta.NavigateMsg:
		switch message.Direction {
		case meta.DOWN:
			if erv.activeRow != erv.viewport.TotalLineCount()-1 {
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

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (erv *entryRowViewer) View(canReconcile bool) string {
	var result strings.Builder

	erv.setViewportContent(canReconcile)

	if canReconcile {
		result.WriteString(erv.renderRow(erv.headers, false))
	} else {
		result.WriteString(erv.renderRow(erv.headers[:len(erv.headers)-1], false))
	}

	result.WriteString("\n")

	result.WriteString(erv.viewport.View())

	return result.String()
}

func (erv *entryRowViewer) updateViewRows(canReconcile, showReconciledRows bool) {
	var shownRows []*database.EntryRow
	if showReconciledRows {
		shownRows = erv.rows
	} else {
		shownRows = erv.getUnreconciledRows()
	}

	var viewRows [][]string
	for _, row := range shownRows {
		var viewRow []string
		viewRow = append(viewRow, row.Date.String())

		var ledger, account string

		availableLedgerIndex := slices.IndexFunc(database.AvailableLedgers, func(ledger database.Ledger) bool {
			return ledger.Id == row.Ledger
		})
		if availableLedgerIndex != -1 {
			ledger = database.AvailableLedgers[availableLedgerIndex].String()
		} else {
			ledger = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Italic(true).Render("error")
		}

		if row.Account == nil {
			account = lipgloss.NewStyle().Italic(true).Render("None")
		} else {
			availableAccountIndex := slices.IndexFunc(database.AvailableAccounts, func(account database.Account) bool {
				return account.Id == *row.Account
			})

			if availableAccountIndex != -1 {
				account = database.AvailableAccounts[availableAccountIndex].String()
			} else {
				ledger = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Italic(true).Render("error")
			}
		}

		viewRow = append(viewRow, ledger)
		viewRow = append(viewRow, account)
		viewRow = append(viewRow, row.Description)
		if row.Value > 0 {
			viewRow = append(viewRow, row.Value.String())
			viewRow = append(viewRow, "")
		} else {
			viewRow = append(viewRow, "")
			viewRow = append(viewRow, (-row.Value).String())
		}

		if canReconcile {
			viewRow = append(viewRow, lipgloss.NewStyle().AlignHorizontal(lipgloss.Center).Render(renderBoolean(row.Reconciled)))
		}

		viewRows = append(viewRows, viewRow)
	}

	erv.viewRows = viewRows
}

func (erv *entryRowViewer) scrollViewport() {
	if erv.activeRow >= erv.viewport.YOffset+erv.height {
		erv.viewport.ScrollDown(erv.activeRow - erv.viewport.YOffset - erv.height + 1)
	}

	if erv.activeRow < erv.viewport.YOffset {
		erv.viewport.ScrollUp(erv.viewport.YOffset - erv.activeRow)
	}
}

func (erv *entryRowViewer) activeEntryRow() *database.EntryRow {
	if len(erv.viewRows) == 0 {
		return nil
	}

	return erv.rows[erv.activeRow]
}

func (erv *entryRowViewer) calculateColumnWidths(canReconcile bool) {
	dateWidth := 10 // This is simply the width of a date field
	reconciledWidth := len("Reconciled")

	var colWidths []int
	if canReconcile {
		// -4 because of left/right margin
		remainingWidth := erv.width - dateWidth - reconciledWidth - 4
		descriptionWidth := remainingWidth / 3

		// -12 because of the 2-wide padding between columns, 6x
		// /4 because there are four other columns
		othersWidth := (remainingWidth - descriptionWidth - 12) / 4
		colWidths = []int{dateWidth, othersWidth, othersWidth, descriptionWidth, othersWidth, othersWidth, reconciledWidth}
	} else {
		// -4 because of left/right margin
		remainingWidth := erv.width - dateWidth - 4
		descriptionWidth := remainingWidth / 3

		// -10 because of the 2-wide padding between columns, 5x
		othersWidth := (remainingWidth - descriptionWidth - 10) / 4
		colWidths = []int{dateWidth, othersWidth, othersWidth, descriptionWidth, othersWidth, othersWidth}
	}

	erv.colWidths = colWidths
}

func (erv *entryRowViewer) setViewportContent(canReconcile bool) {
	var content []string

	for i, row := range erv.viewRows {
		doHighlight := i == erv.activeRow
		if canReconcile {
			content = append(content, erv.renderRow(row, doHighlight))
		} else {
			content = append(content, erv.renderRow(row, doHighlight))
		}
	}

	erv.viewport.SetContent(strings.Join(content, "\n"))
}

func (erv *entryRowViewer) renderRow(values []string, highlight bool) string {
	if len(erv.colWidths) != len(values) {
		panic("You absolute dingus")
	}

	var result strings.Builder

	for i := range values {
		style := lipgloss.NewStyle()
		if highlight {
			style = erv.highlightStyle
		}

		style = style.Width(erv.colWidths[i])

		if i != len(values)-1 {
			style = style.MarginRight(2)
		}

		result.WriteString(style.Render(ansi.Truncate(values[i], erv.colWidths[i], "…")))
	}

	return result.String()
}

func (erv *entryRowViewer) activateEntryRow(entryRow *database.EntryRow) {
	index := slices.IndexFunc(erv.rows, func(row *database.EntryRow) bool { return row == entryRow })

	if index == -1 {
		// This means that the row that was highlighted is reconciled and is now hidden, all good
		return
	}

	erv.activeRow = index
	erv.scrollViewport()
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
