package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/kevinburke/ynab-go"
)

func getBudgets(client *ynab.Client) ([]*ynab.Budget, error) {
	budgetResp, err := client.Budgets.GetPage(context.TODO(), url.Values{})
	if err != nil {
		return nil, err
	}
	return budgetResp.Data.Budgets, nil
}

func getAccounts(client *ynab.Client, budgetID string) ([]*ynab.Account, error) {
	accountResp, err := client.Budgets.GetAccounts(context.TODO(), budgetID, url.Values{})
	if err != nil {
		return nil, err
	}
	return accountResp.Data.Accounts, nil
}

func getTransactions(client *ynab.Client, budgetID string) ([]*ynab.Transaction, error) {
	transactionResp, err := client.Budgets.GetTransactions(context.TODO(), budgetID, url.Values{})
	if err != nil {
		return nil, err
	}
	return transactionResp.Data.Transactions, nil
}

func isBlackBox(accountMap map[string]*ynab.Account, tx *ynab.Transaction) bool {
	txnAccount, ok := accountMap[tx.AccountID]
	if !ok {
		panic("unknown account: " + txnAccount.ID + " " + txnAccount.Name)
	}
	var transferAccount *ynab.Account
	if tx.TransferAccountID.Valid {
		var ok bool
		transferAccount, ok = accountMap[tx.TransferAccountID.String]
		if !ok {
			panic("could not find id: " + tx.TransferAccountID.String)
		}
	}
	if txnAccount.CashBacked() {
		if transferAccount == nil {
			return true
		}
		if transferAccount.Type == "creditCard" || transferAccount.Type == "otherLiability" {
			return true
		}
		//fmt.Printf("transfer tx: %s %s %s %s\n", time.Time(tx.Date).Format("2006-01-02"), tx.AccountName, tx.PayeeName, amt(tx.Amount))
		//fmt.Println(transferAccount.Type)
		return false
	}
	if transferAccount != nil {
		return false
	}
	// if it's a credit account, spending is not actually an "outflow"
	if txnAccount.OnBudget && !txnAccount.CashBacked() && tx.Amount < 0 {
		return false
	}
	return true
}

func main() {
	budgetName := flag.String("budget-name", "", "Name of the budget to compute inputs and outputs for")
	flag.Parse()
	token, ok := os.LookupEnv("YNAB_TOKEN")
	if !ok {
		log.Fatal("please set YNAB_TOKEN in the environment: https://app.youneedabudget.com/settings")
	}
	client := ynab.NewClient(token)
	budgets, err := getBudgets(client)
	if err != nil {
		log.Fatal(err)
	}

	var thisBudget *ynab.Budget
	if len(budgets) == 1 {
		thisBudget = budgets[0]
	} else {
		if *budgetName == "" {
			log.Fatal("please use --budget-name to tell us which budget to calculate!")
		}

		for _, budget := range budgets {
			if budget.Name == *budgetName {
				thisBudget = budget
				break
			}
		}
		if thisBudget == nil {
			log.Fatalf("could not find budget with name %q, please double check!", *budgetName)
		}
	}

	accounts, err := getAccounts(client, thisBudget.ID)
	if err != nil {
		log.Fatal(err)
	}
	accountMap := make(map[string]*ynab.Account, len(accounts))
	for _, account := range accounts {
		accountMap[account.ID] = account
	}
	var txns []*ynab.Transaction
	txns, err = getTransactions(client, thisBudget.ID)
	if err != nil {
		log.Fatal(err)
	}

	for i := range txns {
		tx := txns[i]
		txnAccount, ok := accountMap[tx.AccountID]
		if !ok {
			panic("unknown account: " + tx.AccountID)
		}
		if isBlackBox(accountMap, tx) {
			fmt.Printf("outflow tx: %s %s %s %s\n", time.Time(tx.Date).Format("2006-01-02"), tx.AccountName, tx.PayeeName, amt(tx.Amount))
		}
		_ = txnAccount
	}
}

func amt(amount int64) string {
	return strconv.FormatFloat(float64(amount)/1000, 'f', 2, 64)
}
