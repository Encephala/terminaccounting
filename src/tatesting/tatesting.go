package tatesting

import (
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

func SetupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db := sqlx.MustConnect("sqlite3", ":memory:")

	return db
}

type TestWrapper struct {
	t *testing.T

	program    *tea.Program
	asyncModel *asyncModel

	doneChannel chan tea.Model
}

func InitIntegrationTest(t *testing.T, model tea.Model) TestWrapper {
	asyncModel := &asyncModel{model: model}

	program := tea.NewProgram(asyncModel, tea.WithoutRenderer())

	doneChannel := make(chan tea.Model)
	go func() {
		final, _ := program.Run()
		doneChannel <- final
	}()

	return TestWrapper{
		t: t,

		program:    program,
		asyncModel: asyncModel,

		doneChannel: doneChannel,
	}
}

func (tw *TestWrapper) Send(messages ...tea.Msg) {
	for _, message := range messages {
		tw.program.Send(message)
	}
}

func (tw *TestWrapper) LastMessge() tea.Msg {
	return tw.asyncModel.lastMessage
}

func (tw *TestWrapper) Wait(condition func(tea.Model) bool) tea.Model {
	ticker := time.NewTicker(time.Millisecond * 50)
	timeout := time.After(time.Second * 2)

	for {
		select {
		case <-timeout:
			tw.t.Fatalf("test timed out")
			return nil

		case <-ticker.C:
			currentModel := tw.asyncModel.getCurrentModel()
			if !(condition(currentModel)) {
				continue
			}

			return currentModel
		}
	}
}

// Waits for the provided condition to be met, then quits the program, returning the final program state
func (tw *TestWrapper) WaitQuit(condition func(tea.Model) bool) tea.Model {
	tw.Wait(condition)

	tw.program.Quit()

	finalModel := <-tw.doneChannel
	return finalModel.(*asyncModel).model
}

type asyncModel struct {
	model       tea.Model
	lastMessage tea.Msg

	mutex sync.Mutex
}

func (am *asyncModel) Init() tea.Cmd {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	return am.model.Init()
}

func (am *asyncModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	var cmd tea.Cmd
	am.lastMessage = message
	am.model, cmd = am.model.Update(message)

	return am, cmd
}

func (am *asyncModel) View() string {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	return am.model.View()
}

func (am *asyncModel) getCurrentModel() tea.Model {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	return am.model
}

func KeyMsg(input string) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(input),
	}
}
