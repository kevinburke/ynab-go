// The ynab-export-transactions command retrieves transactions and prints them
// to stdout in CSV format. Use the --start or --category arguments to filter
// the list of transactions returned by the program. Use --budget <budget_name>
// to specify a budget.
//
// Set YNAB_TOKEN in your environment with your API token to configure the
// client.
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/kevinburke/ynab-go"
)

func getBudgets(ctx context.Context, client *ynab.Client) ([]*ynab.Budget, error) {
	budgetResp, err := client.Budgets.GetPage(ctx, url.Values{})
	if err != nil {
		return nil, err
	}
	return budgetResp.Data.Budgets, nil
}

func getTransactions(client *ynab.Client, budgetID string, data url.Values) ([]*ynab.Transaction, error) {
	transactionResp, err := client.Budgets.GetTransactions(context.TODO(), budgetID, data)
	if err != nil {
		return nil, err
	}
	return transactionResp.Data.Transactions, nil
}

func getCategories(client *ynab.Client, budgetID string, data url.Values) ([]*ynab.CategoryGroup, error) {
	categoryResp, err := client.Budgets.GetCategories(context.TODO(), budgetID, data)
	if err != nil {
		return nil, err
	}
	return categoryResp.Data.CategoryGroups, nil
}

func main() {
	budgetName := flag.String("budget-name", "", "Name of the budget to export transactions for")
	category := flag.String("category", "", "Category to filter for")
	start := flag.String("start", "", "Start time (parsed as "+time.RFC3339+")")
	flag.Parse()
	token, ok := os.LookupEnv("YNAB_TOKEN")
	if !ok {
		log.Fatal("please set YNAB_TOKEN in the environment: https://app.youneedabudget.com/settings")
	}
	client := ynab.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	budgets, err := getBudgets(ctx, client)
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
	categories, err := getCategories(client, thisBudget.ID, url.Values{})
	if err != nil {
		log.Fatal(err)
	}
	groupMap := make(map[string][]string)
	categoryMap := make(map[string]string)
	for _, group := range categories {
		groupMap[group.Name] = make([]string, len(group.Categories))
		for i, category := range group.Categories {
			groupMap[group.Name][i] = category.Name
			if !group.Hidden {
				categoryMap[category.Name] = group.Name
			}
		}
	}
	data := url.Values{}
	if *start != "" {
		startTime, err := time.Parse(time.RFC3339, *start)
		if err != nil {
			log.Fatal(err)
		}
		data.Set("since_date", startTime.Format("2006-01-02"))
	}
	txns, err := getTransactions(client, thisBudget.ID, data)
	if err != nil {
		log.Fatal(err)
	}
	w := csv.NewWriter(os.Stdout)
	werr := w.Write([]string{"Account", "Flag", "Date", "Payee", "Category Group/Category", "Category Group", "Category", "Memo", "Outflow", "Inflow", "Cleared"})
	if werr != nil {
		log.Fatal(werr)
	}
	for _, txn := range txns {
		var outflow, inflow string
		if txn.Amount < 0 {
			outflow = strconv.FormatFloat(-1*float64(txn.Amount)/1000, 'f', 2, 64)
		} else {
			inflow = strconv.FormatFloat(float64(txn.Amount)/1000, 'f', 2, 64)
		}
		cgroup := categoryMap[txn.CategoryName.String]
		if *category == "" || txn.CategoryName.String == *category || cgroup == *category {
			w.Write([]string{txn.AccountName, "", txn.Date.String(), txn.PayeeName, "", cgroup, txn.CategoryName.String, txn.Memo, outflow, inflow, txn.Cleared})
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}
