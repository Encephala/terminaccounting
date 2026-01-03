package tat

import (
	"fmt"
	"log/slog"
	"sync"
	"terminaccounting/database"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func SetupTestEnv(t *testing.T) *sqlx.DB {
	t.Helper()

	slog.SetLogLoggerLevel(slog.LevelWarn)

	// Ensure that each connection *within each test* uses the same in-memory database.
	id := fmt.Sprintf("%s-%s", t.Name(), time.Now().String())
	DB := sqlx.MustConnect("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", id))

	err := database.InitSchemas(DB)

	require.Nil(t, err)

	return DB
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

func (tw *TestWrapper) Lock() {
	tw.asyncModel.mutex.Lock()
}

func (tw *TestWrapper) Unlock() {
	tw.asyncModel.mutex.Unlock()
}

func (tw *TestWrapper) Wait(condition func(tea.Model) bool) tea.Model {
	tw.t.Helper()

	ticker := time.NewTicker(time.Millisecond * 1)
	timeout := time.After(time.Second * 1)

	checkConditionFn := func() (tea.Model, bool) {
		tw.asyncModel.mutex.Lock()
		defer tw.asyncModel.mutex.Unlock()

		if condition(tw.asyncModel.model) {
			return tw.asyncModel.model, true
		}

		return nil, false
	}

	for {
		select {
		case <-timeout:
			tw.t.Fatalf("test timed out")
			return nil

		case <-ticker.C:
			if model, ok := checkConditionFn(); ok {
				return model
			}
		}
	}
}

// Waits for the provided condition to be met, then quits the program, returning the final program state
func (tw *TestWrapper) WaitQuit(condition func(tea.Model) bool) tea.Model {
	tw.t.Helper()

	tw.Wait(condition)

	return tw.Quit()
}

func (tw *TestWrapper) Quit() tea.Model {
	tw.t.Helper()

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

	am.lastMessage = message

	var cmd tea.Cmd
	am.model, cmd = am.model.Update(message)

	return am, cmd
}

func (am *asyncModel) View() string {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	return am.model.View()
}

func KeyMsg(input string) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(input),
	}
}
