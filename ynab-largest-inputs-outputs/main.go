package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
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
		if transferAccount.Type == "creditCard" && tx.Amount < 0 {
			return true
		}
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
	monthStr := flag.String("month", "", "Month to print inputs and outputs for")
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
	var month, endOfMonth time.Time
	if *monthStr != "" {
		var err error
		month, err = time.Parse("Jan 2006", *monthStr)
		if err != nil {
			month, err = time.Parse("January 2006", *monthStr)
			if err != nil {
				log.Fatal(err)
			}
		}
		if month.IsZero() {
			log.Fatalf("could not parse month as month: %v", *monthStr)
		}
		endOfMonth = time.Date(month.Year(), month.Month()+1, month.Day(), 0, 0, 0, 0, time.Local)
	}

	inflows := make([]*ynab.Transaction, 0)
	outflows := make([]*ynab.Transaction, 0)
	inflowSum := int64(0)
	outflowSum := int64(0)
	runningTotal := int64(0)
	for i := range txns {
		tx := txns[i]
		if tx.Amount == 0 {
			continue
		}
		if !month.IsZero() {
			tt := time.Time(tx.Date)
			if tt.Before(month) || !tt.Before(endOfMonth) {
				continue
			}
		}
		if isBlackBox(accountMap, tx) {
			runningTotal += tx.Amount
			if tx.Amount < 0 {
				outflows = append(outflows, tx)
				outflowSum += tx.Amount
			}
			if tx.Amount > 0 {
				inflows = append(inflows, tx)
				inflowSum += tx.Amount
			}
		}
	}
	sort.Slice(inflows, func(i, j int) bool {
		return inflows[i].Amount > inflows[j].Amount
	})
	sort.Slice(outflows, func(i, j int) bool {
		return outflows[i].Amount < outflows[j].Amount
	})
	fmt.Println("Month Balance: $" + amt(runningTotal))
	fmt.Printf("\nInflows: $%s\n============================\n", amt(inflowSum))
	count := 0
	for i := range inflows {
		tx := inflows[i]
		fmt.Printf("%s %10s %s %s\n", tx.Date.String(), "$"+amt(tx.Amount), tx.AccountName, tx.PayeeName)
		count++
		if count > 10 && tx.Amount < 100*1000 {
			break
		}
	}
	count = 0
	fmt.Printf("\nOutflows: $%s\n============================\n", amt(outflowSum))
	for i := range outflows {
		tx := outflows[i]
		fmt.Printf("%s %10s %s %s\n", tx.Date.String(), "$"+amt(-1*tx.Amount), tx.AccountName, tx.PayeeName)
		count++
		if count > 10 && (-1*tx.Amount) < 100*1000 {
			break
		}
	}
}

func amt(amount int64) string {
	return strconv.FormatFloat(float64(amount)/1000, 'f', 2, 64)
}
