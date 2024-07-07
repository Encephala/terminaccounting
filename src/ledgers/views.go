package ledgers

import (
	"fmt"
	"log/slog"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

type listView struct {
	db *sqlx.DB

	list list.Model
}

func newListView(db *sqlx.DB) tea.Model {
	return &listView{
		db: db,
	}
}

func (lv *listView) Init() tea.Cmd {
	ledgers, err := SelectAll(lv.db)
	if err != nil {
		slog.Error(fmt.Sprintf("Got error while selecting: %v", err))
		return func() tea.Msg { return err }
	}

	items := []list.Item{}
	for _, ledger := range ledgers {
		items = append(items, ledger)
	}

	lv.list = list.New(items, list.NewDefaultDelegate(), 20, 10)
	// lv.list.Title = "All the epic ledgers"

	return nil
}

func (lv *listView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	lv.list, cmd = lv.list.Update(message)

	return lv, cmd
}

func (lv *listView) View() string {
	slog.Info(fmt.Sprintf("Rendering ledger list view, %d items", len(lv.list.Items())))

	return lv.list.View()
}

func (l Ledger) FilterValue() string {
	return "filtervalue " + l.Name
}

func (l Ledger) Title() string {
	return "title " + l.Name
}

func (l Ledger) Description() string {
	return "description " + l.Name
}
