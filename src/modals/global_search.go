package modals

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
	"github.com/jmoiron/sqlx"
	"github.com/sahilm/fuzzy"
)

type searchItem struct {
	model any
}

func (item *searchItem) filterValue() string {
	var result strings.Builder

	availableLedgers := database.AvailableLedgers()
	availableAccounts := database.AvailableAccounts()
	availableJournals := database.AvailableJournals()

	switch model := item.model.(type) {
	case database.Ledger:
		result.WriteString(fmt.Sprintf("%d", model.Id))
		result.WriteString(model.Name)
		result.WriteString(fmt.Sprintf("%s", model.Type))
		result.WriteString(model.Notes.Collapse())

		if model.IsAccounts {
			result.WriteString("isaccounts")
		}

	case database.Account:
		result.WriteString(fmt.Sprintf("%d", model.Id))
		result.WriteString(model.Name)
		result.WriteString(fmt.Sprintf("%s", model.Type))
		result.WriteString(model.BankNumbers.Collapse())
		result.WriteString(model.Notes.Collapse())

	case database.Journal:
		result.WriteString(fmt.Sprintf("%d", model.Id))
		result.WriteString(model.Name)
		result.WriteString(fmt.Sprintf("%s", model.Type))
		result.WriteString(model.Notes.Collapse())

	case database.Entry:
		result.WriteString(fmt.Sprintf("%d", model.Id))

		journal := availableJournals[slices.IndexFunc(availableJournals, func(other database.Journal) bool {
			return other.Id == model.Journal
		})]
		result.WriteString(journal.Name)

		result.WriteString(model.Notes.Collapse())

	case database.EntryRow:
		result.WriteString(fmt.Sprintf("%d", model.Id))
		result.WriteString(fmt.Sprintf("%d", model.Entry))
		result.WriteString(model.Date.String())

		ledger := availableLedgers[slices.IndexFunc(availableLedgers, func(other database.Ledger) bool {
			return other.Id == model.Ledger
		})]
		result.WriteString(fmt.Sprintf("%s", ledger.Name))

		if model.Account == nil {
			result.WriteString("none")
		} else {
			account := availableAccounts[slices.IndexFunc(availableAccounts, func(other database.Account) bool {
				return other.Id == *model.Account
			})]
			result.WriteString(fmt.Sprintf("%s", account.Name))
		}

		result.WriteString(model.Description)
		if model.Document != nil {
			result.WriteString(*model.Document)
		}

		result.WriteString(model.Value.String())

		if model.Reconciled {
			result.WriteString("reconciled")
		}

	default:
		panic(fmt.Sprintf("unexpected searchItem type: %#v", model))
	}

	return result.String()
}

func (item *searchItem) render(isActive bool) string {
	var color lipgloss.Color
	var text string

	switch model := item.model.(type) {
	case database.Ledger:
		color = meta.LEDGERSCOLOUR
		text = model.String()

	case database.Account:
		color = meta.ACCOUNTSCOLOUR
		text = model.String()

	case database.Journal:
		color = meta.JOURNALSCOLOUR
		text = model.String()

	case database.Entry:
		color = meta.ENTRIESCOLOUR
		text = "Entry " + model.String()

	case database.EntryRow:
		color = meta.ENTRIESCOLOUR
		// TODO
		text = "Entryrow"

	default:
		panic(fmt.Sprintf("unexpected searchItem type: %#v", model))
	}

	if isActive {
		color = lipgloss.Color("7")
	}

	return lipgloss.NewStyle().Foreground(color).Render(text)
}

type globalSearchModal struct {
	DB *sqlx.DB

	width, height int

	searchQuery string

	viewport  viewport.Model
	activeRow int

	items      []searchItem
	shownItems []searchItem
}

func newGlobalSearchModal(DB *sqlx.DB) *globalSearchModal {
	return &globalSearchModal{
		DB: DB,

		viewport: viewport.New(0, 0),
	}
}

func (gsm *globalSearchModal) Init() tea.Cmd {
	return makeSelectAllDataCmd(gsm.DB)
}

func (gsm *globalSearchModal) Update(message tea.Msg) (Modal, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		gsm.width = message.Width
		gsm.height = message.Height

		gsm.viewport.Width = message.Width
		gsm.viewport.Height = message.Height - 2

		return gsm, nil

	case meta.NavigateMsg:
		if message.Direction == meta.DOWN {
			gsm.activeRow = min(gsm.activeRow+1, len(gsm.shownItems)-1)
		} else {
			gsm.activeRow = max(gsm.activeRow-1, 0)
		}

		gsm.scrollViewport()

		return gsm, nil

	case meta.JumpVerticalMsg:
		if message.Down {
			gsm.activeRow = len(gsm.shownItems) - 1
		} else {
			gsm.activeRow = 0
		}

		gsm.scrollViewport()

		return gsm, nil

	case meta.UpdateSearchMsg:
		var cmd tea.Cmd
		gsm.searchQuery = message.Query

		if message.Query != "" {
			gsm.activeRow = 0
		}

		gsm.updateShownItems()

		return gsm, cmd

	case meta.DataLoadedMsg:
		data := message.Data.([]any)
		gsm.items = make([]searchItem, len(data))

		for i, item := range data {
			gsm.items[i] = item.(searchItem)
		}

		gsm.updateShownItems()

		return gsm, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (gsm *globalSearchModal) View() string {
	var result strings.Builder

	result.WriteString("Query: " + gsm.searchQuery)
	result.WriteString("\n\n")

	gsm.updateViewportContent()

	if len(gsm.shownItems) > gsm.viewport.Height {
		gsm.viewport.Width = gsm.width - 2

		scrollState := float64(gsm.viewport.YOffset) / float64(len(gsm.shownItems)-gsm.viewport.Height)

		result.WriteString(lipgloss.JoinHorizontal(lipgloss.Position(scrollState), gsm.viewport.View(), " ", "█"))
	} else {
		gsm.viewport.Width = gsm.width
		result.WriteString(gsm.viewport.View())
	}

	return result.String()
}

func (gsm *globalSearchModal) updateShownItems() {
	if gsm.searchQuery == "" {
		gsm.shownItems = gsm.items
		return
	}

	filterValues := make([]string, len(gsm.items))
	for i, item := range gsm.items {
		filterValues[i] = item.filterValue()
	}

	matches := fuzzy.Find(gsm.searchQuery, filterValues)

	gsm.shownItems = make([]searchItem, len(matches))
	for i, match := range matches {
		gsm.shownItems[i] = gsm.items[match.Index]
	}
}

func (gsm *globalSearchModal) scrollViewport() {
	if gsm.activeRow >= gsm.viewport.YOffset+gsm.viewport.Height {
		gsm.viewport.ScrollDown(gsm.activeRow - gsm.viewport.YOffset - gsm.viewport.Height + 1)
	}

	if gsm.activeRow < gsm.viewport.YOffset {
		gsm.viewport.ScrollUp(gsm.viewport.YOffset - gsm.activeRow)
	}
}

func (gsm *globalSearchModal) updateViewportContent() {
	var result []string
	for i, item := range gsm.shownItems {
		result = append(result, item.render(gsm.activeRow == i))
	}

	gsm.viewport.SetContent(strings.Join(result, "\n"))
}

func (gsm *globalSearchModal) AllowsInsertMode() bool {
	return false
}

func (gsm *globalSearchModal) AllowsSearchMode() bool {
	return true
}

func (gsm *globalSearchModal) MotionSet() meta.Trie[tea.Msg] {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	result.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	// Closures to the rescue once more
	var gotoDetailViewCmd tea.Cmd
	gotoDetailViewCmd = func() tea.Msg {
		if len(gsm.shownItems) == 0 {
			return errors.New("no items shown to go to detail view of")
		}

		var appType *meta.AppType
		var data any

		activeItem := gsm.shownItems[gsm.activeRow]
		switch model := activeItem.model.(type) {
		case database.Ledger:
			tmp := meta.LEDGERSAPP
			appType = &tmp
			data = model

		case database.Account:
			tmp := meta.ACCOUNTSAPP
			appType = &tmp
			data = model

		case database.Journal:
			tmp := meta.JOURNALSAPP
			appType = &tmp
			data = model

		case database.Entry:
			tmp := meta.ENTRIESAPP
			appType = &tmp
			data = model

		case database.EntryRow:
			tmp := meta.ENTRIESAPP
			appType = &tmp

			entry, err := database.SelectEntry(gsm.DB, model.Entry)
			if err != nil {
				return fmt.Errorf("Failed to go to entry detail view: %s", err)
			}

			data = entry

		default:
			panic(fmt.Sprintf("unexpected type: %#v", model))
		}

		return meta.SwitchAppViewMsg{
			App:      appType,
			ViewType: meta.DETAILVIEWTYPE,
			Data:     data,
		}
	}
	result.Insert(meta.Motion{"g", "d"}, gotoDetailViewCmd)

	result.Insert(meta.Motion{"g", "g"}, meta.JumpVerticalMsg{Down: false})
	result.Insert(meta.Motion{"G"}, meta.JumpVerticalMsg{Down: true})

	return result
}

func (gsm *globalSearchModal) CommandSet() meta.Trie[tea.Msg] {
	return meta.Trie[tea.Msg]{}
}

func (gsm *globalSearchModal) Reload() Modal {
	return newGlobalSearchModal(gsm.DB)
}

func makeSelectAllDataCmd(DB *sqlx.DB) tea.Cmd {
	return func() tea.Msg {
		var result []any

		ledgers := database.AvailableLedgers()

		accounts := database.AvailableAccounts()

		journals := database.AvailableJournals()

		entries, err := database.SelectEntries(DB)
		if err != nil {
			return meta.MessageCmd(err)
		}

		entryRows, err := database.SelectRows(DB)
		if err != nil {
			return meta.MessageCmd(err)
		}

		result = slices.Concat(
			toAnySlice(ledgers),
			toAnySlice(accounts),
			toAnySlice(journals),
			toAnySlice(entries),
			toAnySlice(entryRows),
		)

		return meta.DataLoadedMsg{Data: result}
	}
}

func toAnySlice[T any](slice []T) []any {
	result := make([]any, len(slice))

	for i, value := range slice {
		result[i] = searchItem{
			model: value,
		}
	}

	return result
}
