package ynab

import (
	"context"
	"net/url"
	"time"

	"github.com/kevinburke/go-types"
	"github.com/kevinburke/rest"
)

type Client struct {
	*rest.Client

	Accounts     *AccountService
	Budgets      *BudgetService
	Transactions *TransactionService
}

type TransactionListResponse struct {
	Data TransactionListWrapper `json:"data"`
}

type TransactionListWrapper struct {
	Transactions []*Transaction `json:"transactions"`
}

type Date time.Time

func (t *Date) UnmarshalJSON(b []byte) error {
	t2, err := time.Parse(`"2006-01-02"`, string(b))
	if err != nil {
		return err
	}
	*t = Date(t2)
	return nil
}

func (t Date) String() string {
	return time.Time(t).Format("2006-01-02")
}

type Transaction struct {
	AccountID             string `json:"account_id"`
	AccountName           string `json:"account_name"`
	Amount                int64
	Approved              bool
	CategoryName          types.NullString `json:"category_name"`
	Cleared               string
	Date                  Date
	Deleted               bool
	ID                    string `json:"id"`
	Memo                  string
	PayeeName             string           `json:"payee_name"`
	TransferAccountID     types.NullString `json:"transfer_account_id"`
	TransferTransactionID types.NullString `json:"transfer_transaction_id"`
	MatchedTransactionID  types.NullString `json:"matched_transaction_id"`
	Subtransactions       []Transaction    `json:"subtransactions"`
}

type AccountService struct {
	client *Client
}
type Account struct {
	ID              string
	Name            string
	Type            string
	OnBudget        bool `json:"on_budget"`
	Closed          bool
	Note            string
	Balance         int64
	StartingBalance int64 `json:"starting_balance"`
	Deleted         bool
}
type AccountListResponse struct {
	Data AccountListWrapper `json:"data"`
}

type AccountListWrapper struct {
	Accounts []*Account `json:"accounts"`
}

type Budget struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type BudgetListResponse struct {
	Data BudgetListWrapper `json:"data"`
}

type BudgetListWrapper struct {
	Budgets []*Budget `json:"budgets"`
}

func (b *BudgetService) GetPage(ctx context.Context, data url.Values) (*BudgetListResponse, error) {
	req, err := b.client.NewRequest("GET", "/budgets?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	budgetResp := new(BudgetListResponse)
	if err := b.client.Do(req, budgetResp); err != nil {
		return nil, err
	}
	return budgetResp, nil
}

func (b *BudgetService) GetAccounts(ctx context.Context, budgetID string, data url.Values) (*AccountListResponse, error) {
	req, err := b.client.NewRequest("GET", "/budgets/"+budgetID+"/accounts?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	accountResp := new(AccountListResponse)
	if err := b.client.Do(req, accountResp); err != nil {
		return nil, err
	}
	return accountResp, nil
}

func (b *BudgetService) GetTransactions(ctx context.Context, budgetID string, data url.Values) (*TransactionListResponse, error) {
	req, err := b.client.NewRequest("GET", "/budgets/"+budgetID+"/transactions?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	transactionResp := new(TransactionListResponse)
	if err := b.client.Do(req, transactionResp); err != nil {
		return nil, err
	}
	return transactionResp, nil
}

type BudgetService struct {
	client *Client
}
type TransactionService struct {
	client *Client
}

func NewClient(token string) *Client {
	client := rest.NewBearerClient(token, "https://api.youneedabudget.com/v1")
	c := &Client{Client: client}
	c.Accounts = &AccountService{
		client: c,
	}
	c.Budgets = &BudgetService{
		client: c,
	}
	c.Transactions = &TransactionService{
		client: c,
	}
	return c
}
