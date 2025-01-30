// The ynab-largest-inputs-outputs function finds the largest inputs and outputs
// to your Net Worth, optionally filtered by a month argument. Any income or
// outflows that come into either your budget accounts or your tracking accounts
// will appear here. One exception is that credit card spending is accounted at
// the time of payment, not at the time the money is spent.
//
// Pass the --month flag to filter by a given month. The flag accepts arguments
// in the form of 'Jan 2006', e.g. --month='Aug 2019'.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kevinburke/ynab-go"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
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
	exclude := flag.String("exclude", "", "Comma separated list of accounts to exclude")
	monthStr := flag.String("month", "", "Month to print inputs and outputs for")
	yearStr := flag.String("year", "", "Year to print inputs and outputs for")
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
	txns, err := getTransactions(client, thisBudget.ID)
	if err != nil {
		log.Fatal(err)
	}
	var month, endOfMonth time.Time
	if *monthStr != "" && *yearStr != "" {
		log.Fatalf("can't specify both --month and --year")
	}
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
	} else if *yearStr != "" {
		var err error
		month, err = time.Parse("2006", *yearStr)
		if err != nil {
			log.Fatal(err)
		}
		if month.IsZero() {
			log.Fatalf("could not parse year: %v", *yearStr)
		}
		endOfMonth = time.Date(month.Year()+1, month.Month(), month.Day(), 0, 0, 0, 0, time.Local)
	}

	inflows := make([]*ynab.Transaction, 0)
	outflows := make([]*ynab.Transaction, 0)
	inflowSum := int64(0)
	outflowSum := int64(0)
	runningTotal := int64(0)
	excludes := make(map[string]struct{})
	parts := strings.Split(*exclude, ",")
	for _, part := range parts {
		excludes[part] = struct{}{}
	}
	for i := range txns {
		tx := txns[i]
		if _, ok := excludes[tx.AccountName]; ok {
			continue
		}
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
	if *monthStr != "" {
		fmt.Println("Month Balance: $" + amt(runningTotal))
	} else if *yearStr != "" {
		fmt.Println("Year Balance: $" + amt(runningTotal))
	}
	fmt.Printf("\nInflows: $%s\n================================\n", amt(inflowSum))
	count := 0
	runningInflow := int64(0)
	for i := range inflows {
		tx := inflows[i]
		runningInflow += tx.Amount
		payeeFmt := " %s"
		payee := strings.Replace(tx.PayeeName, " : ", ": ", -1)
		var memo string
		if tx.Memo != "" {
			memo = fmt.Sprintf("%q", tx.Memo)
			payeeFmt = "%q"
		}
		fmt.Printf("%s %10s %10s %-22s "+payeeFmt+" %s\n", tx.Date.String(), "$"+amt(tx.Amount), "$"+amt(runningInflow), tx.AccountName, payee, memo)
		count++
		if count > 10 && tx.Amount < 100*1000 {
			break
		}
	}
	count = 0
	runningOutflow := int64(0)
	fmt.Printf("\nOutflows: $%s\n================================\n", amt(-1*outflowSum))
	for i := range outflows {
		tx := outflows[i]
		runningOutflow += tx.Amount
		payee := strings.Replace(tx.PayeeName, " : ", ": ", -1)
		var memo string
		payeeFmt := " %s"
		if tx.Memo != "" {
			memo = fmt.Sprintf("%q", tx.Memo)
			payeeFmt = "%q"
		}
		fmt.Printf("%s %10s %10s %-22s "+payeeFmt+" %s\n", tx.Date.String(), "$"+amt(-1*tx.Amount), "$"+amt(-1*runningOutflow), tx.AccountName, payee, memo)
		count++
		if count > 10 && (-1*tx.Amount) < 100*1000 {
			break
		}
	}
}

var p = message.NewPrinter(language.English)

func amt(amount int64) string {
	return p.Sprintf("%.2f", float64(amount)/1000)
}
