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

func TestGetUser(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"user": {"id": "user-123"}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.GetUser(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/user" {
		t.Errorf("expected /user, got %s", receivedPath)
	}
	if resp.Data.User.ID != "user-123" {
		t.Errorf("expected user ID user-123, got %s", resp.Data.User.ID)
	}
}

func TestGetSettings(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"settings": {"date_format": {"format": "MM/DD/YYYY"}, "currency_format": {"iso_code": "USD", "example_format": "123,456.78", "decimal_digits": 2, "decimal_separator": ".", "symbol_first": true, "group_separator": ",", "currency_symbol": "$", "display_symbol": true}}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").GetSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/settings" {
		t.Errorf("expected /budgets/budget-123/settings, got %s", receivedPath)
	}
	if resp.Data.Settings.DateFormat.Format != "MM/DD/YYYY" {
		t.Errorf("expected date format MM/DD/YYYY, got %s", resp.Data.Settings.DateFormat.Format)
	}
	if resp.Data.Settings.CurrencyFormat.ISOCode != "USD" {
		t.Errorf("expected ISO code USD, got %s", resp.Data.Settings.CurrencyFormat.ISOCode)
	}
	if resp.Data.Settings.CurrencyFormat.DecimalDigits != 2 {
		t.Errorf("expected 2 decimal digits, got %d", resp.Data.Settings.CurrencyFormat.DecimalDigits)
	}
	if !resp.Data.Settings.CurrencyFormat.SymbolFirst {
		t.Errorf("expected symbol_first to be true")
	}
}

func TestCreateAccount(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data": {"account": {"id": "acct-new", "name": "Savings", "type": "savings", "on_budget": true, "closed": false, "note": "", "balance": 100000, "starting_balance": 100000, "deleted": false}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	req := &CreateAccountRequest{
		Account: &SaveAccount{
			Name:    "Savings",
			Type:    "savings",
			Balance: 100000,
		},
	}

	resp, err := client.Budgets("budget-123").CreateAccount(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "POST" {
		t.Errorf("expected POST, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/accounts" {
		t.Errorf("expected /budgets/budget-123/accounts, got %s", receivedPath)
	}

	var sentData CreateAccountRequest
	if err := json.Unmarshal(receivedBody, &sentData); err != nil {
		t.Fatal(err)
	}
	if sentData.Account.Name != "Savings" {
		t.Errorf("expected account name Savings, got %s", sentData.Account.Name)
	}
	if sentData.Account.Balance != 100000 {
		t.Errorf("expected balance 100000, got %d", sentData.Account.Balance)
	}
	if resp.Data.Account.ID != "acct-new" {
		t.Errorf("expected account ID acct-new, got %s", resp.Data.Account.ID)
	}
}

func TestGetAccount(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"account": {"id": "acct-123", "name": "Checking", "type": "checking", "on_budget": true, "closed": false, "note": "", "balance": 500000, "starting_balance": 0, "deleted": false}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").GetAccount(context.Background(), "acct-123")
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/accounts/acct-123" {
		t.Errorf("expected /budgets/budget-123/accounts/acct-123, got %s", receivedPath)
	}
	if resp.Data.Account.ID != "acct-123" {
		t.Errorf("expected account ID acct-123, got %s", resp.Data.Account.ID)
	}
	if resp.Data.Account.Balance != 500000 {
		t.Errorf("expected balance 500000, got %d", resp.Data.Account.Balance)
	}
}

func TestGetCategory(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"category": {"id": "cat-123", "name": "Groceries", "category_group_id": "group-1", "note": "", "hidden": false, "budgeted": 50000, "activity": -30000, "balance": 20000}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").GetCategory(context.Background(), "cat-123")
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/categories/cat-123" {
		t.Errorf("expected /budgets/budget-123/categories/cat-123, got %s", receivedPath)
	}
	if resp.Data.Category.ID != "cat-123" {
		t.Errorf("expected category ID cat-123, got %s", resp.Data.Category.ID)
	}
	if resp.Data.Category.Budgeted != 50000 {
		t.Errorf("expected budgeted 50000, got %d", resp.Data.Category.Budgeted)
	}
}

func TestUpdateCategory(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"category": {"id": "cat-123", "name": "Renamed", "category_group_id": "group-1", "note": "updated note", "hidden": false, "budgeted": 0, "activity": 0, "balance": 0}, "server_knowledge": 100}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	req := &UpdateCategoryRequest{
		Category: &SaveCategory{
			Name: "Renamed",
			Note: "updated note",
		},
	}

	resp, err := client.Budgets("budget-123").UpdateCategory(context.Background(), "cat-123", req)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "PATCH" {
		t.Errorf("expected PATCH, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/categories/cat-123" {
		t.Errorf("expected /budgets/budget-123/categories/cat-123, got %s", receivedPath)
	}

	var sentData UpdateCategoryRequest
	if err := json.Unmarshal(receivedBody, &sentData); err != nil {
		t.Fatal(err)
	}
	if sentData.Category.Name != "Renamed" {
		t.Errorf("expected name Renamed, got %s", sentData.Category.Name)
	}
	if resp.Data.Category.Name != "Renamed" {
		t.Errorf("expected response name Renamed, got %s", resp.Data.Category.Name)
	}
	if resp.Data.ServerKnowledge != 100 {
		t.Errorf("expected server knowledge 100, got %d", resp.Data.ServerKnowledge)
	}
}

func TestPayees(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"payees": [{"id": "payee-1", "name": "Grocery Store", "transfer_account_id": null, "deleted": false}], "server_knowledge": 50}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").Payees(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/payees" {
		t.Errorf("expected /budgets/budget-123/payees, got %s", receivedPath)
	}
	if len(resp.Data.Payees) != 1 {
		t.Fatalf("expected 1 payee, got %d", len(resp.Data.Payees))
	}
	if resp.Data.Payees[0].Name != "Grocery Store" {
		t.Errorf("expected payee name Grocery Store, got %s", resp.Data.Payees[0].Name)
	}
	if resp.Data.ServerKnowledge != 50 {
		t.Errorf("expected server knowledge 50, got %d", resp.Data.ServerKnowledge)
	}
}

func TestGetPayee(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"payee": {"id": "payee-1", "name": "Grocery Store", "transfer_account_id": null, "deleted": false}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").GetPayee(context.Background(), "payee-1")
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/payees/payee-1" {
		t.Errorf("expected /budgets/budget-123/payees/payee-1, got %s", receivedPath)
	}
	if resp.Data.Payee.ID != "payee-1" {
		t.Errorf("expected payee ID payee-1, got %s", resp.Data.Payee.ID)
	}
}

func TestUpdatePayee(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"payee": {"id": "payee-1", "name": "Updated Store", "transfer_account_id": null, "deleted": false}, "server_knowledge": 75}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	req := &UpdatePayeeRequest{
		Payee: &SavePayee{Name: "Updated Store"},
	}

	resp, err := client.Budgets("budget-123").UpdatePayee(context.Background(), "payee-1", req)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "PATCH" {
		t.Errorf("expected PATCH, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/payees/payee-1" {
		t.Errorf("expected /budgets/budget-123/payees/payee-1, got %s", receivedPath)
	}

	var sentData UpdatePayeeRequest
	if err := json.Unmarshal(receivedBody, &sentData); err != nil {
		t.Fatal(err)
	}
	if sentData.Payee.Name != "Updated Store" {
		t.Errorf("expected payee name Updated Store, got %s", sentData.Payee.Name)
	}
	if resp.Data.Payee.Name != "Updated Store" {
		t.Errorf("expected response payee name Updated Store, got %s", resp.Data.Payee.Name)
	}
	if resp.Data.ServerKnowledge != 75 {
		t.Errorf("expected server knowledge 75, got %d", resp.Data.ServerKnowledge)
	}
}

func TestPayeeLocations(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"payee_locations": [{"id": "loc-1", "payee_id": "payee-1", "latitude": "40.7128", "longitude": "-74.0060", "deleted": false}]}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").PayeeLocations(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/payee_locations" {
		t.Errorf("expected /budgets/budget-123/payee_locations, got %s", receivedPath)
	}
	if len(resp.Data.PayeeLocations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(resp.Data.PayeeLocations))
	}
	if resp.Data.PayeeLocations[0].Latitude != "40.7128" {
		t.Errorf("expected latitude 40.7128, got %s", resp.Data.PayeeLocations[0].Latitude)
	}
}

func TestGetPayeeLocation(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"payee_location": {"id": "loc-1", "payee_id": "payee-1", "latitude": "40.7128", "longitude": "-74.0060", "deleted": false}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").GetPayeeLocation(context.Background(), "loc-1")
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/payee_locations/loc-1" {
		t.Errorf("expected /budgets/budget-123/payee_locations/loc-1, got %s", receivedPath)
	}
	if resp.Data.PayeeLocation.ID != "loc-1" {
		t.Errorf("expected location ID loc-1, got %s", resp.Data.PayeeLocation.ID)
	}
	if resp.Data.PayeeLocation.Longitude != "-74.0060" {
		t.Errorf("expected longitude -74.0060, got %s", resp.Data.PayeeLocation.Longitude)
	}
}

func TestPayeeLocationsByPayee(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"payee_locations": [{"id": "loc-1", "payee_id": "payee-1", "latitude": "34.0522", "longitude": "-118.2437", "deleted": false}]}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").PayeeLocationsByPayee(context.Background(), "payee-1")
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/payees/payee-1/payee_locations" {
		t.Errorf("expected /budgets/budget-123/payees/payee-1/payee_locations, got %s", receivedPath)
	}
	if len(resp.Data.PayeeLocations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(resp.Data.PayeeLocations))
	}
	if resp.Data.PayeeLocations[0].PayeeID != "payee-1" {
		t.Errorf("expected payee ID payee-1, got %s", resp.Data.PayeeLocations[0].PayeeID)
	}
}

func TestMonths(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"months": [{"month": "2024-01-01", "note": "", "income": 500000, "budgeted": 400000, "activity": -350000, "to_be_budgeted": 100000, "age_of_money": 30, "deleted": false}], "server_knowledge": 200}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").Months(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/months" {
		t.Errorf("expected /budgets/budget-123/months, got %s", receivedPath)
	}
	if len(resp.Data.Months) != 1 {
		t.Fatalf("expected 1 month, got %d", len(resp.Data.Months))
	}
	if resp.Data.Months[0].Income != 500000 {
		t.Errorf("expected income 500000, got %d", resp.Data.Months[0].Income)
	}
	if *resp.Data.Months[0].AgeOfMoney != 30 {
		t.Errorf("expected age of money 30, got %d", *resp.Data.Months[0].AgeOfMoney)
	}
}

func TestGetMonth(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"month": {"month": "2024-01-01", "note": "January", "income": 500000, "budgeted": 400000, "activity": -350000, "to_be_budgeted": 100000, "age_of_money": 30, "deleted": false, "categories": [{"id": "cat-1", "name": "Groceries", "category_group_id": "group-1", "budgeted": 50000, "activity": -30000, "balance": 20000}]}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").GetMonth(context.Background(), "2024-01-01")
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/months/2024-01-01" {
		t.Errorf("expected /budgets/budget-123/months/2024-01-01, got %s", receivedPath)
	}
	if resp.Data.Month.Note != "January" {
		t.Errorf("expected note January, got %s", resp.Data.Month.Note)
	}
	if len(resp.Data.Month.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(resp.Data.Month.Categories))
	}
	if resp.Data.Month.Categories[0].Name != "Groceries" {
		t.Errorf("expected category name Groceries, got %s", resp.Data.Month.Categories[0].Name)
	}
}

func TestGetTransaction(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"transaction": {"id": "txn-123", "account_id": "acct-1", "account_name": "Cash", "date": "2024-01-15", "amount": -5000, "memo": "Coffee", "cleared": "cleared", "approved": true, "flag_color": null, "payee_name": "Cafe", "category_name": "Dining Out", "subtransactions": [], "deleted": false}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").GetTransaction(context.Background(), "txn-123")
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/transactions/txn-123" {
		t.Errorf("expected /budgets/budget-123/transactions/txn-123, got %s", receivedPath)
	}
	if resp.Data.Transaction.ID != "txn-123" {
		t.Errorf("expected transaction ID txn-123, got %s", resp.Data.Transaction.ID)
	}
	if resp.Data.Transaction.Amount != -5000 {
		t.Errorf("expected amount -5000, got %d", resp.Data.Transaction.Amount)
	}
}

func TestUpdateTransactions(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"transaction_ids": ["txn-1", "txn-2"], "server_knowledge": 300}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	amount1 := int64(-1000)
	amount2 := int64(-2000)
	req := &UpdateTransactionsRequest{
		Transactions: []*UpdateTransaction{
			{Amount: &amount1, Date: Date(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))},
			{Amount: &amount2, Date: Date(time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC))},
		},
	}

	resp, err := client.Budgets("budget-123").UpdateTransactions(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "PATCH" {
		t.Errorf("expected PATCH, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/transactions" {
		t.Errorf("expected /budgets/budget-123/transactions, got %s", receivedPath)
	}

	var sentData UpdateTransactionsRequest
	if err := json.Unmarshal(receivedBody, &sentData); err != nil {
		t.Fatal(err)
	}
	if len(sentData.Transactions) != 2 {
		t.Errorf("expected 2 transactions in request, got %d", len(sentData.Transactions))
	}
	if len(resp.Data.TransactionIDs) != 2 {
		t.Errorf("expected 2 transaction IDs, got %d", len(resp.Data.TransactionIDs))
	}
	if resp.Data.ServerKnowledge != 300 {
		t.Errorf("expected server knowledge 300, got %d", resp.Data.ServerKnowledge)
	}
}

func TestImportTransactions(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data": {"transaction_ids": ["import-1", "import-2"]}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").ImportTransactions(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "POST" {
		t.Errorf("expected POST, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/transactions/import" {
		t.Errorf("expected /budgets/budget-123/transactions/import, got %s", receivedPath)
	}
	if len(resp.Data.TransactionIDs) != 2 {
		t.Errorf("expected 2 transaction IDs, got %d", len(resp.Data.TransactionIDs))
	}
}

func TestAccountTransactions(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"transactions": [{"id": "txn-1", "account_id": "acct-1", "account_name": "Checking", "date": "2024-01-15", "amount": -5000, "memo": "", "cleared": "cleared", "approved": true, "flag_color": null, "payee_name": "Store", "subtransactions": [], "deleted": false}]}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").AccountTransactions(context.Background(), "acct-1", nil)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/accounts/acct-1/transactions" {
		t.Errorf("expected /budgets/budget-123/accounts/acct-1/transactions, got %s", receivedPath)
	}
	if len(resp.Data.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(resp.Data.Transactions))
	}
	if resp.Data.Transactions[0].AccountID != "acct-1" {
		t.Errorf("expected account ID acct-1, got %s", resp.Data.Transactions[0].AccountID)
	}
}

func TestCategoryTransactions(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"transactions": [{"id": "txn-1", "date": "2024-01-15", "amount": -5000, "memo": "", "cleared": "cleared", "approved": true, "flag_color": null, "account_id": "acct-1", "account_name": "Checking", "payee_name": "Store", "category_name": "Groceries", "type": "transaction", "parent_transaction_id": null, "subtransactions": [], "deleted": false}], "server_knowledge": 100}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").CategoryTransactions(context.Background(), "cat-1", nil)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/categories/cat-1/transactions" {
		t.Errorf("expected /budgets/budget-123/categories/cat-1/transactions, got %s", receivedPath)
	}
	if len(resp.Data.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(resp.Data.Transactions))
	}
	if resp.Data.Transactions[0].Type != "transaction" {
		t.Errorf("expected type transaction, got %s", resp.Data.Transactions[0].Type)
	}
}

func TestPayeeTransactions(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"transactions": [{"id": "txn-1", "date": "2024-01-15", "amount": -5000, "memo": "", "cleared": "cleared", "approved": true, "flag_color": null, "account_id": "acct-1", "account_name": "Checking", "payee_name": "Store", "category_name": "Groceries", "type": "transaction", "parent_transaction_id": null, "subtransactions": [], "deleted": false}], "server_knowledge": 100}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").PayeeTransactions(context.Background(), "payee-1", nil)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/payees/payee-1/transactions" {
		t.Errorf("expected /budgets/budget-123/payees/payee-1/transactions, got %s", receivedPath)
	}
	if len(resp.Data.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(resp.Data.Transactions))
	}
	if resp.Data.Transactions[0].PayeeName != "Store" {
		t.Errorf("expected payee name Store, got %s", resp.Data.Transactions[0].PayeeName)
	}
}

func TestMonthTransactions(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"transactions": [{"id": "txn-1", "date": "2024-01-15", "amount": -5000, "memo": "", "cleared": "cleared", "approved": true, "flag_color": null, "account_id": "acct-1", "account_name": "Checking", "payee_name": "Store", "category_name": "Groceries", "type": "transaction", "parent_transaction_id": null, "subtransactions": [], "deleted": false}], "server_knowledge": 100}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").MonthTransactions(context.Background(), "2024-01-01", nil)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/months/2024-01-01/transactions" {
		t.Errorf("expected /budgets/budget-123/months/2024-01-01/transactions, got %s", receivedPath)
	}
	if len(resp.Data.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(resp.Data.Transactions))
	}
}

func TestCreateScheduledTransaction(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data": {"scheduled_transaction": {"id": "st-new", "account_id": "acct-1", "account_name": "Checking", "date_first": "2024-02-01", "date_next": "2024-02-01", "frequency": "monthly", "amount": -50000, "memo": "Rent", "flag_color": null, "payee_name": "Landlord", "cleared": "uncleared", "approved": false, "deleted": false, "subtransactions": []}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	amount := int64(-50000)
	req := &CreateScheduledTransactionRequest{
		ScheduledTransaction: &SaveScheduledTransaction{
			AccountID: "acct-1",
			Date:      Date(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)),
			Amount:    &amount,
			Frequency: "monthly",
			Memo:      types.NullString{String: "Rent", Valid: true},
		},
	}

	resp, err := client.Budgets("budget-123").CreateScheduledTransaction(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "POST" {
		t.Errorf("expected POST, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/scheduled_transactions" {
		t.Errorf("expected /budgets/budget-123/scheduled_transactions, got %s", receivedPath)
	}

	var sentData CreateScheduledTransactionRequest
	if err := json.Unmarshal(receivedBody, &sentData); err != nil {
		t.Fatal(err)
	}
	if sentData.ScheduledTransaction.AccountID != "acct-1" {
		t.Errorf("expected account ID acct-1, got %s", sentData.ScheduledTransaction.AccountID)
	}
	if resp.Data.ScheduledTransaction.ID != "st-new" {
		t.Errorf("expected scheduled transaction ID st-new, got %s", resp.Data.ScheduledTransaction.ID)
	}
	if resp.Data.ScheduledTransaction.Frequency != "monthly" {
		t.Errorf("expected frequency monthly, got %s", resp.Data.ScheduledTransaction.Frequency)
	}
}

func TestGetScheduledTransaction(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"scheduled_transaction": {"id": "st-123", "account_id": "acct-1", "account_name": "Checking", "date_first": "2024-02-01", "date_next": "2024-03-01", "frequency": "monthly", "amount": -50000, "memo": "Rent", "flag_color": null, "payee_name": "Landlord", "cleared": "uncleared", "approved": false, "deleted": false, "subtransactions": []}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").GetScheduledTransaction(context.Background(), "st-123")
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "GET" {
		t.Errorf("expected GET, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/scheduled_transactions/st-123" {
		t.Errorf("expected /budgets/budget-123/scheduled_transactions/st-123, got %s", receivedPath)
	}
	if resp.Data.ScheduledTransaction.ID != "st-123" {
		t.Errorf("expected ID st-123, got %s", resp.Data.ScheduledTransaction.ID)
	}
	if resp.Data.ScheduledTransaction.Amount != -50000 {
		t.Errorf("expected amount -50000, got %d", resp.Data.ScheduledTransaction.Amount)
	}
}

func TestUpdateScheduledTransaction(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"scheduled_transaction": {"id": "st-123", "account_id": "acct-1", "account_name": "Checking", "date_first": "2024-02-01", "date_next": "2024-03-01", "frequency": "monthly", "amount": -60000, "memo": "Updated Rent", "flag_color": null, "payee_name": "Landlord", "cleared": "uncleared", "approved": false, "deleted": false, "subtransactions": []}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	amount := int64(-60000)
	req := &UpdateScheduledTransactionRequest{
		ScheduledTransaction: &SaveScheduledTransaction{
			AccountID: "acct-1",
			Date:      Date(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)),
			Amount:    &amount,
			Memo:      types.NullString{String: "Updated Rent", Valid: true},
		},
	}

	resp, err := client.Budgets("budget-123").UpdateScheduledTransaction(context.Background(), "st-123", req)
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "PUT" {
		t.Errorf("expected PUT, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/scheduled_transactions/st-123" {
		t.Errorf("expected /budgets/budget-123/scheduled_transactions/st-123, got %s", receivedPath)
	}

	var sentData UpdateScheduledTransactionRequest
	if err := json.Unmarshal(receivedBody, &sentData); err != nil {
		t.Fatal(err)
	}
	if *sentData.ScheduledTransaction.Amount != -60000 {
		t.Errorf("expected amount -60000, got %d", *sentData.ScheduledTransaction.Amount)
	}
	if resp.Data.ScheduledTransaction.Amount != -60000 {
		t.Errorf("expected response amount -60000, got %d", resp.Data.ScheduledTransaction.Amount)
	}
}

func TestDeleteScheduledTransaction(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"scheduled_transaction": {"id": "st-123", "account_id": "acct-1", "account_name": "Checking", "date_first": "2024-02-01", "date_next": "2024-03-01", "frequency": "monthly", "amount": -50000, "memo": "Rent", "flag_color": null, "payee_name": "Landlord", "cleared": "uncleared", "approved": false, "deleted": true, "subtransactions": []}}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.Base = server.URL

	resp, err := client.Budgets("budget-123").DeleteScheduledTransaction(context.Background(), "st-123")
	if err != nil {
		t.Fatal(err)
	}

	if receivedMethod != "DELETE" {
		t.Errorf("expected DELETE, got %s", receivedMethod)
	}
	if receivedPath != "/budgets/budget-123/scheduled_transactions/st-123" {
		t.Errorf("expected /budgets/budget-123/scheduled_transactions/st-123, got %s", receivedPath)
	}
	if !resp.Data.ScheduledTransaction.Deleted {
		t.Errorf("expected scheduled transaction to be marked as deleted")
	}
	if resp.Data.ScheduledTransaction.ID != "st-123" {
		t.Errorf("expected ID st-123, got %s", resp.Data.ScheduledTransaction.ID)
	}
}

func TestCategoryGoalFieldsParsing(t *testing.T) {
	jsonData := `{
		"data": {
			"category": {
				"id": "cat-goal-1",
				"name": "Internet",
				"category_group_id": "group-bills",
				"note": "",
				"hidden": false,
				"deleted": false,
				"budgeted": 50000,
				"activity": -49990,
				"balance": 10,
				"goal_type": "NEED",
				"goal_target": 50000,
				"goal_percentage_complete": 100,
				"goal_months_to_budget": 0,
				"goal_under_funded": 0,
				"goal_overall_funded": 50000,
				"goal_overall_left": 0
			}
		}
	}`

	var resp CategoryResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatal(err)
	}

	cat := resp.Data.Category
	if cat.ID != "cat-goal-1" {
		t.Errorf("expected ID cat-goal-1, got %s", cat.ID)
	}
	if !cat.GoalType.Valid || cat.GoalType.String != "NEED" {
		t.Errorf("expected goal_type NEED, got valid=%v string=%q", cat.GoalType.Valid, cat.GoalType.String)
	}
	if cat.GoalTarget == nil || *cat.GoalTarget != 50000 {
		t.Errorf("expected goal_target 50000, got %v", cat.GoalTarget)
	}
	if cat.GoalPercentageComplete == nil || *cat.GoalPercentageComplete != 100 {
		t.Errorf("expected goal_percentage_complete 100, got %v", cat.GoalPercentageComplete)
	}
	if cat.GoalMonthsToBudget == nil || *cat.GoalMonthsToBudget != 0 {
		t.Errorf("expected goal_months_to_budget 0, got %v", cat.GoalMonthsToBudget)
	}
	if cat.GoalUnderFunded == nil || *cat.GoalUnderFunded != 0 {
		t.Errorf("expected goal_under_funded 0, got %v", cat.GoalUnderFunded)
	}
	if cat.GoalOverallFunded == nil || *cat.GoalOverallFunded != 50000 {
		t.Errorf("expected goal_overall_funded 50000, got %v", cat.GoalOverallFunded)
	}
	if cat.GoalOverallLeft == nil || *cat.GoalOverallLeft != 0 {
		t.Errorf("expected goal_overall_left 0, got %v", cat.GoalOverallLeft)
	}
	if cat.Deleted {
		t.Error("expected deleted to be false")
	}
}

func TestCategoryGoalFieldsNull(t *testing.T) {
	jsonData := `{
		"data": {
			"category": {
				"id": "cat-nogoal-1",
				"name": "Fun Money",
				"category_group_id": "group-fun",
				"note": "",
				"hidden": false,
				"deleted": false,
				"budgeted": 10000,
				"activity": -5000,
				"balance": 5000,
				"goal_type": null,
				"goal_target": null,
				"goal_percentage_complete": null,
				"goal_months_to_budget": null,
				"goal_under_funded": null,
				"goal_overall_funded": null,
				"goal_overall_left": null
			}
		}
	}`

	var resp CategoryResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatal(err)
	}

	cat := resp.Data.Category
	if cat.ID != "cat-nogoal-1" {
		t.Errorf("expected ID cat-nogoal-1, got %s", cat.ID)
	}
	if cat.GoalType.Valid {
		t.Errorf("expected goal_type to be null, got %q", cat.GoalType.String)
	}
	if cat.GoalTarget != nil {
		t.Errorf("expected goal_target to be nil, got %v", *cat.GoalTarget)
	}
	if cat.GoalPercentageComplete != nil {
		t.Errorf("expected goal_percentage_complete to be nil, got %v", *cat.GoalPercentageComplete)
	}
	if cat.GoalMonthsToBudget != nil {
		t.Errorf("expected goal_months_to_budget to be nil, got %v", *cat.GoalMonthsToBudget)
	}
	if cat.GoalUnderFunded != nil {
		t.Errorf("expected goal_under_funded to be nil, got %v", *cat.GoalUnderFunded)
	}
	if cat.GoalOverallFunded != nil {
		t.Errorf("expected goal_overall_funded to be nil, got %v", *cat.GoalOverallFunded)
	}
	if cat.GoalOverallLeft != nil {
		t.Errorf("expected goal_overall_left to be nil, got %v", *cat.GoalOverallLeft)
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
