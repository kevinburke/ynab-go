package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kevinburke/ynab-go"
)

func getAccounts(client *ynab.Client, budgetID string) ([]*ynab.Account, error) {
	accountResp, err := client.Budgets.GetAccounts(context.TODO(), budgetID, url.Values{})
	if err != nil {
		return nil, err
	}
	return accountResp.Data.Accounts, nil
}

func getBudgets(client *ynab.Client) ([]*ynab.Budget, error) {
	budgetResp, err := client.Budgets.GetPage(context.TODO(), url.Values{})
	if err != nil {
		return nil, err
	}
	return budgetResp.Data.Budgets, nil
}

func getTransactions(client *ynab.Client, budgetID string) ([]*ynab.Transaction, error) {
	transactionResp, err := client.Budgets.GetTransactions(context.TODO(), budgetID, url.Values{})
	if err != nil {
		return nil, err
	}
	return transactionResp.Data.Transactions, nil
}

func getScheduledTransactions(client *ynab.Client, budgetID string) ([]*ynab.ScheduledTransaction, error) {
	transactionResp, err := client.Budgets.GetScheduledTransactions(context.TODO(), budgetID, url.Values{})
	if err != nil {
		return nil, err
	}
	return transactionResp.Data.ScheduledTransactions, nil
}

func isOutflow(accountMap map[string]*ynab.Account, tx *ynab.Transaction, scheduled bool) bool {
	txnAccount, ok := accountMap[tx.AccountID]
	if !ok {
		panic("unknown account: " + txnAccount.ID + " " + txnAccount.Name)
	}
	if txnAccount.OnBudget == false {
		return false
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
		if transferAccount == nil || transferAccount.OnBudget == false ||
			// For scheduled transfers we only see one side of the transaction
			// - cash => credit transfers in the past get caught below but we need
			// to catch scheduled ones here also.
			(scheduled && !transferAccount.CashBacked()) {
			// cash direct spending, or transfer to off budget account
			return tx.Amount < 0
		}
		// cash <> cash transfer is just moving money around, not an outflow
		// the bank account side of a credit card transfer is ignored, we count
		// the credit card transfer inflow instead.
		return false
	}
	// transaction account is not cash backed:
	if transferAccount == nil {
		// credit card spending or similar outflow.
		return false
	}
	if !transferAccount.CashBacked() {
		// not cash backed and transfer account is not cash backed, e.g.
		// transfer from mortgage account to escrow account.
		return false
	}
	if tx.Amount >= 0 {
		// transfer from cash backed account to credit account, reverse the
		// amount and count it as an outflow.
		tx.Amount = -1 * tx.Amount
		return true
	}
	return false
}

func main() {
	debug := flag.Bool("debug", false, "Enable debug")
	file := flag.String("file", "", "Filename to read txns from")
	budgetName := flag.String("budget-name", "", "Name of the budget to compute AOM for")
	includeScheduledIncome := flag.Bool("include-scheduled-income", false, "Include scheduled income")
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
		if *debug {
			fmt.Println("account", account.ID, account.Name, account.Type, account.Note, "on budget:", account.OnBudget)
		}
		accountMap[account.ID] = account
	}
	scheduledTxns, err := getScheduledTransactions(client, thisBudget.ID)
	if err != nil {
		log.Fatal(err)
	}
	sort.Slice(scheduledTxns, func(i, j int) bool {
		return time.Time(scheduledTxns[i].DateNext).Before(time.Time(scheduledTxns[j].DateNext))
	})
	var txns []*ynab.Transaction

	if *file != "" {
		data, err := ioutil.ReadFile(*file)
		switch {
		case os.IsNotExist(err):
			txns, err = getTransactions(client, thisBudget.ID)
			if err != nil {
				log.Fatal(err)
			}
		case err != nil:
			log.Fatal(err)
		default:
			transactionResp := new(ynab.TransactionListResponse)
			if err := json.Unmarshal(data, &transactionResp); err != nil {
				log.Fatal(err)
			}
			txns = transactionResp.Data.Transactions
		}
	} else {
		txns, err = getTransactions(client, thisBudget.ID)
		if err != nil {
			log.Fatal(err)
		}
	}

	buckets := make([]*ynab.Transaction, 0)
	for i := range txns {
		tx := txns[i]
		txnAccount, ok := accountMap[tx.AccountID]
		if !ok {
			panic("unknown account: " + tx.AccountID)
		}
		if txnAccount.OnBudget == false {
			continue
		}
		if !txnAccount.CashBacked() {
			continue
		}
		if tx.Amount <= 0 {
			// should be counted in spending.
			continue
		}
		if tx.TransferAccountID.Valid {
			transferAccount, ok := accountMap[tx.TransferAccountID.String]
			if !ok {
				panic("could not find transfer acct id: " + tx.TransferAccountID.String)
			}
			// transfers from off budget accounts are income, on budget, they
			// are just moving money around
			if transferAccount.OnBudget == true {
				continue
			}
		}
		buckets = append(buckets, tx)
		continue
	}
	cumEarned := int64(0)
	for i := range buckets {
		cumEarned += buckets[i].Amount
		if *debug {
			fmt.Println("income:", buckets[i].Date.String(), amt(cumEarned), amt(buckets[i].Amount), buckets[i].AccountName, buckets[i].PayeeName)
		}
	}
	spending := make([]*ynab.Transaction, 0)
	for i := range txns {
		tx := txns[i]
		if isOutflow(accountMap, tx, false) {
			spending = append(spending, tx)
		}

	}
	cumSpent := int64(0)
	for i := range spending {
		cumSpent += spending[i].Amount
	}
	if *debug {
		fmt.Println("budget difference", amt(cumEarned+cumSpent))
	}
	if len(buckets) == 0 {
		log.Fatal("Can't generate age of money without any money!")
	}
	// Now that we have inflow buckets and outflows, match up spending to the
	// bucket we spent it in.
	currentBucketIdx := 0
	// Amount of money that's been spent from the current bucket.
	bucketSpend := int64(0)
	for i := range spending {
		amount := -1 * spending[i].Amount
		if amount == 0 {
			continue
		}
		if amount < 0 {
			panic("spending less than zero")
		}
		for amount > 0 {
			if amount < buckets[currentBucketIdx].Amount-bucketSpend {
				bucketSpend += amount
				amount = 0
				continue
			} else {
				// exhaust this bucket
				amount -= buckets[currentBucketIdx].Amount - bucketSpend
				currentBucketIdx++
				bucketSpend = 0
			}
		}
		ageHours := time.Time(spending[i].Date).Sub(time.Time(buckets[currentBucketIdx].Date)).Hours()
		ageOfMoney := int(math.Round(float64(ageHours) / 24))
		fmt.Printf("%3d Earned: %s Spent: %s %10s %s %s\n",
			ageOfMoney, buckets[currentBucketIdx].Date.String(),
			spending[i].Date.String(), "$"+amt(-1*spending[i].Amount),
			spending[i].AccountName, clean(spending[i].PayeeName))
	}
	fmt.Println("")
	fmt.Println("Upcoming spending thresholds (and age if you spent today):")
	fmt.Println("==========================================================")
	threshold := int64(0)
	for i := currentBucketIdx; (i-currentBucketIdx < 25 && threshold <= 20000*1000) && i < len(buckets); i++ {
		ageHours := time.Since(time.Time(buckets[i].Date)).Hours()
		ageDays := int(math.Round(float64(ageHours)/24)) - 1
		if i == currentBucketIdx {
			threshold += buckets[i].Amount - bucketSpend
		} else {
			threshold += buckets[i].Amount
		}
		fmt.Printf("%3d %s %10s %s %s\n", ageDays, buckets[i].Date.String(), "$"+amt(threshold), buckets[i].AccountName, clean(buckets[i].PayeeName))
	}
	if len(scheduledTxns) == 0 {
		return
	}
	fmt.Println("")
	fmt.Println("Projected age of scheduled transactions:")
	fmt.Println("========================================")
	for i := range scheduledTxns {
		// TODO the duplication is not great here.

		// lazy type hack
		txnIsh := &ynab.Transaction{
			AccountID:         scheduledTxns[i].AccountID,
			Amount:            scheduledTxns[i].Amount,
			TransferAccountID: scheduledTxns[i].TransferAccountID,
			Date:              scheduledTxns[i].DateNext,
			PayeeName:         scheduledTxns[i].PayeeName,
			Memo:              scheduledTxns[i].Memo,
		}

		if !isOutflow(accountMap, txnIsh, true) {
			if *includeScheduledIncome == false {
				continue
			}
			txnAccount, ok := accountMap[txnIsh.AccountID]
			if !ok {
				panic("unknown account: " + txnIsh.AccountID)
			}
			if txnIsh.TransferAccountID.Valid {
				transferAccount, ok := accountMap[txnIsh.TransferAccountID.String]
				if !ok {
					panic("could not find transfer acct id: " + txnIsh.TransferAccountID.String)
				}
				if !txnAccount.CashBacked() && txnIsh.Amount < 0 && transferAccount.OnBudget {
					// transfer from off budget to on budget, this is valid
					txnIsh.Amount = txnIsh.Amount * -1
					buckets = append(buckets, txnIsh)
					continue
				}
				// transfers from off budget accounts are income, on budget, they
				// are just moving money around
				if txnAccount.CashBacked() && transferAccount.OnBudget == true {
					continue
				}
			}
			if !txnAccount.CashBacked() {
				continue
			}
			buckets = append(buckets, txnIsh)
			continue
		}
		amount := -1 * txnIsh.Amount
		if amount == 0 {
			continue
		}
		if amount < 0 {
			panic("spending less than zero")
		}
		for amount > 0 {
			if currentBucketIdx >= len(buckets) {
				break
			}
			if amount < buckets[currentBucketIdx].Amount-bucketSpend {
				bucketSpend += amount
				amount = 0
				continue
			} else {
				// exhaust this bucket
				amount -= buckets[currentBucketIdx].Amount - bucketSpend
				currentBucketIdx++
				bucketSpend = 0
			}
		}
		if currentBucketIdx >= len(buckets) {
			//          113 Earned: 2019-07-25 Spend on: 2019-11-15
			fmt.Printf("N/A Not earned yet.    Spend on: %s %10s %s %s\n",
				scheduledTxns[i].DateNext.String(), "$"+amt(-1*txnIsh.Amount),
				scheduledTxns[i].AccountName, clean(scheduledTxns[i].PayeeName))
			break
		}
		ageHours := time.Time(scheduledTxns[i].DateNext).Sub(time.Time(buckets[currentBucketIdx].Date)).Hours()
		ageOfMoney := int(math.Round(float64(ageHours) / 24))
		fmt.Printf("%3d Earned: %s Spend on: %s %10s %s %s\n",
			ageOfMoney, buckets[currentBucketIdx].Date.String(),
			scheduledTxns[i].DateNext.String(), "$"+amt(-1*txnIsh.Amount),
			scheduledTxns[i].AccountName, clean(scheduledTxns[i].PayeeName))
	}
}

func clean(payee string) string {
	return strings.Replace(payee, "Transfer :", "Transfer:", -1)
}

func amt(amount int64) string {
	return strconv.FormatFloat(float64(amount)/1000, 'f', 2, 64)
}
