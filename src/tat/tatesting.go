package tat

import (
	"fmt"
	"log/slog"
	"strings"
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
	id := fmt.Sprintf("%s-%s", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().String())
	DB := sqlx.MustConnect("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", id))
	t.Cleanup(func() { DB.Close() })

	err := database.InitSchemas(DB)

	require.Nil(t, err)

	return DB
}

func makeKeyMsg(input string) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(input),
	}
}

type TestWrapper struct {
	model tea.Model

	runtimeInfo *runtimeInfo

	lastCmdResults []tea.Msg
}

func NewTestWrapper(model tea.Model) *TestWrapper {
	return &TestWrapper{
		model: model,
	}
}

type runtimeInfo struct {
	program           *tea.Program
	runtimeErrChannel chan error

	mutex *sync.Mutex
}

func (tw *TestWrapper) RunAsync(t *testing.T) *TestWrapper {
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
	time.Sleep(time.Millisecond * 10)

	tw.runtimeInfo = &runtimeInfo{
		program:           program,
		runtimeErrChannel: runtimeErrChannel,

		mutex: &asyncModel.mutex,
	}

	t.Cleanup(func() { tw.Send(tea.QuitMsg{}) })

	return tw
}

func (tw *TestWrapper) Send(messages ...tea.Msg) *TestWrapper {
	if tw.runtimeInfo == nil {
		tw.lastCmdResults = make([]tea.Msg, 0)

		for _, message := range messages {
			var cmd tea.Cmd
			tw.model, cmd = tw.model.Update(message)

			tw.handleCmd(cmd)
		}

		return tw
	}

	for _, message := range messages {
		tw.runtimeInfo.program.Send(message)
	}

	return tw
}

func (tw *TestWrapper) SendText(input string) *TestWrapper {
	var messages []tea.Msg

	for _, char := range input {
		messages = append(messages, makeKeyMsg(string(char)))
	}

	tw.Send(messages...)

	return tw
}

// Simulate runtime handling cmds returned by an Update
func (tw *TestWrapper) handleCmd(cmd tea.Cmd) {
	var queue []tea.Cmd
	queue = append(queue, cmd)

	for len(queue) > 0 {
		cmd := queue[0]
		queue = queue[1:]

		if cmd == nil {
			continue
		}

		switch message := cmd().(type) {
		case tea.BatchMsg:
			queue = append(queue, message...)

		// Nil message, e.g. meta.MessageCmd(err) but err was nil
		case nil:
			continue

		default:
			tw.lastCmdResults = append(tw.lastCmdResults, message)
			tw.model, cmd = tw.model.Update(message)

			queue = append(queue, cmd)
		}
	}
}

func (tw *TestWrapper) GetLastCmdResults() []tea.Msg {
	// tw.lastMessages is only (can only be) maintained if running asynchronously
	if tw.runtimeInfo != nil {
		panic("TestWrapper is running asynchronously, can't get last messages")
	}

	return tw.lastCmdResults
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

func (tw *TestWrapper) Wait(t *testing.T, condition func(tea.Model) bool) {
	t.Helper()

	if tw.runtimeInfo == nil {
		panic("TestWrapper is running synchronously, nothing to wait for")
	}

	ticker := time.NewTicker(time.Millisecond * 1)
	timeout := time.After(time.Millisecond * 100)

	for {
		select {
		case <-ticker.C:
			if tw.checkCondition(condition) {
				return
			}

		case <-timeout:
			tw.Send(tea.QuitMsg{})
			t.Fatalf("test timed out")

		case err := <-tw.runtimeInfo.runtimeErrChannel:
			t.Fatalf("program runtime error: %q", err)
		}
	}
}

func (tw *TestWrapper) checkCondition(condition func(tea.Model) bool) bool {
	tw.lock()
	defer tw.unlock()

	innerModel := tw.model.(*asyncModel).model

	return condition(innerModel)
}

func (tw *TestWrapper) AssertEqual(t *testing.T, actualGetter func(tea.Model) any, expected any) {
	t.Helper()

	if tw.runtimeInfo == nil {
		value := actualGetter(tw.model)
		assert.Equal(t, expected, value)

		return
	}

	tw.lock()
	defer tw.unlock()

	innerModel := tw.model.(*asyncModel).model
	value := actualGetter(innerModel)
	assert.Equal(t, expected, value)
}

func (tw *TestWrapper) AssertViewContains(t *testing.T, expected string) {
	t.Helper()

	if tw.runtimeInfo == nil {
		assert.Contains(t, tw.model.View(), expected)
		return
	}

	tw.lock()
	defer tw.unlock()

	innerModel := tw.model.(*asyncModel).model
	assert.Contains(t, innerModel.View(), expected)
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
