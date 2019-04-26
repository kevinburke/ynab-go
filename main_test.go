package ynab

import (
	"encoding/json"
	"fmt"
	"testing"
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
		"flag_color": null,
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
	fmt.Println("txnList", txnList.Data.Transactions)
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
