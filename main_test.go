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
	resp, err := client.Budgets.GetCategories(context.Background(), "category-id", nil)
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

	resp, err := client.Transactions.UpdateTransaction(context.Background(), "budget-123", "txn-456", req)
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
