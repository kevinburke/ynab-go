package ynab_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kevinburke/go-types"
	"github.com/kevinburke/ynab-go"
)

func ExampleBudgetService_CreateTransaction() {
	client := ynab.NewClient("your-api-token")

	// Create a normal expense transaction
	txn := &ynab.NewTransaction{
		AccountID:  "checking-account-id",
		Date:       ynab.Date(time.Date(2023, 6, 15, 0, 0, 0, 0, time.Local)),
		Amount:     -25000, // -$25.00 in milliunits
		PayeeName:  types.NullString{String: "Coffee Shop", Valid: true},
		CategoryID: types.NullString{String: "food-category-id", Valid: true},
		Memo:       types.NullString{String: "Morning coffee", Valid: true},
		Cleared:    ynab.ClearedStatusCleared,
		Approved:   true,
	}

	resp, err := client.Budgets("budget-id").CreateTransaction(
		context.Background(),
		&ynab.CreateTransactionRequest{Transaction: txn},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created transaction: %s\n", resp.Data.Transaction.ID)
}

func ExampleBudgetService_UpdateTransaction() {
	client := ynab.NewClient("your-api-token")

	// Update an existing transaction's memo and category
	update := &ynab.UpdateTransaction{
		Date:       ynab.Date(time.Date(2023, 6, 15, 0, 0, 0, 0, time.Local)),
		Memo:       types.NullString{String: "Updated memo", Valid: true},
		CategoryID: types.NullString{String: "new-category-id", Valid: true},
	}

	resp, err := client.Budgets("budget-id").UpdateTransaction(
		context.Background(),
		"transaction-id",
		&ynab.UpdateTransactionRequest{Transaction: update},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Updated transaction: %s\n", resp.Data.Transaction.ID)
}

func ExampleNewTransferTransaction() {
	client := ynab.NewClient("your-api-token")

	// First, get the accounts to find the transfer_payee_id
	accounts, err := client.Budgets("budget-id").Accounts(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	// Find the source and target accounts
	var checkingAccount, savingsAccount *ynab.Account
	for _, acct := range accounts.Data.Accounts {
		switch acct.Name {
		case "Checking":
			checkingAccount = acct
		case "Savings":
			savingsAccount = acct
		}
	}

	if checkingAccount == nil || savingsAccount == nil {
		log.Fatal("could not find accounts")
	}

	// Create a transfer from Checking to Savings
	// Negative amount means money leaves the source account (Checking)
	txn, err := ynab.NewTransferTransaction(
		checkingAccount.ID, // source account
		savingsAccount,     // target account (has TransferPayeeID)
		-100000,            // -$100.00 transfer out of Checking
		ynab.Date(time.Date(2023, 6, 15, 0, 0, 0, 0, time.Local)),
	)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Budgets("budget-id").CreateTransaction(
		context.Background(),
		&ynab.CreateTransactionRequest{Transaction: txn},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created transfer: %s\n", resp.Data.Transaction.ID)
	fmt.Printf("Transfer to account: %s\n", resp.Data.Transaction.TransferAccountID.String)
}

func ExampleUpdateTransactionToTransfer() {
	client := ynab.NewClient("your-api-token")
	ctx := context.Background()

	// Get transactions to find the one we want to convert to a transfer
	// (e.g., an ATM withdrawal that should actually be a transfer to cash account)
	txnResp, err := client.Budgets("budget-id").Transactions(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Find the transaction to convert (in practice, you'd have the ID already)
	var existingTxn *ynab.Transaction
	for _, txn := range txnResp.Data.Transactions {
		if txn.PayeeName == "ATM Withdrawal" {
			existingTxn = txn
			break
		}
	}
	if existingTxn == nil {
		log.Fatal("transaction not found")
	}

	// Get the target account for the transfer
	accounts, err := client.Budgets("budget-id").Accounts(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	var cashAccount *ynab.Account
	for _, acct := range accounts.Data.Accounts {
		if acct.Name == "Cash" {
			cashAccount = acct
			break
		}
	}

	if cashAccount == nil {
		log.Fatal("could not find Cash account")
	}

	// Convert the transaction to a transfer
	// This preserves the date, amount, memo, cleared status, and approval
	update, err := ynab.UpdateTransactionToTransfer(existingTxn, cashAccount)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Budgets("budget-id").UpdateTransaction(
		ctx,
		existingTxn.ID,
		&ynab.UpdateTransactionRequest{Transaction: update},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Converted to transfer: %s\n", resp.Data.Transaction.ID)
	fmt.Printf("Now transfers to: %s\n", resp.Data.Transaction.TransferAccountID.String)
}
