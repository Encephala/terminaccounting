package modals

import (
	"fmt"
	"testing"

	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/tat"
	"terminaccounting/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextModal_Rendering(t *testing.T) {
	tm := NewTextModal("first line", "second line", "third line")
	tw := tat.NewTestWrapperSpecific(Modal(tm))

	tw.AssertViewContains(t, "first line")
	tw.AssertViewContains(t, "second line")
	tw.AssertViewContains(t, "third line")
}

func TestTextModal_Scroll(t *testing.T) {
	// Need more lines than viewport height (40) so scrolling is meaningful
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i)
	}

	tm := NewTextModal(lines...)
	tw := tat.NewTestWrapperSpecific(Modal(tm))

	tw.Send(meta.NavigateMsg{Direction: meta.DOWN})
	assert.Equal(t, 1, tm.viewport.YOffset, "scrolling down should increase YOffset")

	tw.Send(meta.NavigateMsg{Direction: meta.UP})
	assert.Equal(t, 0, tm.viewport.YOffset, "scrolling up should restore YOffset")
}

var testCSVHeaders = []string{"Date", "Name", "Account", "Counterparty", "Code", "Direction", "Amount", "Type", "Description"}
var testCSVData = [][]string{
	{"20240101", "My Account", "ACC001", "ACC002", "GT", "Credit", "100,00", "Transfer", "plain description"},
	{"20240115", "My Account", "ACC001", "ACC003", "GT", "Debit", "50,25", "Transfer", "another description"},
}

// setupBankImporter creates a bankImporter with pre-loaded CSV data, bypassing the zenity
// file picker that Init() would otherwise open.
func setupBankImporter(t *testing.T) *bankImporter {
	t.Helper()

	bi := NewBankImporter()
	bi.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	bi.fileLoaded = true
	bi.headers = testCSVHeaders
	bi.data = testCSVData
	bi.colWidths = bi.calculateColWidths(100)

	return bi
}

func TestBankImporter_Rendering(t *testing.T) {
	tat.SetupTestEnv(t)
	bi := setupBankImporter(t)

	rendered := bi.View()

	assert.Contains(t, rendered, "File format")
	assert.Contains(t, rendered, "Journal")
	assert.Contains(t, rendered, "Bank ledger")
	assert.Contains(t, rendered, "ING")
	assert.Contains(t, rendered, ":write")
}

func TestBankImporter_FocusNavigation(t *testing.T) {
	tat.SetupTestEnv(t)
	bi := setupBankImporter(t)

	assert.Equal(t, 0, bi.activeInput, "initial active input should be parser picker (0)")

	bi.Update(meta.SwitchFocusMsg{Direction: meta.NEXT})
	assert.Equal(t, 1, bi.activeInput, "after NEXT should be journal picker (1)")

	bi.Update(meta.SwitchFocusMsg{Direction: meta.NEXT})
	assert.Equal(t, 2, bi.activeInput, "after NEXT should be bank ledger picker (2)")

	bi.Update(meta.SwitchFocusMsg{Direction: meta.NEXT})
	assert.Equal(t, 3, bi.activeInput, "after NEXT should be preview (3)")

	bi.Update(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
	assert.Equal(t, 2, bi.activeInput, "PREVIOUS from preview should return to bank ledger (2)")

	bi.activeInput = 0
	bi.Update(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
	assert.Equal(t, 3, bi.activeInput, "PREVIOUS from parser should wrap to preview (3)")

	bi.Update(meta.SwitchFocusMsg{Direction: meta.NEXT})
	assert.Equal(t, 0, bi.activeInput, "NEXT from preview should wrap to parser picker (0)")
}

func TestBankImporter_Commit(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	// An accounts ledger is required by the importer's commit handler
	accountsLedger := database.Ledger{Name: "Accounts Ledger", Type: database.ASSETLEDGER, IsAccounts: true}
	_, err := accountsLedger.Insert(DB)
	require.NoError(t, err)

	bankLedger := database.Ledger{Name: "Bank Ledger", Type: database.ASSETLEDGER}
	bankLedgerId, err := bankLedger.Insert(DB)
	require.NoError(t, err)
	bankLedger.Id = bankLedgerId

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)
	journal.Id = journalId

	bi := setupBankImporter(t)
	require.NoError(t, bi.journalPicker.SetValue(journal))
	require.NoError(t, bi.bankLedgerPicker.SetValue(bankLedger))

	_, cmd := bi.Update(meta.CommitMsg{})
	require.NotNil(t, cmd)

	switchMsg, ok := cmd().(meta.SwitchAppViewMsg)
	require.True(t, ok, "commit should return a SwitchAppViewMsg")

	assert.Equal(t, meta.CREATEVIEWTYPE, switchMsg.ViewType)
	require.NotNil(t, switchMsg.App)
	assert.Equal(t, meta.ENTRIESAPP, *switchMsg.App)

	prefillData, ok := switchMsg.Data.(view.EntryPrefillData)
	require.True(t, ok, "SwitchAppViewMsg.Data should be EntryPrefillData")
	assert.Equal(t, journalId, prefillData.Journal.Id)
	// 2 CSV rows × 2 ledger rows each = 4 entry rows
	assert.Len(t, prefillData.Rows, 4)
	assert.NotEmpty(t, prefillData.Notes)
}

func TestBankImporter_Commit_NoJournal(t *testing.T) {
	tat.SetupTestEnv(t)
	bi := setupBankImporter(t)

	_, cmd := bi.Update(meta.CommitMsg{})
	require.NotNil(t, cmd)

	err, ok := cmd().(error)
	require.True(t, ok)
	assert.EqualError(t, err, "no journal selected (none available)")
}

func TestBankImporter_Commit_NoBankLedger(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	// Insert journal before setupBankImporter so journalPicker is populated for SetValue.
	// Insert accountsLedger after so bankLedgerPicker starts empty (Value() returns nil) to test the error occurring
	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)
	journal.Id = journalId

	bi := setupBankImporter(t)

	accountsLedger := database.Ledger{Name: "Accounts Ledger", Type: database.ASSETLEDGER, IsAccounts: true}
	_, err = accountsLedger.Insert(DB)
	require.NoError(t, err)

	require.NoError(t, bi.journalPicker.SetValue(journal))

	_, cmd := bi.Update(meta.CommitMsg{})
	require.NotNil(t, cmd)

	err, ok := cmd().(error)
	require.True(t, ok)
	assert.EqualError(t, err, "no bank ledger selected (none available)")
}

func TestBankImporter_Navigate(t *testing.T) {
	tat.SetupTestEnv(t)
	bi := setupBankImporter(t)
	bi.activeInput = 3
	bi.View() // populate viewport content so TotalLineCount() is correct

	bi.Update(meta.NavigateMsg{Direction: meta.DOWN})
	assert.Equal(t, 1, bi.activeRow)

	bi.Update(meta.NavigateMsg{Direction: meta.UP})
	assert.Equal(t, 0, bi.activeRow)

	bi.Update(meta.NavigateMsg{Direction: meta.UP})
	assert.Equal(t, 0, bi.activeRow)

	bi.activeRow = 1
	bi.Update(meta.NavigateMsg{Direction: meta.DOWN})
	assert.Equal(t, 1, bi.activeRow)
}

func TestBankImporter_Navigate_RequiresPreviewFocus(t *testing.T) {
	tat.SetupTestEnv(t)
	bi := setupBankImporter(t)

	_, cmd := bi.Update(meta.NavigateMsg{Direction: meta.DOWN})
	require.NotNil(t, cmd)

	err, ok := cmd().(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "jk navigation only works within preview table")
}

func TestBankImporter_Navigate_ScrollsViewport(t *testing.T) {
	tat.SetupTestEnv(t)

	// Need more rows than preview viewport height (Height:40 -> preview.Height = 31)
	manyRows := make([][]string, 40)
	for i := range manyRows {
		manyRows[i] = []string{
			fmt.Sprintf("202401%02d", i%28+1),
			"My Account", "ACC001", "ACC002", "GT",
			"Credit", "420,00", "Transfer",
			fmt.Sprintf("transaction %d", i),
		}
	}

	bi := NewBankImporter()
	bi.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	bi.fileLoaded = true
	bi.headers = testCSVHeaders
	bi.data = manyRows
	bi.colWidths = bi.calculateColWidths(100)
	bi.activeInput = 3
	bi.View() // populate viewport content

	for i := 0; i < bi.preview.Height+1; i++ {
		bi.Update(meta.NavigateMsg{Direction: meta.DOWN})
	}
	assert.Greater(t, bi.preview.YOffset, 0)

	for i := 0; i < bi.preview.Height+1; i++ {
		bi.Update(meta.NavigateMsg{Direction: meta.UP})
	}
	assert.Equal(t, 0, bi.preview.YOffset)
}

// Kinda silly test, but helpful to have this one fail to remind me to fix tests, should ING CSV format ever change
func TestIngParser_UsedColumns(t *testing.T) {
	ip := ingParser{}
	assert.Equal(t, []int{0, 3, 6, 8}, ip.usedColumns())
}

func TestIngParser_ParseDescription(t *testing.T) {
	ip := ingParser{}

	t.Run("with Description field", func(t *testing.T) {
		// ING description format: "... Description: <text> IBAN: ..."
		description := "Naam: John Description: test payment IBAN: NL01ABCD0000000001"
		result := ip.parseDescription(description)
		require.NotNil(t, result)
		assert.Equal(t, "test payment", *result)
	})

	t.Run("without Description field", func(t *testing.T) {
		result := ip.parseDescription("plain description without structured fields")
		assert.Nil(t, result)
	})
}

func TestIngParser_CompileRows_Credit(t *testing.T) {
	tat.SetupTestEnv(t)

	ip := ingParser{}
	data := [][]string{
		{"20240101", "", "", "NL01ABCD0000000001", "", "Credit", "100,00", "", "plain description"},
	}

	rows, err := ip.compileRows(data, 1, 2)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	accountRow, bankRow := rows[0], rows[1]

	assert.Equal(t, database.CurrencyValue(10000), accountRow.Value, "credit should be positive on account ledger")
	assert.Equal(t, database.CurrencyValue(-10000), bankRow.Value, "credit should be negative on bank ledger")
	assert.Equal(t, 1, accountRow.Ledger)
	assert.Equal(t, 2, bankRow.Ledger)
	assert.Equal(t, "plain description", accountRow.Description)
	assert.Equal(t, "plain description", bankRow.Description)
}

func TestIngParser_CompileRows_Debit(t *testing.T) {
	tat.SetupTestEnv(t)

	ip := ingParser{}
	data := [][]string{
		{"20240115", "", "", "NL01ABCD0000000001", "", "Debit", "50,25", "", "another description"},
	}

	rows, err := ip.compileRows(data, 1, 2)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, database.CurrencyValue(-5025), rows[0].Value, "debit should be negative on account ledger")
	assert.Equal(t, database.CurrencyValue(5025), rows[1].Value, "debit should be positive on bank ledger")
}

func TestIngParser_CompileRows_ParsedDescription(t *testing.T) {
	tat.SetupTestEnv(t)

	ip := ingParser{}
	data := [][]string{
		{"20240101", "", "", "NL01ABCD0000000001", "", "Credit", "10,00", "", "Naam: X Description: structured payment IBAN: NL01ABCD0000000001"},
	}

	rows, err := ip.compileRows(data, 1, 2)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "structured payment", rows[0].Description)
}

func TestIngParser_CompileRows_AccountMatching(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	counterpartyIBAN := "NL01ABCD0123456789"
	account := database.Account{
		Name:        "Counterparty Corp",
		Type:        database.DEBTOR,
		BankNumbers: meta.Notes{counterpartyIBAN},
	}
	accountId, err := account.Insert(DB)
	require.NoError(t, err)

	ip := ingParser{}
	data := [][]string{
		{"20240101", "", "", counterpartyIBAN, "", "Credit", "75,00", "", "payment"},
	}

	rows, err := ip.compileRows(data, 1, 2)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	require.NotNil(t, rows[0].Account, "account row should have matched account ID")
	assert.Equal(t, accountId, *rows[0].Account)
	require.NotNil(t, rows[1].Account, "bank row should have matched account ID")
	assert.Equal(t, accountId, *rows[1].Account)
}

func TestIngParser_CompileRows_NoAccountMatch(t *testing.T) {
	tat.SetupTestEnv(t)

	ip := ingParser{}
	data := [][]string{
		{"20240101", "", "", "NL01UNKNOWN0000001", "", "Credit", "10,00", "", "payment"},
	}

	rows, err := ip.compileRows(data, 1, 2)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Nil(t, rows[0].Account, "unmatched IBAN should leave Account as nil")
	assert.Nil(t, rows[1].Account)
}

func TestIngParser_CompileRows_MultipleRows(t *testing.T) {
	tat.SetupTestEnv(t)

	ip := ingParser{}
	data := [][]string{
		{"20240101", "", "", "NL01ABCD0000000001", "", "Credit", "100,00", "", "first"},
		{"20240102", "", "", "NL01ABCD0000000002", "", "Debit", "30,00", "", "second"},
	}

	rows, err := ip.compileRows(data, 1, 2)
	require.NoError(t, err)
	assert.Len(t, rows, 4, "2 CSV rows should produce 4 entry rows (2 per CSV row)")
}
