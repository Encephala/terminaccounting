package modals

import (
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

func (sm *globalSearchModal) Init() tea.Cmd {
	return makeSelectAllDataCmd(sm.DB)
}

func (sm *globalSearchModal) Update(message tea.Msg) (Modal, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		sm.viewport.Width = message.Width
		sm.viewport.Height = message.Height - 2

		return sm, nil

	case meta.NavigateMsg:
		if message.Direction == meta.DOWN {
			sm.activeRow = min(sm.activeRow+1, len(sm.shownItems)-1)
		} else {
			sm.activeRow = max(sm.activeRow-1, 0)
		}

		return sm, nil

	case meta.UpdateSearchMsg:
		var cmd tea.Cmd
		sm.searchQuery = message.Query

		if message.Query != "" {
			sm.activeRow = 0
		}

		sm.updateShownItems()

		return sm, cmd

	case meta.GlobalSearchDataLoadedMsg:
		sm.items = make([]searchItem, len(message.Data))

		for i, item := range message.Data {
			sm.items[i] = item.(searchItem)
		}

		sm.updateShownItems()

		return sm, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (sm *globalSearchModal) View() string {
	var result strings.Builder

	result.WriteString("Query: " + sm.searchQuery)
	result.WriteString("\n\n")

	sm.updateViewportContent()

	result.WriteString(sm.viewport.View())

	return result.String()
}

func (sm *globalSearchModal) updateShownItems() {
	if sm.searchQuery == "" {
		sm.shownItems = sm.items
		return
	}

	filterValues := make([]string, len(sm.items))
	for i, item := range sm.items {
		filterValues[i] = item.filterValue()
	}

	matches := fuzzy.Find(sm.searchQuery, filterValues)

	sm.shownItems = make([]searchItem, len(matches))
	for i, match := range matches {
		sm.shownItems[i] = sm.items[match.Index]
	}
}

func (sm *globalSearchModal) updateViewportContent() {
	var result []string
	for i, item := range sm.shownItems {
		result = append(result, item.render(sm.activeRow == i))
	}

	sm.viewport.SetContent(strings.Join(result, "\n"))
}

func (sm *globalSearchModal) AllowsInsertMode() bool {
	return false
}

func (sm *globalSearchModal) AllowsSearchMode() bool {
	return true
}

func (sm *globalSearchModal) MotionSet() meta.Trie[tea.Msg] {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	result.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	// Closures to the rescue once more
	var gotoDetailViewCmd tea.Cmd
	gotoDetailViewCmd = func() tea.Msg {
		var appType *meta.AppType
		var data any

		// TODO: handle panic if zero items shown

		activeItem := sm.shownItems[sm.activeRow]
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

			entry, err := database.SelectEntry(sm.DB, model.Entry)
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

	// TODO: gg/G motion

	return result
}

func (sm *globalSearchModal) CommandSet() meta.Trie[tea.Msg] {
	return meta.Trie[tea.Msg]{}
}

func (sm *globalSearchModal) Reload() Modal {
	return newGlobalSearchModal(sm.DB)
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

		return meta.GlobalSearchDataLoadedMsg{Data: result}
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
