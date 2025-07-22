package ynab

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kevinburke/go-types"
	"github.com/kevinburke/rest/restclient"
)

type Client struct {
	*restclient.Client

	Budgets func(budgetID string) *BudgetService
}

type TransactionListResponse struct {
	Data TransactionListWrapper `json:"data"`
}

type TransactionResponse struct {
	Data TransactionWrapper `json:"data"`
}

type TransactionWrapper struct {
	Transaction *Transaction `json:"transaction"`
}

type CreateTransactionRequest struct {
	Transaction *NewTransaction `json:"transaction"`
}

type NewTransaction struct {
	AccountID       string               `json:"account_id"`
	Date            Date                 `json:"date"`
	Amount          int64                `json:"amount"`
	PayeeID         types.NullString     `json:"payee_id,omitempty"`
	PayeeName       types.NullString     `json:"payee_name,omitempty"`
	CategoryID      types.NullString     `json:"category_id,omitempty"`
	Memo            types.NullString     `json:"memo,omitempty"`
	Cleared         ClearedStatus        `json:"cleared,omitempty"`
	Approved        bool                 `json:"approved"` // Defaults to false if not specified
	FlagColor       FlagColor            `json:"flag_color,omitempty"`
	Subtransactions []*NewSubTransaction `json:"subtransactions,omitempty"`
	ImportID        types.NullString     `json:"import_id,omitempty"`
}

type NewSubTransaction struct {
	Amount     int64            `json:"amount"`
	PayeeID    types.NullString `json:"payee_id,omitempty"`
	PayeeName  types.NullString `json:"payee_name,omitempty"`
	CategoryID types.NullString `json:"category_id,omitempty"`
	Memo       types.NullString `json:"memo,omitempty"`
}

type CreateTransactionResponse struct {
	Data CreateTransactionData `json:"data"`
}

type CreateTransactionData struct {
	TransactionIDs  []string     `json:"transaction_ids"`
	Transaction     *Transaction `json:"transaction,omitempty"`
	ServerKnowledge int64        `json:"server_knowledge"`
}

type UpdateTransactionRequest struct {
	Transaction *UpdateTransaction `json:"transaction"`
}

type UpdateTransaction struct {
	AccountID       *string           `json:"account_id,omitempty"`
	Date            Date              `json:"date"`
	Amount          *int64            `json:"amount,omitempty"`
	PayeeID         types.NullString  `json:"payee_id,omitempty"`
	PayeeName       types.NullString  `json:"payee_name,omitempty"`
	CategoryID      types.NullString  `json:"category_id,omitempty"`
	Memo            types.NullString  `json:"memo,omitempty"`
	Cleared         types.NullString  `json:"cleared,omitempty"`
	Approved        *bool             `json:"approved,omitempty"`
	FlagColor       types.NullString  `json:"flag_color,omitempty"`
	Subtransactions []*SubTransaction `json:"subtransactions,omitempty"`
}

type SubTransaction struct {
	Amount     int64            `json:"amount"`
	PayeeID    types.NullString `json:"payee_id,omitempty"`
	PayeeName  types.NullString `json:"payee_name,omitempty"`
	CategoryID types.NullString `json:"category_id,omitempty"`
	Memo       types.NullString `json:"memo,omitempty"`
}

type CategoryListResponse struct {
	Data CategoryListWrapper `json:"data"`
}

type CategoryListWrapper struct {
	CategoryGroups []*CategoryGroup `json:"category_groups"`
}

type CategoryGroup struct {
	ID         string
	Name       string
	Hidden     bool
	Deleted    bool
	Categories []*Category
}

type Category struct {
	ID              string
	Name            string
	CategoryGroupID string `json:"category_group_id"`
	Note            string
	Hidden          bool
	Budgeted        int64
	Activity        int64
}

type TransactionListWrapper struct {
	Transactions []*Transaction `json:"transactions"`
}

type ScheduledTransactionListResponse struct {
	Data ScheduledTransactionListWrapper `json:"data"`
}

type ScheduledTransactionListWrapper struct {
	ScheduledTransactions []*ScheduledTransaction `json:"scheduled_transactions"`
}

type Date time.Time

func (t *Date) UnmarshalJSON(b []byte) error {
	t2, err := time.ParseInLocation(`"2006-01-02"`, string(b), time.Local)
	if err != nil {
		return err
	}
	*t = Date(t2)
	return nil
}

func (t Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format("2006-01-02") + `"`), nil
}

func (t Date) String() string {
	return time.Time(t).Format("2006-01-02")
}

func (t Date) GoString() string {
	return time.Time(t).GoString()
}

// A NullDate is a Date that may be null.
type NullDate struct {
	Valid bool
	Date  Date
}

func (nt *NullDate) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		nt.Valid = false
		return nil
	}
	var d Date
	err := json.Unmarshal(b, &d)
	if err != nil {
		return err
	}
	nt.Valid = true
	nt.Date = d
	return nil
}

func (nt NullDate) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	b, err := json.Marshal(nt.Date)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

type ScheduledTransaction struct {
	AccountID         string `json:"account_id"`
	AccountName       string `json:"account_name"`
	Amount            int64
	Approved          bool
	CategoryName      types.NullString `json:"category_name"`
	Cleared           string
	DateFirst         Date `json:"date_first"`
	DateNext          Date `json:"date_next"`
	Deleted           bool
	FlagColor         FlagColor `json:"flag_color"`
	Frequency         string
	ID                string `json:"id"`
	Memo              string
	PayeeName         string           `json:"payee_name"`
	TransferAccountID types.NullString `json:"transfer_account_id"`
	Subtransactions   []Transaction    `json:"subtransactions"`
}

type Transaction struct {
	AccountID             string `json:"account_id"`
	AccountName           string `json:"account_name"`
	Amount                int64
	Approved              bool
	CategoryID            types.NullString `json:"category_id"`
	CategoryName          types.NullString `json:"category_name"`
	Cleared               string
	Date                  Date
	Deleted               bool
	FlagColor             FlagColor `json:"flag_color"`
	ID                    string    `json:"id"`
	Memo                  string
	PayeeName             string           `json:"payee_name"`
	TransferAccountID     types.NullString `json:"transfer_account_id"`
	TransferTransactionID types.NullString `json:"transfer_transaction_id"`
	MatchedTransactionID  types.NullString `json:"matched_transaction_id"`
	Subtransactions       []Transaction    `json:"subtransactions"`
}

// ClearedStatus represents the cleared status of a transaction
type ClearedStatus string

const (
	ClearedStatusCleared    ClearedStatus = "cleared"
	ClearedStatusUncleared  ClearedStatus = "uncleared"
	ClearedStatusReconciled ClearedStatus = "reconciled"
)

// FlagColor represents the available flag colors for transactions
type FlagColor string

const (
	FlagColorRed    FlagColor = "red"
	FlagColorOrange FlagColor = "orange"
	FlagColorYellow FlagColor = "yellow"
	FlagColorGreen  FlagColor = "green"
	FlagColorBlue   FlagColor = "blue"
	FlagColorPurple FlagColor = "purple"
	FlagColorEmpty  FlagColor = ""
)

func (fc *FlagColor) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*fc = FlagColorEmpty
		return nil
	}
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*fc = FlagColor(s)
	return nil
}

func (fc FlagColor) MarshalJSON() ([]byte, error) {
	if fc == FlagColorEmpty {
		return []byte("null"), nil
	}
	return json.Marshal(string(fc))
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

func (a Account) CashBacked() bool {
	return a.Type == "cash" || a.Type == "savings" || a.Type == "checking"
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

func (c *Client) GetBudgets(ctx context.Context, data url.Values) (*BudgetListResponse, error) {
	req, err := c.NewRequestWithContext(ctx, "GET", "/budgets?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	budgetResp := new(BudgetListResponse)
	if err := c.Do(req, budgetResp); err != nil {
		return nil, err
	}
	return budgetResp, nil
}

func (b *BudgetService) Accounts(ctx context.Context, data url.Values) (*AccountListResponse, error) {
	req, err := b.client.NewRequestWithContext(ctx, "GET", "/budgets/"+b.id+"/accounts?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	accountResp := new(AccountListResponse)
	if err := b.client.Do(req, accountResp); err != nil {
		return nil, err
	}
	return accountResp, nil
}

func (b *BudgetService) Transactions(ctx context.Context, data url.Values) (*TransactionListResponse, error) {
	req, err := b.client.NewRequestWithContext(ctx, "GET", "/budgets/"+b.id+"/transactions?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	transactionResp := new(TransactionListResponse)
	if err := b.client.Do(req, transactionResp); err != nil {
		return nil, err
	}
	return transactionResp, nil
}

func (b *BudgetService) ScheduledTransactions(ctx context.Context, data url.Values) (*ScheduledTransactionListResponse, error) {
	req, err := b.client.NewRequestWithContext(ctx, "GET", "/budgets/"+b.id+"/scheduled_transactions?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	transactionResp := new(ScheduledTransactionListResponse)
	if err := b.client.Do(req, transactionResp); err != nil {
		return nil, err
	}
	return transactionResp, nil
}

func (b *BudgetService) Categories(ctx context.Context, data url.Values) (*CategoryListResponse, error) {
	req, err := b.client.NewRequestWithContext(ctx, "GET", "/budgets/"+b.id+"/categories?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	categoryResp := new(CategoryListResponse)
	if err := b.client.Do(req, categoryResp); err != nil {
		return nil, err
	}
	return categoryResp, nil
}

type BudgetService struct {
	client *Client
	// the budget ID
	id string
}

func (c *Client) PutResource(ctx context.Context, pathPart string, sid string, req interface{}, resp interface{}) error {
	sidPart := strings.Join([]string{pathPart, sid}, "/")
	return c.MakeRequest(ctx, "PUT", sidPart, nil, req, resp)
}

func (b *BudgetService) CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*CreateTransactionResponse, error) {
	resp := new(CreateTransactionResponse)
	err := b.client.MakeRequest(ctx, "POST", "/budgets/"+b.id+"/transactions", nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *BudgetService) UpdateTransaction(ctx context.Context, transactionID string, req *UpdateTransactionRequest) (*TransactionResponse, error) {
	resp := new(TransactionResponse)
	err := b.client.PutResource(ctx, "/budgets/"+b.id+"/transactions", transactionID, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) MakeRequest(ctx context.Context, method string, pathPart string, data url.Values, reqBody interface{}, v interface{}) error {
	var rb io.Reader
	if reqBody != nil || (method == "POST" || method == "PUT") {
		reqBodyJSON, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		rb = bytes.NewReader(reqBodyJSON)
	}
	if method == "GET" && data != nil {
		pathPart = pathPart + "?" + data.Encode()
	}
	req, err := c.NewRequestWithContext(ctx, method, pathPart, rb)
	if err != nil {
		return err
	}
	if reqBody != nil && (method == "POST" || method == "PUT") {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	return c.Do(req, &v)
}

const Version = "1.2.0"

func (c *Client) NewRequestWithContext(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := c.Client.NewRequestWithContext(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ynab-go/"+Version+" "+req.Header.Get("User-Agent"))
	return req, nil
}

func NewClient(token string) *Client {
	client := restclient.NewBearerClient(token, "https://api.youneedabudget.com/v1")
	c := &Client{Client: client}
	c.Budgets = func(id string) *BudgetService {
		return &BudgetService{
			client: c,
			id:     id,
		}
	}
	return c
}
