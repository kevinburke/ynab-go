package ynab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kevinburke/go-types"
)

var resp = []byte(`{"data": {
	"transactions": [{
		"account_id": "e0dc51c5-5136-4a3f-9019-84487d266cbb",
		"account_name": "Cash",
		"amount": -2000,
		"approved": true,
		"category_id": "97634123-0823-4c37-a2a6-8ec2bccb3c63",
		"category_name": "Dining Out",
		"cleared": "reconciled",
		"date": "2019-01-18",
		"deleted": false,
		"flag_color": "red",
		"id": "e0d8d32f-6c93-4b92-be48-c4590f3ed2a7",
		"import_id": null,
		"matched_transaction_id": null,
		"memo": "Ice Cream",
		"payee_id": "0fb63639-3dd8-435c-b17c-d50f8b7bbeb6",
		"payee_name": "Corner Store",
		"subtransactions": [],
		"transfer_account_id": null,
		"transfer_transaction_id": null
	}]
}}`)

func TestResponseParsing(t *testing.T) {
	txnList := new(TransactionListResponse)
	err := json.Unmarshal(resp, txnList)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(txnList.Data.Transactions); l != 1 {
		t.Errorf("expected txn list to have one item, got %d", l)
	}
	tx := txnList.Data.Transactions[0]
	if tx.Amount != -2000 {
		t.Errorf("bad amount")
	}
	if tx.CategoryName.String != "Dining Out" {
		t.Errorf("bad category name")
	}
	if tx.AccountName != "Cash" {
		t.Errorf("bad account name")
	}
}

func TestUserAgentHeader(t *testing.T) {
	// Create a test server that captures the User-Agent header
	var capturedUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"budgets": []}}`))
	}))
	defer server.Close()

	// Create a client with the test server URL
	client := NewClient("test-token")
	client.Base = server.URL

	// Make a request to trigger the User-Agent header
	resp, err := client.Budgets("budget-id").Categories(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("resp", resp)

	// Verify the User-Agent contains "ynab-go"
	if !strings.Contains(capturedUserAgent, "ynab-go") {
		t.Errorf("User-Agent header does not contain 'ynab-go': %s", capturedUserAgent)
	}

	// Verify it contains the version
	expectedVersion := "ynab-go/" + Version
	if !strings.Contains(capturedUserAgent, expectedVersion) {
		t.Errorf("User-Agent header does not contain expected version '%s': %s", expectedVersion, capturedUserAgent)
	}
}

func TestDateMarshalJSON(t *testing.T) {
	date := Date(time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC))

	data, err := json.Marshal(date)
	if err != nil {
		t.Fatal(err)
	}

	expected := `"2023-05-15"`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestDateUnmarshalJSON(t *testing.T) {
	jsonData := `"2023-05-15"`
	var date Date

	err := json.Unmarshal([]byte(jsonData), &date)
	if err != nil {
		t.Fatal(err)
	}

	expected := time.Date(2023, 5, 15, 0, 0, 0, 0, time.Local)
	if time.Time(date) != expected {
		t.Errorf("expected %v, got %v", expected, time.Time(date))
	}
}

func TestUpdateTransaction(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody []byte

	mockResponse := `{
		"data": {
			"transaction": {
				"account_id": "e0dc51c5-5136-4a3f-9019-84487d266cbb",
				"account_name": "Cash",
				"amount": -3000,
				"approved": true,
				"category_id": "97634123-0823-4c37-a2a6-8ec2bccb3c63",
				"category_name": "Groceries",
				"cleared": "cleared",
				"date": "2023-05-15",
				"deleted": false,
				"flag_color": "red",
				"id": "e0d8d32f-6c93-4b92-be48-c4590f3ed2a7",
				"memo": "Updated memo",
				"payee_name": "Updated Store",
				"subtransactions": []
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error("Failed to read request body:", err)
			return
		}
		receivedBody = body

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	amount := int64(-3000)
	memo := types.NullString{String: "Updated memo", Valid: true}
	req := &UpdateTransactionRequest{
		Transaction: &UpdateTransaction{
			Amount:    &amount,
			Memo:      memo,
			FlagColor: types.NullString{String: string(FlagColorRed), Valid: true},
		},
	}

	resp, err := client.Budgets("budget-123").UpdateTransaction(context.Background(), "txn-456", req)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "PUT" {
		t.Errorf("expected PUT method, got %s", receivedMethod)
	}

	expectedPath := "/budgets/budget-123/transactions/txn-456"
	if receivedPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, receivedPath)
	}

	t.Logf("Received body: %q", string(receivedBody))

	var sentData UpdateTransactionRequest
	if err := json.Unmarshal(receivedBody, &sentData); err != nil {
		t.Fatal("failed to unmarshal sent body:", err)
	}

	if *sentData.Transaction.Amount != -3000 {
		t.Errorf("expected amount -3000, got %d", *sentData.Transaction.Amount)
	}

	if sentData.Transaction.Memo.String != "Updated memo" {
		t.Errorf("expected memo 'Updated memo', got %s", sentData.Transaction.Memo.String)
	}

	if resp.Data.Transaction.Amount != -3000 {
		t.Errorf("expected response amount -3000, got %d", resp.Data.Transaction.Amount)
	}

	if resp.Data.Transaction.FlagColor != FlagColorRed {
		t.Errorf("expected color red, got %q", resp.Data.Transaction.FlagColor)
	}

	if resp.Data.Transaction.Memo != "Updated memo" {
		t.Errorf("expected response memo 'Updated memo', got %s", resp.Data.Transaction.Memo)
	}
}

func TestCreateTransaction(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody []byte

	mockResponse := `{
		"data": {
			"transaction_ids": ["e0d8d32f-6c93-4b92-be48-c4590f3ed2a7"],
			"transaction": {
				"account_id": "e0dc51c5-5136-4a3f-9019-84487d266cbb",
				"account_name": "Cash",
				"amount": -2500,
				"approved": false,
				"category_id": "97634123-0823-4c37-a2a6-8ec2bccb3c63",
				"category_name": "Groceries",
				"cleared": "uncleared",
				"date": "2023-05-15",
				"deleted": false,
				"flag_color": "blue",
				"id": "e0d8d32f-6c93-4b92-be48-c4590f3ed2a7",
				"memo": "New grocery purchase",
				"payee_name": "Local Store",
				"subtransactions": []
			},
			"server_knowledge": 123456
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error("Failed to read request body:", err)
			return
		}
		receivedBody = body

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	req := &CreateTransactionRequest{
		Transaction: &NewTransaction{
			AccountID:  "e0dc51c5-5136-4a3f-9019-84487d266cbb",
			Date:       Date(time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)),
			Amount:     -2500,
			PayeeName:  types.NullString{String: "Local Store", Valid: true},
			CategoryID: types.NullString{String: "97634123-0823-4c37-a2a6-8ec2bccb3c63", Valid: true},
			Memo:       types.NullString{String: "New grocery purchase", Valid: true},
			Cleared:    ClearedStatusUncleared,
			Approved:   false,
			FlagColor:  FlagColorBlue,
		},
	}

	resp, err := client.Budgets("budget-123").CreateTransaction(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "POST" {
		t.Errorf("expected POST method, got %s", receivedMethod)
	}

	expectedPath := "/budgets/budget-123/transactions"
	if receivedPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, receivedPath)
	}

	t.Logf("Received body: %q", string(receivedBody))

	var sentData CreateTransactionRequest
	if err := json.Unmarshal(receivedBody, &sentData); err != nil {
		t.Fatal("failed to unmarshal sent body:", err)
	}

	if sentData.Transaction.Amount != -2500 {
		t.Errorf("expected amount -2500, got %d", sentData.Transaction.Amount)
	}

	if sentData.Transaction.Memo.String != "New grocery purchase" {
		t.Errorf("expected memo 'New grocery purchase', got %s", sentData.Transaction.Memo.String)
	}

	if sentData.Transaction.Cleared != ClearedStatusUncleared {
		t.Errorf("expected cleared status 'uncleared', got %s", sentData.Transaction.Cleared)
	}

	if len(resp.Data.TransactionIDs) != 1 {
		t.Errorf("expected 1 transaction ID, got %d", len(resp.Data.TransactionIDs))
	}

	if resp.Data.Transaction.Amount != -2500 {
		t.Errorf("expected response amount -2500, got %d", resp.Data.Transaction.Amount)
	}

	if resp.Data.Transaction.FlagColor != FlagColorBlue {
		t.Errorf("expected color blue, got %q", resp.Data.Transaction.FlagColor)
	}

	if resp.Data.ServerKnowledge != 123456 {
		t.Errorf("expected server knowledge 123456, got %d", resp.Data.ServerKnowledge)
	}
}

func TestDeleteTransaction(t *testing.T) {
	var receivedMethod, receivedPath string

	mockResponse := `{
		"data": {
			"transaction": {
				"account_id": "e0dc51c5-5136-4a3f-9019-84487d266cbb",
				"account_name": "Cash",
				"amount": -2000,
				"approved": true,
				"category_id": "97634123-0823-4c37-a2a6-8ec2bccb3c63",
				"category_name": "Dining Out",
				"cleared": "reconciled",
				"date": "2023-05-15",
				"deleted": true,
				"flag_color": "red",
				"id": "e0d8d32f-6c93-4b92-be48-c4590f3ed2a7",
				"memo": "Ice Cream",
				"payee_name": "Corner Store",
				"subtransactions": []
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").DeleteTransaction(context.Background(), "txn-456")
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "DELETE" {
		t.Errorf("expected DELETE method, got %s", receivedMethod)
	}

	expectedPath := "/budgets/budget-123/transactions/txn-456"
	if receivedPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, receivedPath)
	}

	if !resp.Data.Transaction.Deleted {
		t.Errorf("expected transaction to be marked as deleted")
	}

	if resp.Data.Transaction.ID != "e0d8d32f-6c93-4b92-be48-c4590f3ed2a7" {
		t.Errorf("expected transaction ID 'e0d8d32f-6c93-4b92-be48-c4590f3ed2a7', got %s", resp.Data.Transaction.ID)
	}
}

func TestCustomUserAgent(t *testing.T) {
	var capturedUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"budgets": []}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	// Test default user agent
	defaultUA := client.GetUserAgent()
	expectedDefault := "ynab-go/" + Version
	if defaultUA != expectedDefault {
		t.Errorf("expected default user agent '%s', got '%s'", expectedDefault, defaultUA)
	}

	// Test setting custom user agent
	customUA := "transaction-categorizer/1.2.0 ynab-go/" + Version
	client.SetUserAgent(customUA)

	if client.GetUserAgent() != customUA {
		t.Errorf("expected custom user agent '%s', got '%s'", customUA, client.GetUserAgent())
	}

	// Make a request to verify the custom user agent is sent
	_, err := client.Budgets("budget-id").Categories(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(capturedUserAgent, "transaction-categorizer/1.2.0") {
		t.Errorf("User-Agent header does not contain 'transaction-categorizer/1.2.0': %s", capturedUserAgent)
	}

	if !strings.Contains(capturedUserAgent, "ynab-go/"+Version) {
		t.Errorf("User-Agent header does not contain 'ynab-go/%s': %s", Version, capturedUserAgent)
	}
}

func TestNewTransferTransaction(t *testing.T) {
	sourceAccountID := "source-account-123"
	targetAccount := &Account{
		ID:              "target-account-456",
		Name:            "Savings",
		TransferPayeeID: types.NullString{String: "transfer-payee-789", Valid: true},
	}
	amount := int64(-50000) // $50 transfer out
	date := Date(time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC))

	txn, err := NewTransferTransaction(sourceAccountID, targetAccount, amount, date)
	if err != nil {
		t.Fatal(err)
	}

	if txn.AccountID != sourceAccountID {
		t.Errorf("expected account_id %s, got %s", sourceAccountID, txn.AccountID)
	}

	if txn.PayeeID.String != "transfer-payee-789" {
		t.Errorf("expected payee_id 'transfer-payee-789', got %s", txn.PayeeID.String)
	}

	if !txn.PayeeID.Valid {
		t.Error("expected payee_id to be valid")
	}

	if txn.Amount != amount {
		t.Errorf("expected amount %d, got %d", amount, txn.Amount)
	}

	// Verify the JSON marshaling is correct
	data, err := json.Marshal(txn)
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded["payee_id"] != "transfer-payee-789" {
		t.Errorf("expected JSON payee_id to be 'transfer-payee-789', got %v", decoded["payee_id"])
	}
}

func TestNewTransferTransactionInvalidTarget(t *testing.T) {
	sourceAccountID := "source-account-123"
	targetAccount := &Account{
		ID:              "target-account-456",
		Name:            "Savings",
		TransferPayeeID: types.NullString{Valid: false}, // No transfer_payee_id
	}
	amount := int64(-50000)
	date := Date(time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC))

	_, err := NewTransferTransaction(sourceAccountID, targetAccount, amount, date)
	if err == nil {
		t.Error("expected error for account without transfer_payee_id, got nil")
	}

	expectedMsg := "target account does not have a valid transfer_payee_id"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestUpdateTransactionToTransfer(t *testing.T) {
	existingTxn := &Transaction{
		ID:          "existing-txn-123",
		AccountID:   "checking-account",
		AccountName: "Checking",
		Date:        Date(time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)),
		Amount:      -50000,
		Memo:        "ATM withdrawal",
		Cleared:     ClearedStatusCleared,
		Approved:    true,
		PayeeName:   "ATM",
	}

	targetAccount := &Account{
		ID:              "target-account-456",
		Name:            "Savings",
		TransferPayeeID: types.NullString{String: "transfer-payee-789", Valid: true},
	}

	update, err := UpdateTransactionToTransfer(existingTxn, targetAccount)
	if err != nil {
		t.Fatal(err)
	}

	if update.PayeeID.String != "transfer-payee-789" {
		t.Errorf("expected payee_id 'transfer-payee-789', got %s", update.PayeeID.String)
	}

	if !update.PayeeID.Valid {
		t.Error("expected payee_id to be valid")
	}

	// Verify existing transaction data is preserved
	if *update.Amount != -50000 {
		t.Errorf("expected amount -50000, got %d", *update.Amount)
	}

	if update.Memo.String != "ATM withdrawal" {
		t.Errorf("expected memo 'ATM withdrawal', got %s", update.Memo.String)
	}

	if update.Cleared.String != "cleared" {
		t.Errorf("expected cleared 'cleared', got %s", update.Cleared.String)
	}

	if !*update.Approved {
		t.Error("expected approved to be true")
	}

	// Verify the JSON marshaling is correct
	data, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded["payee_id"] != "transfer-payee-789" {
		t.Errorf("expected JSON payee_id to be 'transfer-payee-789', got %v", decoded["payee_id"])
	}
}

func TestUpdateTransactionToTransferInvalidTarget(t *testing.T) {
	existingTxn := &Transaction{
		ID:        "existing-txn-123",
		AccountID: "checking-account",
		Date:      Date(time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)),
		Amount:    -50000,
	}

	targetAccount := &Account{
		ID:              "target-account-456",
		Name:            "Savings",
		TransferPayeeID: types.NullString{Valid: false},
	}

	_, err := UpdateTransactionToTransfer(existingTxn, targetAccount)
	if err == nil {
		t.Error("expected error for account without transfer_payee_id, got nil")
	}

	expectedMsg := "target account does not have a valid transfer_payee_id"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestCreateTransferTransactionIntegration(t *testing.T) {
	var receivedBody []byte

	mockResponse := `{
		"data": {
			"transaction_ids": ["transfer-txn-123"],
			"transaction": {
				"account_id": "checking-account",
				"account_name": "Checking",
				"amount": -100000,
				"approved": false,
				"category_id": null,
				"category_name": null,
				"cleared": "uncleared",
				"date": "2023-06-15",
				"deleted": false,
				"flag_color": null,
				"id": "transfer-txn-123",
				"memo": "",
				"payee_name": "Transfer : Savings",
				"subtransactions": [],
				"transfer_account_id": "savings-account",
				"transfer_transaction_id": "transfer-txn-456"
			},
			"server_knowledge": 12345
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = body

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	savingsAccount := &Account{
		ID:              "savings-account",
		Name:            "Savings",
		TransferPayeeID: types.NullString{String: "savings-transfer-payee", Valid: true},
	}

	txn, err := NewTransferTransaction(
		"checking-account",
		savingsAccount,
		-100000, // $100 transfer out
		Date(time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)),
	)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Budgets("budget-123").CreateTransaction(context.Background(), &CreateTransactionRequest{
		Transaction: txn,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify the request body contains the transfer payee ID
	var sentData CreateTransactionRequest
	if err := json.Unmarshal(receivedBody, &sentData); err != nil {
		t.Fatal("failed to unmarshal sent body:", err)
	}

	if sentData.Transaction.PayeeID.String != "savings-transfer-payee" {
		t.Errorf("expected payee_id 'savings-transfer-payee', got %s", sentData.Transaction.PayeeID.String)
	}

	// Verify response shows transfer was created
	if resp.Data.Transaction.TransferAccountID.String != "savings-account" {
		t.Errorf("expected transfer_account_id 'savings-account', got %s", resp.Data.Transaction.TransferAccountID.String)
	}
}

func TestUpdateToTransferIntegration(t *testing.T) {
	var receivedBody []byte

	mockResponse := `{
		"data": {
			"transaction": {
				"account_id": "checking-account",
				"account_name": "Checking",
				"amount": -50000,
				"approved": true,
				"category_id": null,
				"category_name": null,
				"cleared": "cleared",
				"date": "2023-06-15",
				"deleted": false,
				"flag_color": null,
				"id": "existing-txn-123",
				"memo": "ATM withdrawal",
				"payee_name": "Transfer : Savings",
				"subtransactions": [],
				"transfer_account_id": "savings-account",
				"transfer_transaction_id": "transfer-txn-789"
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = body

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	// Simulate an existing transaction that we want to convert to a transfer
	existingTxn := &Transaction{
		ID:          "existing-txn-123",
		AccountID:   "checking-account",
		AccountName: "Checking",
		Date:        Date(time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)),
		Amount:      -50000,
		Memo:        "ATM withdrawal",
		Cleared:     ClearedStatusCleared,
		Approved:    true,
		PayeeName:   "ATM",
	}

	savingsAccount := &Account{
		ID:              "savings-account",
		Name:            "Savings",
		TransferPayeeID: types.NullString{String: "savings-transfer-payee", Valid: true},
	}

	update, err := UpdateTransactionToTransfer(existingTxn, savingsAccount)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Budgets("budget-123").UpdateTransaction(
		context.Background(),
		existingTxn.ID,
		&UpdateTransactionRequest{Transaction: update},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the request body contains the transfer payee ID
	var sentData UpdateTransactionRequest
	if err := json.Unmarshal(receivedBody, &sentData); err != nil {
		t.Fatal("failed to unmarshal sent body:", err)
	}

	if sentData.Transaction.PayeeID.String != "savings-transfer-payee" {
		t.Errorf("expected payee_id 'savings-transfer-payee', got %s", sentData.Transaction.PayeeID.String)
	}

	// Verify the existing transaction data was preserved in the request
	if *sentData.Transaction.Amount != -50000 {
		t.Errorf("expected amount -50000, got %d", *sentData.Transaction.Amount)
	}

	if sentData.Transaction.Memo.String != "ATM withdrawal" {
		t.Errorf("expected memo 'ATM withdrawal', got %s", sentData.Transaction.Memo.String)
	}

	// Verify response shows it's now a transfer
	if resp.Data.Transaction.TransferAccountID.String != "savings-account" {
		t.Errorf("expected transfer_account_id 'savings-account', got %s", resp.Data.Transaction.TransferAccountID.String)
	}
}

func TestAccountTransferPayeeIDParsing(t *testing.T) {
	jsonData := `{
		"data": {
			"accounts": [
				{
					"id": "account-123",
					"name": "Checking",
					"type": "checking",
					"on_budget": true,
					"closed": false,
					"note": null,
					"balance": 500000,
					"cleared_balance": 500000,
					"uncleared_balance": 0,
					"transfer_payee_id": "payee-456",
					"direct_import_linked": false,
					"direct_import_in_error": false,
					"deleted": false
				},
				{
					"id": "account-789",
					"name": "Tracking Account",
					"type": "otherAsset",
					"on_budget": false,
					"closed": false,
					"note": null,
					"balance": 100000,
					"cleared_balance": 100000,
					"uncleared_balance": 0,
					"transfer_payee_id": null,
					"direct_import_linked": false,
					"direct_import_in_error": false,
					"deleted": false
				}
			]
		}
	}`

	var resp AccountListResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatal(err)
	}

	if len(resp.Data.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(resp.Data.Accounts))
	}

	// First account has transfer_payee_id
	acct1 := resp.Data.Accounts[0]
	if !acct1.TransferPayeeID.Valid {
		t.Error("expected first account to have valid transfer_payee_id")
	}
	if acct1.TransferPayeeID.String != "payee-456" {
		t.Errorf("expected transfer_payee_id 'payee-456', got %s", acct1.TransferPayeeID.String)
	}

	// Second account has null transfer_payee_id
	acct2 := resp.Data.Accounts[1]
	if acct2.TransferPayeeID.Valid {
		t.Error("expected second account to have invalid (null) transfer_payee_id")
	}
}
