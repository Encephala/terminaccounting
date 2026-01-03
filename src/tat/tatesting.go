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
	"github.com/stretchr/testify/assert"
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

	runtimeErrChannel chan error
}

func InitIntegrationTest(t *testing.T, model tea.Model) TestWrapper {
	asyncModel := &asyncModel{model: model}

	program := tea.NewProgram(asyncModel, tea.WithoutRenderer())

	// Buffered channel to make sure no goroutines leak in weird situations,
	// though I don't believe that can ever happen
	runtimeErrChannel := make(chan error, 1)
	go func() {
		_, finalErr := program.Run()
		runtimeErrChannel <- finalErr
	}()

	// Give model some time to init
	// Utterly arbitrary amount of time. This should really be a TestWrapper.Wait call, but like
	// how do I verify that the model has processed the init messages?
	// I see no easy way
	time.Sleep(time.Millisecond * 1)

	return TestWrapper{
		t: t,

		program:    program,
		asyncModel: asyncModel,

		runtimeErrChannel: runtimeErrChannel,
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

// Small convenience methods
func (tw *TestWrapper) lock() {
	tw.asyncModel.mutex.Lock()
}
func (tw *TestWrapper) unlock() {
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
			tw.Quit()
			tw.t.Fatalf("test timed out")
			return nil

		case <-ticker.C:
			if model, ok := checkConditionFn(); ok {
				return model
			}

		case err := <-tw.runtimeErrChannel:
			tw.t.Fatalf("program runtime error: %q", err)
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

	tw.Send(tea.QuitMsg{})

	return tw.asyncModel.model
}

func (tw *TestWrapper) AssertEqual(actualGetter func(tea.Model) interface{}, expected interface{}) {
	tw.t.Helper()

	tw.lock()
	defer tw.unlock()

	value := actualGetter(tw.asyncModel.model)
	assert.Equal(tw.t, expected, value)
}

func (tw *TestWrapper) AssertViewContains(expected string) {
	tw.t.Helper()

	tw.lock()
	defer tw.unlock()

	assert.Contains(tw.t, tw.asyncModel.model.View(), expected)
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
