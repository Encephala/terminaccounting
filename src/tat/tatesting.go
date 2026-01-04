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
	t.Cleanup(func() { DB.Close() })

	err := database.InitSchemas(DB)

	require.Nil(t, err)

	return DB
}

func KeyMsg(input string) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(input),
	}
}

type TestWrapper struct {
	t *testing.T

	model tea.Model

	runtimeInfo *runtimeInfo
}

func NewTestWrapper(t *testing.T, model tea.Model) *TestWrapper {
	return &TestWrapper{
		t: t,

		model: model,
	}
}

type runtimeInfo struct {
	program           *tea.Program
	runtimeErrChannel chan error

	mutex *sync.Mutex
}

func (tw *TestWrapper) RunAsync() *TestWrapper {
	if tw.runtimeInfo != nil {
		panic("tried to make already async TestWrapper async again, that seems wrong and dumb")
	}

	asyncModel := &asyncModel{model: tw.model}
	tw.model = asyncModel

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
	// how do I verify that the model has processed the init messages? I see no easy way
	time.Sleep(time.Millisecond * 1)

	tw.runtimeInfo = &runtimeInfo{
		program:           program,
		runtimeErrChannel: runtimeErrChannel,

		mutex: &asyncModel.mutex,
	}

	tw.t.Cleanup(func() { slog.Warn("cleaning up fam"); tw.Quit() })

	return tw
}

func (tw *TestWrapper) Send(messages ...tea.Msg) *[]tea.Msg {
	if tw.runtimeInfo == nil {
		var returnedMessages []tea.Msg

		for _, message := range messages {
			// Simulate runtime
			var cmd tea.Cmd
			tw.model, cmd = tw.model.Update(message)

			// TODO: Does this handle Batch?
			for cmd != nil {
				msg := cmd()
				returnedMessages = append(returnedMessages, msg)
				tw.model, cmd = tw.model.Update(msg)
			}
		}

		return &returnedMessages
	}

	for _, message := range messages {
		tw.runtimeInfo.program.Send(message)
	}

	return nil
}

// Small convenience methods
func (tw *TestWrapper) lock() {
	if tw.runtimeInfo == nil {
		panic("locking but not running async")
	}

	tw.runtimeInfo.mutex.Lock()
}
func (tw *TestWrapper) unlock() {
	if tw.runtimeInfo == nil {
		panic("unlocking but not running async")
	}

	tw.runtimeInfo.mutex.Unlock()
}

func (tw *TestWrapper) Wait(condition func(tea.Model) bool) tea.Model {
	tw.t.Helper()

	if tw.runtimeInfo == nil {
		panic("TestWrapper is running synchronously, nothing to wait for")
	}

	ticker := time.NewTicker(time.Millisecond * 1)
	timeout := time.After(time.Second * 1)

	checkConditionFn := func() (tea.Model, bool) {
		tw.runtimeInfo.mutex.Lock()
		defer tw.runtimeInfo.mutex.Unlock()

		innerModel := tw.model.(*asyncModel).model
		if condition(innerModel) {
			return innerModel, true
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

		case err := <-tw.runtimeInfo.runtimeErrChannel:
			tw.t.Fatalf("program runtime error: %q", err)
		}
	}
}

func (tw *TestWrapper) Quit() tea.Model {
	tw.t.Helper()

	tw.Send(tea.QuitMsg{})

	return tw.model
}

func (tw *TestWrapper) AssertEqual(actualGetter func(tea.Model) any, expected any) {
	tw.t.Helper()

	if tw.runtimeInfo == nil {
		value := actualGetter(tw.model)
		assert.Equal(tw.t, expected, value)

		return
	}

	tw.lock()
	defer tw.unlock()

	innerModel := tw.model.(*asyncModel).model
	value := actualGetter(innerModel)
	assert.Equal(tw.t, expected, value)
}

func (tw *TestWrapper) AssertViewContains(expected string) {
	tw.t.Helper()

	if tw.runtimeInfo == nil {
		assert.Contains(tw.t, tw.model.View(), expected)
		return
	}

	tw.lock()
	defer tw.unlock()

	innerModel := tw.model.(*asyncModel).model
	assert.Contains(tw.t, innerModel.View(), expected)
}

type asyncModel struct {
	model tea.Model

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
	am.model, cmd = am.model.Update(message)

	return am, cmd
}

func (am *asyncModel) View() string {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	return am.model.View()
}
