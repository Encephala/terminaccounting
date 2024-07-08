package ledgers

import (
	"fmt"
	"log/slog"
	"strings"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

type listView struct {
	db  *sqlx.DB
	app meta.App

	list list.Model
}

func newListView(db *sqlx.DB, app meta.App) tea.Model {
	return &listView{
		db:  db,
		app: app,
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

	listDelegator := list.NewDefaultDelegate()
	listDelegator.Styles.SelectedDesc = listDelegator.Styles.SelectedDesc.Foreground(lv.app.BackgroundColour()).
		BorderForeground(lv.app.BackgroundColour())
	listDelegator.Styles.SelectedTitle = listDelegator.Styles.SelectedTitle.Foreground(lv.app.AccentColour()).
		BorderForeground(lv.app.AccentColour())

	list := list.New(items, listDelegator, 20, 16)
	list.Title = "Ledgers"
	list.Styles.Title = lv.list.Styles.Title.Background(lv.app.AccentColour())
	list.SetShowHelp(false)

	lv.list = list

	return nil
}

func (lv *listView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	lv.list, cmd = lv.list.Update(message)

	return lv, cmd
}

func (lv *listView) View() string {
	return lv.list.View()
}

func (l Ledger) FilterValue() string {
	result := l.Name
	result += strings.Join(l.Notes, ";")
	return result
}

func (l Ledger) Title() string {
	return l.Name
}

func (l Ledger) Description() string {
	return l.Name
}
