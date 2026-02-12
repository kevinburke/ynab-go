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

	userAgent string
	Budgets   func(budgetID string) *BudgetService
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
	AccountID  string           `json:"account_id"`
	Date       Date             `json:"date"`   // The transaction date in ISO format (e.g. 2016-12-01). Future dates (scheduled transactions) are not permitted.
	Amount     int64            `json:"amount"` // The transaction amount in milliunits format
	PayeeID    types.NullString `json:"payee_id,omitempty"`
	PayeeName  types.NullString `json:"payee_name,omitempty"`  // If provided and payee_id is null, used to resolve the payee by matching rename rule, same name, or creation of a new payee.
	CategoryID types.NullString `json:"category_id,omitempty"` // To configure a split, specify null and provide a subtransactions array. Credit Card Payment categories are not permitted.
	Memo       types.NullString `json:"memo,omitempty"`
	Cleared    ClearedStatus    `json:"cleared,omitempty"`
	Approved   bool             `json:"approved"` // Whether or not the transaction is approved. If not supplied, transaction will be unapproved by default.
	FlagColor  FlagColor        `json:"flag_color,omitempty"`
	// An array of subtransactions to configure a transaction as a split. Updating subtransactions on an existing split transaction is not supported.
	Subtransactions   []*NewSubTransaction `json:"subtransactions,omitempty"`
	ImportID          types.NullString     `json:"import_id,omitempty"`
	TransferAccountID types.NullString     `json:"transfer_account_id,omitempty"`
}

type NewSubTransaction struct {
	Amount     int64            `json:"amount"` // The subtransaction amount in milliunits format
	PayeeID    types.NullString `json:"payee_id,omitempty"`
	PayeeName  types.NullString `json:"payee_name,omitempty"`  // If provided and payee_id is null, used to resolve the payee by matching rename rule, same name, or creation of a new payee.
	CategoryID types.NullString `json:"category_id,omitempty"` // Credit Card Payment categories are not permitted.
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
	AccountID  *string          `json:"account_id,omitempty"`
	Date       Date             `json:"date"`                  // The transaction date in ISO format (e.g. 2016-12-01). Split transaction dates cannot be changed.
	Amount     *int64           `json:"amount,omitempty"`      // The transaction amount in milliunits format. Split transaction amounts cannot be changed.
	PayeeID    types.NullString `json:"payee_id,omitempty"`    // To create a transfer, use the account transfer payee pointing to the target account.
	PayeeName  types.NullString `json:"payee_name,omitempty"`  // If provided and payee_id is null, used to resolve the payee by matching rename rule, same name, or creation of a new payee.
	CategoryID types.NullString `json:"category_id,omitempty"` // Credit Card Payment categories are not permitted.
	Memo       types.NullString `json:"memo,omitempty"`
	Cleared    types.NullString `json:"cleared,omitempty"`
	Approved   *bool            `json:"approved,omitempty"` // Whether or not the transaction is approved.
	FlagColor  types.NullString `json:"flag_color,omitempty"`
	// An array of subtransactions to configure a transaction as a split. Updating subtransactions on an existing split transaction is not supported.
	Subtransactions []*SubTransaction `json:"subtransactions,omitempty"`
}

type SubTransaction struct {
	Amount     int64            `json:"amount"` // The subtransaction amount in milliunits format
	PayeeID    types.NullString `json:"payee_id,omitempty"`
	PayeeName  types.NullString `json:"payee_name,omitempty"`
	CategoryID types.NullString `json:"category_id,omitempty"` // Credit Card Payment categories are not permitted.
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
	Budgeted        int64 // Budgeted amount in milliunits format
	Activity        int64 // Activity amount in milliunits format
	Balance         int64 // Balance in milliunits format
}

// UpdateMonthCategoryRequest is the request body for updating a category's budget for a month.
type UpdateMonthCategoryRequest struct {
	Category SaveMonthCategory `json:"category"`
}

// SaveMonthCategory contains the budgeted amount to set for a category in a month.
type SaveMonthCategory struct {
	Budgeted int64 `json:"budgeted"` // Budgeted amount in milliunits format
}

// SaveCategoryResponse is the response from updating a category's budget.
type SaveCategoryResponse struct {
	Data SaveCategoryData `json:"data"`
}

// SaveCategoryData contains the updated category and server knowledge.
type SaveCategoryData struct {
	Category        *Category `json:"category"`
	ServerKnowledge int64     `json:"server_knowledge"`
}

// CategoryResponse is the response from getting a single category.
type CategoryResponse struct {
	Data CategoryData `json:"data"`
}

// CategoryData contains the category.
type CategoryData struct {
	Category *Category `json:"category"`
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
	Amount            int64  // The scheduled transaction amount in milliunits format
	Approved          bool
	CategoryName      types.NullString `json:"category_name"`
	Cleared           string
	DateFirst         Date `json:"date_first"` // The first date for which the Scheduled Transaction was scheduled.
	DateNext          Date `json:"date_next"`  // The next date for which the Scheduled Transaction is scheduled.
	Deleted           bool
	FlagColor         FlagColor `json:"flag_color"`
	Frequency         string
	ID                string `json:"id"`
	Memo              string
	PayeeName         string           `json:"payee_name"`
	TransferAccountID types.NullString `json:"transfer_account_id"` // If a transfer, the account_id which the scheduled transaction transfers to
	Subtransactions   []Transaction    `json:"subtransactions"`
}

type Transaction struct {
	AccountID             string           `json:"account_id"`
	AccountName           string           `json:"account_name"`
	Amount                int64            // The transaction amount in milliunits format
	Approved              bool             // Whether or not the transaction is approved
	CategoryID            types.NullString `json:"category_id"`
	CategoryName          types.NullString `json:"category_name"` // If a split transaction, this will be 'Split'.
	Cleared               ClearedStatus
	Date                  Date // The transaction date in ISO format (e.g. 2016-12-01)
	Deleted               bool
	FlagColor             FlagColor `json:"flag_color"`
	ID                    string    `json:"id"`
	Memo                  string
	PayeeName             string           `json:"payee_name"`
	TransferAccountID     types.NullString `json:"transfer_account_id"`     // If a transfer transaction, the account to which it transfers
	TransferTransactionID types.NullString `json:"transfer_transaction_id"` // If a transfer transaction, the id of transaction on the other side of the transfer
	MatchedTransactionID  types.NullString `json:"matched_transaction_id"`  // If transaction is matched, the id of the matched transaction
	Subtransactions       []Transaction    `json:"subtransactions"`         // If a split transaction, the subtransactions.
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
	OnBudget        bool `json:"on_budget"` // Whether this account is on budget or not
	Closed          bool // Whether this account is closed or not
	Note            string
	Balance         int64 // The current balance of the account in milliunits format
	StartingBalance int64 `json:"starting_balance"`
	Deleted         bool
	TransferPayeeID types.NullString `json:"transfer_payee_id"` // The payee id which should be used when transferring to this account
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

// MonthDetail represents a budget month with all its categories.
type MonthDetail struct {
	Month      string      `json:"month"`
	Note       string      `json:"note"`
	Income     int64       `json:"income"`         // The total amount of transactions categorized to 'Inflow: Ready to Assign' in the month
	Budgeted   int64       `json:"budgeted"`       // The total amount budgeted in the month
	Activity   int64       `json:"activity"`       // The total amount of transactions in the month, excluding those categorized to 'Inflow: Ready to Assign'
	ToBeBudget int64       `json:"to_be_budgeted"` // The available amount for 'Ready to Assign'
	AgeOfMoney int         `json:"age_of_money"`   // The Age of Money as of the month
	Deleted    bool        `json:"deleted"`
	Categories []*Category `json:"categories"` // Amounts (budgeted, activity, balance, etc.) are specific to this month.
}

// BudgetDetail represents a full budget export with all related entities.
type BudgetDetail struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	LastModifiedOn string         `json:"last_modified_on"` // The last time any changes were made to the budget from either a web or mobile client
	DateFormat     interface{}    `json:"date_format"`
	CurrencyFormat interface{}    `json:"currency_format"`
	Accounts       []*Account     `json:"accounts"`
	Categories     []*Category    `json:"categories"`
	Months         []*MonthDetail `json:"months"`
}

// BudgetDetailResponse wraps the budget detail response
type BudgetDetailResponse struct {
	Data struct {
		Budget          *BudgetDetail `json:"budget"`
		ServerKnowledge int64         `json:"server_knowledge"`
	} `json:"data"`
}

// GetBudget returns a full budget export with all months and categories.
// This is more efficient than making per-month API calls.
func (b *BudgetService) GetBudget(ctx context.Context) (*BudgetDetailResponse, error) {
	req, err := b.client.NewRequestWithContext(ctx, "GET", "/budgets/"+b.id, nil)
	if err != nil {
		return nil, err
	}
	budgetResp := new(BudgetDetailResponse)
	if err := b.client.Do(req, budgetResp); err != nil {
		return nil, err
	}
	return budgetResp, nil
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

// GetMonthCategory retrieves a category for a specific budget month.
// The month should be in ISO format (e.g., "2024-01-01") or "current" for the current month.
// Amounts (budgeted, activity, balance) are specific to the requested month.
func (b *BudgetService) GetMonthCategory(ctx context.Context, month string, categoryID string) (*CategoryResponse, error) {
	resp := new(CategoryResponse)
	path := "/budgets/" + b.id + "/months/" + month + "/categories/" + categoryID
	err := b.client.MakeRequest(ctx, "GET", path, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateMonthCategory updates the budgeted amount for a category in a specific month.
// The month should be in ISO format (e.g., "2024-01-01") or "current" for the current month.
// The budgeted amount is in milliunits (e.g., $50.00 = 50000).
func (b *BudgetService) UpdateMonthCategory(ctx context.Context, month string, categoryID string, budgeted int64) (*SaveCategoryResponse, error) {
	req := &UpdateMonthCategoryRequest{
		Category: SaveMonthCategory{Budgeted: budgeted},
	}
	resp := new(SaveCategoryResponse)
	path := "/budgets/" + b.id + "/months/" + month + "/categories/" + categoryID
	err := b.client.MakeRequest(ctx, "PATCH", path, nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// User represents a YNAB user.
type User struct {
	ID string `json:"id"`
}

// UserResponse wraps the user response.
type UserResponse struct {
	Data struct {
		User *User `json:"user"`
	} `json:"data"`
}

// DateFormat represents the date format setting for a budget.
type DateFormat struct {
	Format string `json:"format"`
}

// CurrencyFormat represents the currency format setting for a budget.
type CurrencyFormat struct {
	ISOCode          string `json:"iso_code"`
	ExampleFormat    string `json:"example_format"`
	DecimalDigits    int    `json:"decimal_digits"`
	DecimalSeparator string `json:"decimal_separator"`
	SymbolFirst      bool   `json:"symbol_first"`
	GroupSeparator   string `json:"group_separator"`
	CurrencySymbol   string `json:"currency_symbol"`
	DisplaySymbol    bool   `json:"display_symbol"`
}

// BudgetSettings represents the settings for a budget.
type BudgetSettings struct {
	DateFormat     DateFormat     `json:"date_format"`
	CurrencyFormat CurrencyFormat `json:"currency_format"`
}

// BudgetSettingsResponse wraps the budget settings response.
type BudgetSettingsResponse struct {
	Data struct {
		Settings BudgetSettings `json:"settings"`
	} `json:"data"`
}

// AccountResponse wraps a single account response.
type AccountResponse struct {
	Data struct {
		Account *Account `json:"account"`
	} `json:"data"`
}

// SaveAccount represents the data for creating a new account.
type SaveAccount struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Balance int64  `json:"balance"` // The current balance of the account in milliunits format
}

// CreateAccountRequest is the request body for creating an account.
type CreateAccountRequest struct {
	Account *SaveAccount `json:"account"`
}

// SaveCategory represents the data for updating a category.
type SaveCategory struct {
	Name            string `json:"name,omitempty"`
	Note            string `json:"note,omitempty"`
	CategoryGroupID string `json:"category_group_id,omitempty"`
	GoalTarget      *int64 `json:"goal_target,omitempty"` // The goal target amount in milliunits. Can only be changed if the category already has a goal (goal_type != null).
}

// UpdateCategoryRequest is the request body for updating a category.
type UpdateCategoryRequest struct {
	Category *SaveCategory `json:"category"`
}

// Payee represents a YNAB payee.
type Payee struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	TransferAccountID types.NullString `json:"transfer_account_id"` // If a transfer payee, the account_id to which this payee transfers to
	Deleted           bool             `json:"deleted"`
}

// PayeeResponse wraps a single payee response.
type PayeeResponse struct {
	Data struct {
		Payee *Payee `json:"payee"`
	} `json:"data"`
}

// PayeeListResponse wraps the payee list response.
type PayeeListResponse struct {
	Data struct {
		Payees          []*Payee `json:"payees"`
		ServerKnowledge int64    `json:"server_knowledge"`
	} `json:"data"`
}

// SavePayee represents the data for updating a payee.
type SavePayee struct {
	Name string `json:"name"` // Maximum 500 characters.
}

// UpdatePayeeRequest is the request body for updating a payee.
type UpdatePayeeRequest struct {
	Payee *SavePayee `json:"payee"`
}

// SavePayeeResponse wraps the response from updating a payee.
type SavePayeeResponse struct {
	Data struct {
		Payee           *Payee `json:"payee"`
		ServerKnowledge int64  `json:"server_knowledge"`
	} `json:"data"`
}

// PayeeLocation represents a payee's geographic location.
type PayeeLocation struct {
	ID        string `json:"id"`
	PayeeID   string `json:"payee_id"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
	Deleted   bool   `json:"deleted"`
}

// PayeeLocationResponse wraps a single payee location response.
type PayeeLocationResponse struct {
	Data struct {
		PayeeLocation *PayeeLocation `json:"payee_location"`
	} `json:"data"`
}

// PayeeLocationListResponse wraps the payee location list response.
type PayeeLocationListResponse struct {
	Data struct {
		PayeeLocations []*PayeeLocation `json:"payee_locations"`
	} `json:"data"`
}

// MonthSummary represents a budget month summary without category details.
type MonthSummary struct {
	Month        string `json:"month"`
	Note         string `json:"note"`
	Income       int64  `json:"income"`         // The total amount of transactions categorized to 'Inflow: Ready to Assign' in the month
	Budgeted     int64  `json:"budgeted"`       // The total amount budgeted in the month
	Activity     int64  `json:"activity"`       // The total amount of transactions in the month, excluding those categorized to 'Inflow: Ready to Assign'
	ToBeBudgeted int64  `json:"to_be_budgeted"` // The available amount for 'Ready to Assign'
	AgeOfMoney   *int   `json:"age_of_money"`   // The Age of Money as of the month
	Deleted      bool   `json:"deleted"`
}

// MonthSummaryListResponse wraps the month summary list response.
type MonthSummaryListResponse struct {
	Data struct {
		Months          []*MonthSummary `json:"months"`
		ServerKnowledge int64           `json:"server_knowledge"`
	} `json:"data"`
}

// MonthDetailResponse wraps a single month detail response.
type MonthDetailResponse struct {
	Data struct {
		Month *MonthDetail `json:"month"`
	} `json:"data"`
}

// HybridTransaction represents a transaction that may be either a regular
// transaction or a subtransaction, returned by payee/category/month transaction endpoints.
type HybridTransaction struct {
	ID                    string           `json:"id"`
	Date                  Date             `json:"date"`   // The transaction date in ISO format (e.g. 2016-12-01)
	Amount                int64            `json:"amount"` // The transaction amount in milliunits format
	Memo                  string           `json:"memo"`
	Cleared               ClearedStatus    `json:"cleared"`
	Approved              bool             `json:"approved"` // Whether or not the transaction is approved
	FlagColor             FlagColor        `json:"flag_color"`
	AccountID             string           `json:"account_id"`
	AccountName           string           `json:"account_name"`
	PayeeID               types.NullString `json:"payee_id"`
	PayeeName             string           `json:"payee_name"`
	CategoryID            types.NullString `json:"category_id"`
	CategoryName          types.NullString `json:"category_name"`           // If a split transaction, this will be 'Split'.
	TransferAccountID     types.NullString `json:"transfer_account_id"`     // If a transfer transaction, the account to which it transfers
	TransferTransactionID types.NullString `json:"transfer_transaction_id"` // If a transfer transaction, the id of transaction on the other side of the transfer
	MatchedTransactionID  types.NullString `json:"matched_transaction_id"`  // If transaction is matched, the id of the matched transaction
	Deleted               bool             `json:"deleted"`
	Type                  string           `json:"type"`                  // Whether the hybrid transaction represents a regular transaction or a subtransaction
	ParentTransactionID   types.NullString `json:"parent_transaction_id"` // For subtransaction types, this is the id of the parent transaction. For transaction types, this will be null.
	Subtransactions       []Transaction    `json:"subtransactions"`
}

// HybridTransactionListResponse wraps the hybrid transaction list response.
type HybridTransactionListResponse struct {
	Data struct {
		Transactions    []*HybridTransaction `json:"transactions"`
		ServerKnowledge int64                `json:"server_knowledge"`
	} `json:"data"`
}

// TransactionsImportResponse wraps the response from importing transactions.
type TransactionsImportResponse struct {
	Data struct {
		TransactionIDs []string `json:"transaction_ids"`
	} `json:"data"`
}

// UpdateTransactionsRequest is the request body for bulk-updating transactions.
type UpdateTransactionsRequest struct {
	Transactions []*UpdateTransaction `json:"transactions"`
}

// ScheduledTransactionResponse wraps a single scheduled transaction response.
type ScheduledTransactionResponse struct {
	Data struct {
		ScheduledTransaction *ScheduledTransaction `json:"scheduled_transaction"`
	} `json:"data"`
}

// SaveScheduledTransaction represents the data for creating or updating a scheduled transaction.
type SaveScheduledTransaction struct {
	AccountID  string           `json:"account_id"`
	Date       Date             `json:"date"`                  // The scheduled transaction date in ISO format (e.g. 2016-12-01). Must be a future date no more than 5 years out.
	Amount     *int64           `json:"amount,omitempty"`      // The scheduled transaction amount in milliunits format
	PayeeID    types.NullString `json:"payee_id,omitempty"`    // To create a transfer, use the account transfer payee pointing to the target account.
	PayeeName  types.NullString `json:"payee_name,omitempty"`  // If provided and payee_id is null, used to resolve the payee by same name or creation of a new payee.
	CategoryID types.NullString `json:"category_id,omitempty"` // Credit Card Payment categories are not permitted. Split scheduled transactions are not supported.
	Memo       types.NullString `json:"memo,omitempty"`
	FlagColor  FlagColor        `json:"flag_color,omitempty"`
	Frequency  string           `json:"frequency,omitempty"`
}

// CreateScheduledTransactionRequest is the request body for creating a scheduled transaction.
type CreateScheduledTransactionRequest struct {
	ScheduledTransaction *SaveScheduledTransaction `json:"scheduled_transaction"`
}

// UpdateScheduledTransactionRequest is the request body for updating a scheduled transaction.
type UpdateScheduledTransactionRequest struct {
	ScheduledTransaction *SaveScheduledTransaction `json:"scheduled_transaction"`
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

func (b *BudgetService) DeleteTransaction(ctx context.Context, transactionID string) (*TransactionResponse, error) {
	resp := new(TransactionResponse)
	err := b.client.MakeRequest(ctx, "DELETE", "/budgets/"+b.id+"/transactions/"+transactionID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetUser returns the authenticated user.
func (c *Client) GetUser(ctx context.Context) (*UserResponse, error) {
	resp := new(UserResponse)
	err := c.MakeRequest(ctx, "GET", "/user", nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSettings returns the settings for this budget.
func (b *BudgetService) GetSettings(ctx context.Context) (*BudgetSettingsResponse, error) {
	resp := new(BudgetSettingsResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/settings", nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateAccount creates a new account in this budget.
func (b *BudgetService) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*AccountResponse, error) {
	resp := new(AccountResponse)
	err := b.client.MakeRequest(ctx, "POST", "/budgets/"+b.id+"/accounts", nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetAccount returns a single account by ID.
func (b *BudgetService) GetAccount(ctx context.Context, accountID string) (*AccountResponse, error) {
	resp := new(AccountResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/accounts/"+accountID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetCategory returns a single category by ID.
func (b *BudgetService) GetCategory(ctx context.Context, categoryID string) (*CategoryResponse, error) {
	resp := new(CategoryResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/categories/"+categoryID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateCategory updates a category.
func (b *BudgetService) UpdateCategory(ctx context.Context, categoryID string, req *UpdateCategoryRequest) (*SaveCategoryResponse, error) {
	resp := new(SaveCategoryResponse)
	err := b.client.MakeRequest(ctx, "PATCH", "/budgets/"+b.id+"/categories/"+categoryID, nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Payees returns the list of payees for this budget.
func (b *BudgetService) Payees(ctx context.Context, data url.Values) (*PayeeListResponse, error) {
	resp := new(PayeeListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/payees", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPayee returns a single payee by ID.
func (b *BudgetService) GetPayee(ctx context.Context, payeeID string) (*PayeeResponse, error) {
	resp := new(PayeeResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/payees/"+payeeID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdatePayee updates a payee.
func (b *BudgetService) UpdatePayee(ctx context.Context, payeeID string, req *UpdatePayeeRequest) (*SavePayeeResponse, error) {
	resp := new(SavePayeeResponse)
	err := b.client.MakeRequest(ctx, "PATCH", "/budgets/"+b.id+"/payees/"+payeeID, nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// PayeeLocations returns all payee locations for this budget.
func (b *BudgetService) PayeeLocations(ctx context.Context) (*PayeeLocationListResponse, error) {
	resp := new(PayeeLocationListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/payee_locations", nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPayeeLocation returns a single payee location by ID.
func (b *BudgetService) GetPayeeLocation(ctx context.Context, locationID string) (*PayeeLocationResponse, error) {
	resp := new(PayeeLocationResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/payee_locations/"+locationID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// PayeeLocationsByPayee returns all payee locations for a specific payee.
func (b *BudgetService) PayeeLocationsByPayee(ctx context.Context, payeeID string) (*PayeeLocationListResponse, error) {
	resp := new(PayeeLocationListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/payees/"+payeeID+"/payee_locations", nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Months returns the list of budget months for this budget.
func (b *BudgetService) Months(ctx context.Context, data url.Values) (*MonthSummaryListResponse, error) {
	resp := new(MonthSummaryListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/months", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetMonth returns a single budget month.
// The month should be in ISO format (e.g., "2024-01-01") or "current".
func (b *BudgetService) GetMonth(ctx context.Context, month string) (*MonthDetailResponse, error) {
	resp := new(MonthDetailResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/months/"+month, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetTransaction returns a single transaction by ID.
func (b *BudgetService) GetTransaction(ctx context.Context, transactionID string) (*TransactionResponse, error) {
	resp := new(TransactionResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/transactions/"+transactionID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateTransactions bulk-updates multiple transactions.
func (b *BudgetService) UpdateTransactions(ctx context.Context, req *UpdateTransactionsRequest) (*CreateTransactionResponse, error) {
	resp := new(CreateTransactionResponse)
	err := b.client.MakeRequest(ctx, "PATCH", "/budgets/"+b.id+"/transactions", nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ImportTransactions imports transactions from linked accounts.
func (b *BudgetService) ImportTransactions(ctx context.Context) (*TransactionsImportResponse, error) {
	resp := new(TransactionsImportResponse)
	err := b.client.MakeRequest(ctx, "POST", "/budgets/"+b.id+"/transactions/import", nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// AccountTransactions returns the transactions for a specific account.
func (b *BudgetService) AccountTransactions(ctx context.Context, accountID string, data url.Values) (*TransactionListResponse, error) {
	resp := new(TransactionListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/accounts/"+accountID+"/transactions", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CategoryTransactions returns the transactions for a specific category.
func (b *BudgetService) CategoryTransactions(ctx context.Context, categoryID string, data url.Values) (*HybridTransactionListResponse, error) {
	resp := new(HybridTransactionListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/categories/"+categoryID+"/transactions", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// PayeeTransactions returns the transactions for a specific payee.
func (b *BudgetService) PayeeTransactions(ctx context.Context, payeeID string, data url.Values) (*HybridTransactionListResponse, error) {
	resp := new(HybridTransactionListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/payees/"+payeeID+"/transactions", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// MonthTransactions returns the transactions for a specific month.
func (b *BudgetService) MonthTransactions(ctx context.Context, month string, data url.Values) (*HybridTransactionListResponse, error) {
	resp := new(HybridTransactionListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/months/"+month+"/transactions", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateScheduledTransaction creates a new scheduled transaction.
func (b *BudgetService) CreateScheduledTransaction(ctx context.Context, req *CreateScheduledTransactionRequest) (*ScheduledTransactionResponse, error) {
	resp := new(ScheduledTransactionResponse)
	err := b.client.MakeRequest(ctx, "POST", "/budgets/"+b.id+"/scheduled_transactions", nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetScheduledTransaction returns a single scheduled transaction by ID.
func (b *BudgetService) GetScheduledTransaction(ctx context.Context, scheduledTransactionID string) (*ScheduledTransactionResponse, error) {
	resp := new(ScheduledTransactionResponse)
	err := b.client.MakeRequest(ctx, "GET", "/budgets/"+b.id+"/scheduled_transactions/"+scheduledTransactionID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateScheduledTransaction updates an existing scheduled transaction.
func (b *BudgetService) UpdateScheduledTransaction(ctx context.Context, scheduledTransactionID string, req *UpdateScheduledTransactionRequest) (*ScheduledTransactionResponse, error) {
	resp := new(ScheduledTransactionResponse)
	err := b.client.PutResource(ctx, "/budgets/"+b.id+"/scheduled_transactions", scheduledTransactionID, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// DeleteScheduledTransaction deletes a scheduled transaction.
func (b *BudgetService) DeleteScheduledTransaction(ctx context.Context, scheduledTransactionID string) (*ScheduledTransactionResponse, error) {
	resp := new(ScheduledTransactionResponse)
	err := b.client.MakeRequest(ctx, "DELETE", "/budgets/"+b.id+"/scheduled_transactions/"+scheduledTransactionID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// NewTransferTransaction creates a NewTransaction configured as a transfer between accounts.
// The sourceAccountID is where the transaction will appear (the "from" account).
// The targetAccount must have a valid TransferPayeeID (the "to" account).
// A positive amount transfers money into the source account; a negative amount transfers out.
func NewTransferTransaction(sourceAccountID string, targetAccount *Account, amount int64, date Date) (*NewTransaction, error) {
	if !targetAccount.TransferPayeeID.Valid {
		return nil, &Error{Message: "target account does not have a valid transfer_payee_id"}
	}
	return &NewTransaction{
		AccountID: sourceAccountID,
		Date:      date,
		Amount:    amount,
		PayeeID:   targetAccount.TransferPayeeID,
		Approved:  false,
	}, nil
}

// UpdateTransactionToTransfer creates an UpdateTransaction that converts an existing
// transaction into a transfer to the target account. The existing transaction's
// date, amount, memo, cleared status, and approval are preserved.
// The targetAccount must have a valid TransferPayeeID.
func UpdateTransactionToTransfer(existingTxn *Transaction, targetAccount *Account) (*UpdateTransaction, error) {
	if !targetAccount.TransferPayeeID.Valid {
		return nil, &Error{Message: "target account does not have a valid transfer_payee_id"}
	}
	return &UpdateTransaction{
		Date:     existingTxn.Date,
		Amount:   &existingTxn.Amount,
		PayeeID:  targetAccount.TransferPayeeID,
		Memo:     types.NullString{String: existingTxn.Memo, Valid: existingTxn.Memo != ""},
		Cleared:  types.NullString{String: string(existingTxn.Cleared), Valid: existingTxn.Cleared != ""},
		Approved: &existingTxn.Approved,
	}, nil
}

// Error represents an error from the YNAB API or this client library.
type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func (c *Client) MakeRequest(ctx context.Context, method string, pathPart string, data url.Values, reqBody interface{}, v interface{}) error {
	var rb io.Reader
	if reqBody != nil || (method == "POST" || method == "PUT" || method == "PATCH") {
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
	if reqBody != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	return c.Do(req, &v)
}

const Version = "1.4.0"

var defaultUserAgent = "ynab-go/" + Version

func (c *Client) NewRequestWithContext(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := c.Client.NewRequestWithContext(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	userAgent := c.userAgent
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	req.Header.Set("User-Agent", userAgent+" "+req.Header.Get("User-Agent"))
	return req, nil
}

// GetUserAgent returns the current User-Agent string that will be sent with requests
func (c *Client) GetUserAgent() string {
	if c.userAgent == "" {
		return defaultUserAgent
	}
	return c.userAgent
}

// SetUserAgent sets a custom User-Agent string for requests
func (c *Client) SetUserAgent(userAgent string) {
	c.userAgent = userAgent
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
