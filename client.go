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
	Plans     func(planID string) *PlanService
	// Budgets is deprecated. Use Plans.
	Budgets func(budgetID string) *PlanService
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
	PayeeID    types.NullString `json:"payee_id"`
	PayeeName  types.NullString `json:"payee_name"`  // If provided and payee_id is null, used to resolve the payee by matching rename rule, same name, or creation of a new payee.
	CategoryID types.NullString `json:"category_id"` // To configure a split, specify null and provide a subtransactions array. Credit Card Payment categories are not permitted.
	Memo       types.NullString `json:"memo"`
	Cleared    ClearedStatus    `json:"cleared,omitempty"`
	Approved   bool             `json:"approved"` // Whether or not the transaction is approved. If not supplied, transaction will be unapproved by default.
	FlagColor  FlagColor        `json:"flag_color,omitempty"`
	// An array of subtransactions to configure a transaction as a split. Updating subtransactions on an existing split transaction is not supported.
	Subtransactions   []*NewSubTransaction `json:"subtransactions,omitempty"`
	ImportID          types.NullString     `json:"import_id"`
	TransferAccountID types.NullString     `json:"transfer_account_id"`
}

type NewSubTransaction struct {
	Amount     int64            `json:"amount"` // The subtransaction amount in milliunits format
	PayeeID    types.NullString `json:"payee_id"`
	PayeeName  types.NullString `json:"payee_name"`  // If provided and payee_id is null, used to resolve the payee by matching rename rule, same name, or creation of a new payee.
	CategoryID types.NullString `json:"category_id"` // Credit Card Payment categories are not permitted.
	Memo       types.NullString `json:"memo"`
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
	Date       Date             `json:"date"`             // The transaction date in ISO format (e.g. 2016-12-01). Split transaction dates cannot be changed.
	Amount     *int64           `json:"amount,omitempty"` // The transaction amount in milliunits format. Split transaction amounts cannot be changed.
	PayeeID    types.NullString `json:"payee_id"`         // To create a transfer, use the account transfer payee pointing to the target account.
	PayeeName  types.NullString `json:"payee_name"`       // If provided and payee_id is null, used to resolve the payee by matching rename rule, same name, or creation of a new payee.
	CategoryID types.NullString `json:"category_id"`      // Credit Card Payment categories are not permitted.
	Memo       types.NullString `json:"memo"`
	Cleared    types.NullString `json:"cleared"`
	Approved   *bool            `json:"approved,omitempty"` // Whether or not the transaction is approved.
	FlagColor  types.NullString `json:"flag_color"`
	// An array of subtransactions to configure a transaction as a split. Updating subtransactions on an existing split transaction is not supported.
	Subtransactions []*SubTransaction `json:"subtransactions,omitempty"`
}

type SubTransaction struct {
	Amount     int64            `json:"amount"` // The subtransaction amount in milliunits format
	PayeeID    types.NullString `json:"payee_id"`
	PayeeName  types.NullString `json:"payee_name"`
	CategoryID types.NullString `json:"category_id"` // Credit Card Payment categories are not permitted.
	Memo       types.NullString `json:"memo"`
}

type CategoryListResponse struct {
	Data CategoryListWrapper `json:"data"`
}

type CategoryListWrapper struct {
	CategoryGroups  []*CategoryGroup `json:"category_groups"`
	ServerKnowledge int64            `json:"server_knowledge"`
}

type CategoryGroup struct {
	ID         string
	Name       string
	Hidden     bool
	Internal   bool
	Deleted    bool
	Categories []*Category
}

type Category struct {
	ID                      string
	Name                    string
	CategoryGroupID         string           `json:"category_group_id"`
	CategoryGroupName       string           `json:"category_group_name"` // The name of the category group
	OriginalCategoryGroupID types.NullString `json:"original_category_group_id"`
	Note                    string           `json:"note"`
	Hidden                  bool
	Internal                bool
	Deleted                 bool
	Budgeted                int64 // Assigned (budgeted) amount in milliunits format
	Activity                int64 // Activity amount in milliunits format
	Balance                 int64 // Available balance in milliunits format

	BudgetedFormatted string  `json:"budgeted_formatted"`
	BudgetedCurrency  float64 `json:"budgeted_currency"`
	ActivityFormatted string  `json:"activity_formatted"`
	ActivityCurrency  float64 `json:"activity_currency"`
	BalanceFormatted  string  `json:"balance_formatted"`
	BalanceCurrency   float64 `json:"balance_currency"`

	GoalType                   types.NullString `json:"goal_type"`                     // The type of goal, or null. TB=Target Category Balance, TBD=Target Category Balance by Date, MF=Monthly Funding, NEED=Plan Your Spending, DEBT=Debt Payoff
	GoalTarget                 *int64           `json:"goal_target"`                   // The goal target amount in milliunits
	GoalTargetFormatted        types.NullString `json:"goal_target_formatted"`         // The goal target amount formatted in the plan's currency format
	GoalTargetCurrency         *float64         `json:"goal_target_currency"`          // The goal target amount as a decimal currency amount
	GoalPercentageComplete     *int32           `json:"goal_percentage_complete"`      // The percentage completed of the goal
	GoalMonthsToBudget         *int32           `json:"goal_months_to_budget"`         // The number of months remaining until the goal is completed
	GoalUnderFunded            *int64           `json:"goal_under_funded"`             // The amount of funding still needed in milliunits
	GoalUnderFundedFormatted   types.NullString `json:"goal_under_funded_formatted"`   // The goal underfunded amount formatted in the plan's currency format
	GoalUnderFundedCurrency    *float64         `json:"goal_under_funded_currency"`    // The goal underfunded amount as a decimal currency amount
	GoalOverallFunded          *int64           `json:"goal_overall_funded"`           // The total amount funded towards the goal in milliunits
	GoalOverallFundedFormatted types.NullString `json:"goal_overall_funded_formatted"` // The total amount funded towards the goal formatted in the plan's currency format
	GoalOverallFundedCurrency  *float64         `json:"goal_overall_funded_currency"`  // The total amount funded towards the goal as a decimal currency amount
	GoalOverallLeft            *int64           `json:"goal_overall_left"`             // The amount still left to fund the goal in milliunits
	GoalOverallLeftFormatted   types.NullString `json:"goal_overall_left_formatted"`   // The amount still left to fund the goal formatted in the plan's currency format
	GoalOverallLeftCurrency    *float64         `json:"goal_overall_left_currency"`    // The amount still left to fund the goal as a decimal currency amount
	GoalNeedsWholeAmount       *bool            `json:"goal_needs_whole_amount"`       // For NEED goals: true=Set Aside, false=Refill. Null for other goal types
	GoalDay                    *int32           `json:"goal_day"`                      // Day offset for the goal's due date
	GoalCadence                *int32           `json:"goal_cadence"`                  // The goal cadence (0-14)
	GoalCadenceFrequency       *int32           `json:"goal_cadence_frequency"`        // The goal cadence frequency
	GoalCreationMonth          types.NullString `json:"goal_creation_month"`           // The month a goal was created
	GoalTargetMonth            types.NullString `json:"goal_target_month"`             // The original target month for the goal to be completed
	GoalTargetDate             types.NullString `json:"goal_target_date"`              // The target date for the goal to be completed
	GoalSnoozedAt              types.NullString `json:"goal_snoozed_at"`               // The date/time the goal was snoozed
}

// UpdateMonthCategoryRequest is the request body for updating a category's assigned amount for a month.
type UpdateMonthCategoryRequest struct {
	Category SaveMonthCategory `json:"category"`
}

// SaveMonthCategory contains the assigned amount to set for a category in a month.
type SaveMonthCategory struct {
	Budgeted int64 `json:"budgeted"` // Budgeted amount in milliunits format
}

// SaveCategoryResponse is the response from creating or updating a category.
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
	Transactions    []*Transaction `json:"transactions"`
	ServerKnowledge int64          `json:"server_knowledge"`
}

type ScheduledTransactionListResponse struct {
	Data ScheduledTransactionListWrapper `json:"data"`
}

type ScheduledTransactionListWrapper struct {
	ScheduledTransactions []*ScheduledTransaction `json:"scheduled_transactions"`
	ServerKnowledge       int64                   `json:"server_knowledge"`
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
	AccountID         string           `json:"account_id"`
	AccountName       string           `json:"account_name"`
	Amount            int64            // The scheduled transaction amount in milliunits format
	AmountFormatted   string           `json:"amount_formatted"`
	AmountCurrency    float64          `json:"amount_currency"`
	CategoryID        types.NullString `json:"category_id"`
	CategoryName      types.NullString `json:"category_name"`
	DateFirst         Date             `json:"date_first"` // The first date for which the Scheduled Transaction was scheduled.
	DateNext          Date             `json:"date_next"`  // The next date for which the Scheduled Transaction is scheduled.
	Deleted           bool
	FlagColor         FlagColor        `json:"flag_color"`
	FlagName          types.NullString `json:"flag_name"` // The customized name of a transaction flag
	Frequency         string
	ID                string `json:"id"`
	Memo              string
	PayeeID           types.NullString `json:"payee_id"`
	PayeeName         string           `json:"payee_name"`
	TransferAccountID types.NullString `json:"transfer_account_id"` // If a transfer, the account_id which the scheduled transaction transfers to
	Subtransactions   []Transaction    `json:"subtransactions"`
}

type Transaction struct {
	AccountID               string           `json:"account_id"`
	AccountName             string           `json:"account_name"`
	Amount                  int64            // The transaction amount in milliunits format
	AmountFormatted         string           `json:"amount_formatted"`
	AmountCurrency          float64          `json:"amount_currency"`
	Approved                bool             // Whether or not the transaction is approved
	CategoryID              types.NullString `json:"category_id"`
	CategoryName            types.NullString `json:"category_name"` // The name of the category. If a split transaction, this will be 'Split'.
	Cleared                 ClearedStatus    // The cleared status of the transaction
	Date                    Date             // The transaction date in ISO format (e.g. 2016-12-01)
	DebtTransactionType     types.NullString `json:"debt_transaction_type"` // If a debt/loan account transaction, the type of transaction
	Deleted                 bool             // Whether or not the transaction has been deleted. Deleted transactions will only be included in delta requests.
	FlagColor               FlagColor        `json:"flag_color"` // The transaction flag
	FlagName                types.NullString `json:"flag_name"`  // The customized name of a transaction flag
	ID                      string           `json:"id"`
	ImportID                types.NullString `json:"import_id"`                  // If the transaction was imported, a unique (by account) import identifier
	ImportPayeeName         types.NullString `json:"import_payee_name"`          // If the transaction was imported, the payee name that was used when importing and before applying any payee rename rules
	ImportPayeeNameOriginal types.NullString `json:"import_payee_name_original"` // If the transaction was imported, the original payee name as it appeared on the statement
	Memo                    string
	PayeeID                 types.NullString `json:"payee_id"`
	PayeeName               string           `json:"payee_name"`
	TransferAccountID       types.NullString `json:"transfer_account_id"`     // If a transfer transaction, the account to which it transfers
	TransferTransactionID   types.NullString `json:"transfer_transaction_id"` // If a transfer transaction, the id of transaction on the other side of the transfer
	MatchedTransactionID    types.NullString `json:"matched_transaction_id"`  // If transaction is matched, the id of the matched transaction
	Subtransactions         []Transaction    `json:"subtransactions"`         // If a split transaction, the subtransactions.
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
	ID                        string
	Name                      string
	Type                      string
	OnBudget                  bool `json:"on_budget"` // Whether this account is on budget or not
	Closed                    bool // Whether this account is closed or not
	Note                      string
	Balance                   int64   // The current balance of the account in milliunits format
	ClearedBalance            int64   `json:"cleared_balance"`   // The current cleared balance of the account in milliunits format
	UnclearedBalance          int64   `json:"uncleared_balance"` // The current uncleared balance of the account in milliunits format
	BalanceFormatted          string  `json:"balance_formatted"`
	BalanceCurrency           float64 `json:"balance_currency"`
	ClearedBalanceFormatted   string  `json:"cleared_balance_formatted"`
	ClearedBalanceCurrency    float64 `json:"cleared_balance_currency"`
	UnclearedBalanceFormatted string  `json:"uncleared_balance_formatted"`
	UnclearedBalanceCurrency  float64 `json:"uncleared_balance_currency"`
	StartingBalance           int64   `json:"starting_balance"`
	Deleted                   bool
	TransferPayeeID           types.NullString `json:"transfer_payee_id"`      // The payee id which should be used when transferring to this account
	DirectImportLinked        bool             `json:"direct_import_linked"`   // Whether or not the account is linked to a financial institution for automatic transaction import
	DirectImportInError       bool             `json:"direct_import_in_error"` // If an account linked to a financial institution and the linked connection is not in a healthy state, this will be true
	LastReconciledAt          types.NullString `json:"last_reconciled_at"`     // A date/time specifying when the account was last reconciled
	DebtInterestRates         map[string]int64 `json:"debt_interest_rates"`    // Loan account periodic interest rate values
	DebtMinimumPayments       map[string]int64 `json:"debt_minimum_payments"`  // Loan account periodic minimum payment values
	DebtEscrowAmounts         map[string]int64 `json:"debt_escrow_amounts"`    // Loan account periodic escrow amount values
}

func (a Account) CashBacked() bool {
	return a.Type == "cash" || a.Type == "savings" || a.Type == "checking"
}

type AccountListResponse struct {
	Data AccountListWrapper `json:"data"`
}

type AccountListWrapper struct {
	Accounts        []*Account `json:"accounts"`
	ServerKnowledge int64      `json:"server_knowledge"`
}

type Plan struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	LastModifiedOn string     `json:"last_modified_on"` // The last time any changes were made to the plan from either a web or mobile client
	FirstMonth     string     `json:"first_month"`      // The earliest plan month
	LastMonth      string     `json:"last_month"`       // The latest plan month
	DateFormat     any        `json:"date_format"`
	CurrencyFormat any        `json:"currency_format"`
	Accounts       []*Account `json:"accounts,omitempty"`
}

// Budget is deprecated. Use Plan.
type Budget = Plan

type PlanListResponse struct {
	Data PlanListWrapper `json:"data"`
}

type PlanListWrapper struct {
	Plans       []*Plan `json:"plans"`
	DefaultPlan *Plan   `json:"default_plan"`
}

type BudgetListResponse struct {
	Data BudgetListWrapper `json:"data"`
}

type BudgetListWrapper struct {
	Budgets       []*Budget `json:"plans"`
	DefaultBudget *Budget   `json:"default_plan"`
}

// MonthDetail represents a plan month with all its categories.
type MonthDetail struct {
	Month                 string      `json:"month"`
	Note                  string      `json:"note"`
	Income                int64       `json:"income"` // The total amount of transactions categorized to 'Inflow: Ready to Assign' in the month
	IncomeFormatted       string      `json:"income_formatted"`
	IncomeCurrency        float64     `json:"income_currency"`
	Budgeted              int64       `json:"budgeted"` // The total amount assigned in the month
	BudgetedFormatted     string      `json:"budgeted_formatted"`
	BudgetedCurrency      float64     `json:"budgeted_currency"`
	Activity              int64       `json:"activity"` // The total amount of transactions in the month, excluding those categorized to 'Inflow: Ready to Assign'
	ActivityFormatted     string      `json:"activity_formatted"`
	ActivityCurrency      float64     `json:"activity_currency"`
	ToBeBudget            int64       `json:"to_be_budgeted"` // The available amount for 'Ready to Assign'
	ToBeBudgetedFormatted string      `json:"to_be_budgeted_formatted"`
	ToBeBudgetedCurrency  float64     `json:"to_be_budgeted_currency"`
	AgeOfMoney            int         `json:"age_of_money"` // The Age of Money as of the month
	Deleted               bool        `json:"deleted"`
	Categories            []*Category `json:"categories"` // Amounts (budgeted, activity, balance, etc.) are specific to this month.
}

// PlanDetail represents a full plan export with all related entities.
type PlanDetail struct {
	ID                       string                  `json:"id"`
	Name                     string                  `json:"name"`
	LastModifiedOn           string                  `json:"last_modified_on"` // The last time any changes were made to the plan from either a web or mobile client
	FirstMonth               string                  `json:"first_month"`      // The earliest plan month
	LastMonth                string                  `json:"last_month"`       // The latest plan month
	DateFormat               any                     `json:"date_format"`
	CurrencyFormat           any                     `json:"currency_format"`
	Accounts                 []*Account              `json:"accounts"`
	Payees                   []*Payee                `json:"payees"`
	PayeeLocations           []*PayeeLocation        `json:"payee_locations"`
	CategoryGroups           []*CategoryGroup        `json:"category_groups"`
	Categories               []*Category             `json:"categories"`
	Months                   []*MonthDetail          `json:"months"`
	Transactions             []*Transaction          `json:"transactions"`
	Subtransactions          []*Transaction          `json:"subtransactions"`
	ScheduledTransactions    []*ScheduledTransaction `json:"scheduled_transactions"`
	ScheduledSubtransactions []*ScheduledTransaction `json:"scheduled_subtransactions"`
}

// BudgetDetail is deprecated. Use PlanDetail.
type BudgetDetail = PlanDetail

// PlanDetailResponse wraps the plan detail response.
type PlanDetailResponse struct {
	Data struct {
		Plan            *PlanDetail `json:"plan"`
		ServerKnowledge int64       `json:"server_knowledge"`
	} `json:"data"`
}

// BudgetDetailResponse wraps the plan detail response with the old field name.
type BudgetDetailResponse struct {
	Data struct {
		Budget          *BudgetDetail `json:"plan"`
		ServerKnowledge int64         `json:"server_knowledge"`
	} `json:"data"`
}

// GetPlan returns a full plan export with all months and categories.
// This is more efficient than making per-month API calls.
func (b *PlanService) GetPlan(ctx context.Context) (*PlanDetailResponse, error) {
	req, err := b.client.NewRequestWithContext(ctx, "GET", "/plans/"+b.id, nil)
	if err != nil {
		return nil, err
	}
	planResp := new(PlanDetailResponse)
	if err := b.client.Do(req, planResp); err != nil {
		return nil, err
	}
	return planResp, nil
}

// GetBudget is deprecated. Use GetPlan.
func (b *PlanService) GetBudget(ctx context.Context) (*BudgetDetailResponse, error) {
	req, err := b.client.NewRequestWithContext(ctx, "GET", "/plans/"+b.id, nil)
	if err != nil {
		return nil, err
	}
	budgetResp := new(BudgetDetailResponse)
	if err := b.client.Do(req, budgetResp); err != nil {
		return nil, err
	}
	return budgetResp, nil
}

func (c *Client) GetPlans(ctx context.Context, data url.Values) (*PlanListResponse, error) {
	req, err := c.NewRequestWithContext(ctx, "GET", "/plans?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	planResp := new(PlanListResponse)
	if err := c.Do(req, planResp); err != nil {
		return nil, err
	}
	return planResp, nil
}

// GetBudgets is deprecated. Use GetPlans.
func (c *Client) GetBudgets(ctx context.Context, data url.Values) (*BudgetListResponse, error) {
	req, err := c.NewRequestWithContext(ctx, "GET", "/plans?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	budgetResp := new(BudgetListResponse)
	if err := c.Do(req, budgetResp); err != nil {
		return nil, err
	}
	return budgetResp, nil
}

func (b *PlanService) Accounts(ctx context.Context, data url.Values) (*AccountListResponse, error) {
	req, err := b.client.NewRequestWithContext(ctx, "GET", "/plans/"+b.id+"/accounts?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	accountResp := new(AccountListResponse)
	if err := b.client.Do(req, accountResp); err != nil {
		return nil, err
	}
	return accountResp, nil
}

func (b *PlanService) Transactions(ctx context.Context, data url.Values) (*TransactionListResponse, error) {
	req, err := b.client.NewRequestWithContext(ctx, "GET", "/plans/"+b.id+"/transactions?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	transactionResp := new(TransactionListResponse)
	if err := b.client.Do(req, transactionResp); err != nil {
		return nil, err
	}
	return transactionResp, nil
}

func (b *PlanService) ScheduledTransactions(ctx context.Context, data url.Values) (*ScheduledTransactionListResponse, error) {
	req, err := b.client.NewRequestWithContext(ctx, "GET", "/plans/"+b.id+"/scheduled_transactions?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	transactionResp := new(ScheduledTransactionListResponse)
	if err := b.client.Do(req, transactionResp); err != nil {
		return nil, err
	}
	return transactionResp, nil
}

func (b *PlanService) Categories(ctx context.Context, data url.Values) (*CategoryListResponse, error) {
	req, err := b.client.NewRequestWithContext(ctx, "GET", "/plans/"+b.id+"/categories?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}
	categoryResp := new(CategoryListResponse)
	if err := b.client.Do(req, categoryResp); err != nil {
		return nil, err
	}
	return categoryResp, nil
}

// GetMonthCategory retrieves a category for a specific plan month.
// The month should be in ISO format (e.g., "2024-01-01") or "current" for the current month.
// Amounts (budgeted, activity, balance) are specific to the requested month.
func (b *PlanService) GetMonthCategory(ctx context.Context, month string, categoryID string) (*CategoryResponse, error) {
	resp := new(CategoryResponse)
	path := "/plans/" + b.id + "/months/" + month + "/categories/" + categoryID
	err := b.client.MakeRequest(ctx, "GET", path, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateMonthCategory updates the budgeted amount for a category in a specific month.
// The month should be in ISO format (e.g., "2024-01-01") or "current" for the current month.
// The budgeted amount is in milliunits (e.g., $50.00 = 50000).
func (b *PlanService) UpdateMonthCategory(ctx context.Context, month string, categoryID string, budgeted int64) (*SaveCategoryResponse, error) {
	req := &UpdateMonthCategoryRequest{
		Category: SaveMonthCategory{Budgeted: budgeted},
	}
	resp := new(SaveCategoryResponse)
	path := "/plans/" + b.id + "/months/" + month + "/categories/" + categoryID
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

// DateFormat represents the date format setting for a plan.
type DateFormat struct {
	Format string `json:"format"`
}

// CurrencyFormat represents the currency format setting for a plan.
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

// PlanSettings represents the settings for a plan.
type PlanSettings struct {
	DateFormat     DateFormat     `json:"date_format"`
	CurrencyFormat CurrencyFormat `json:"currency_format"`
}

// BudgetSettings is deprecated. Use PlanSettings.
type BudgetSettings = PlanSettings

// PlanSettingsResponse wraps the plan settings response.
type PlanSettingsResponse struct {
	Data struct {
		Settings PlanSettings `json:"settings"`
	} `json:"data"`
}

// BudgetSettingsResponse is deprecated. Use PlanSettingsResponse.
type BudgetSettingsResponse = PlanSettingsResponse

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
	Name                 string `json:"name,omitempty"`
	Note                 string `json:"note,omitempty"`
	CategoryGroupID      string `json:"category_group_id,omitempty"`
	GoalTarget           *int64 `json:"goal_target,omitempty"` // The goal target amount in milliunits.
	GoalTargetDate       string `json:"goal_target_date,omitempty"`
	GoalNeedsWholeAmount *bool  `json:"goal_needs_whole_amount,omitempty"`
}

// CreateCategoryRequest is the request body for creating a category.
type CreateCategoryRequest struct {
	Category *SaveCategory `json:"category"`
}

// UpdateCategoryRequest is the request body for updating a category.
type UpdateCategoryRequest struct {
	Category *SaveCategory `json:"category"`
}

// SaveCategoryGroup represents the data for creating or updating a category group.
type SaveCategoryGroup struct {
	Name string `json:"name"` // Maximum 50 characters.
}

// CreateCategoryGroupRequest is the request body for creating a category group.
type CreateCategoryGroupRequest struct {
	CategoryGroup *SaveCategoryGroup `json:"category_group"`
}

// UpdateCategoryGroupRequest is the request body for updating a category group.
type UpdateCategoryGroupRequest struct {
	CategoryGroup *SaveCategoryGroup `json:"category_group"`
}

// SaveCategoryGroupResponse wraps the response from creating or updating a category group.
type SaveCategoryGroupResponse struct {
	Data struct {
		CategoryGroup   *CategoryGroup `json:"category_group"`
		ServerKnowledge int64          `json:"server_knowledge"`
	} `json:"data"`
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

// MonthSummary represents a plan month summary without category details.
type MonthSummary struct {
	Month                 string  `json:"month"`
	Note                  string  `json:"note"`
	Income                int64   `json:"income"` // The total amount of transactions categorized to 'Inflow: Ready to Assign' in the month
	IncomeFormatted       string  `json:"income_formatted"`
	IncomeCurrency        float64 `json:"income_currency"`
	Budgeted              int64   `json:"budgeted"` // The total amount assigned in the month
	BudgetedFormatted     string  `json:"budgeted_formatted"`
	BudgetedCurrency      float64 `json:"budgeted_currency"`
	Activity              int64   `json:"activity"` // The total amount of transactions in the month, excluding those categorized to 'Inflow: Ready to Assign'
	ActivityFormatted     string  `json:"activity_formatted"`
	ActivityCurrency      float64 `json:"activity_currency"`
	ToBeBudgeted          int64   `json:"to_be_budgeted"` // The available amount for 'Ready to Assign'
	ToBeBudgetedFormatted string  `json:"to_be_budgeted_formatted"`
	ToBeBudgetedCurrency  float64 `json:"to_be_budgeted_currency"`
	AgeOfMoney            *int    `json:"age_of_money"` // The Age of Money as of the month
	Deleted               bool    `json:"deleted"`
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

// MoneyMovement represents a category money movement in a plan.
type MoneyMovement struct {
	ID                   string           `json:"id"`
	Month                NullDate         `json:"month"`
	MovedAt              types.NullString `json:"moved_at"`
	Note                 types.NullString `json:"note"`
	MoneyMovementGroupID types.NullString `json:"money_movement_group_id"`
	PerformedByUserID    types.NullString `json:"performed_by_user_id"`
	FromCategoryID       types.NullString `json:"from_category_id"`
	ToCategoryID         types.NullString `json:"to_category_id"`
	Amount               int64            `json:"amount"`
	AmountFormatted      string           `json:"amount_formatted"`
	AmountCurrency       float64          `json:"amount_currency"`
}

// MoneyMovementsResponse wraps the money movements list response.
type MoneyMovementsResponse struct {
	Data struct {
		MoneyMovements  []*MoneyMovement `json:"money_movements"`
		ServerKnowledge int64            `json:"server_knowledge"`
	} `json:"data"`
}

// MoneyMovementGroup represents a group of money movements in a plan.
type MoneyMovementGroup struct {
	ID                string           `json:"id"`
	GroupCreatedAt    string           `json:"group_created_at"`
	Month             Date             `json:"month"`
	Note              types.NullString `json:"note"`
	PerformedByUserID types.NullString `json:"performed_by_user_id"`
}

// MoneyMovementGroupsResponse wraps the money movement groups list response.
type MoneyMovementGroupsResponse struct {
	Data struct {
		MoneyMovementGroups []*MoneyMovementGroup `json:"money_movement_groups"`
		ServerKnowledge     int64                 `json:"server_knowledge"`
	} `json:"data"`
}

// HybridTransaction represents a transaction that may be either a regular
// transaction or a subtransaction, returned by payee/category/month transaction endpoints.
type HybridTransaction struct {
	ID                      string           `json:"id"`
	Date                    Date             `json:"date"`   // The transaction date in ISO format (e.g. 2016-12-01)
	Amount                  int64            `json:"amount"` // The transaction amount in milliunits format
	AmountFormatted         string           `json:"amount_formatted"`
	AmountCurrency          float64          `json:"amount_currency"`
	Memo                    string           `json:"memo"`
	Cleared                 ClearedStatus    `json:"cleared"`
	Approved                bool             `json:"approved"` // Whether or not the transaction is approved
	FlagColor               FlagColor        `json:"flag_color"`
	FlagName                types.NullString `json:"flag_name"` // The customized name of a transaction flag
	AccountID               string           `json:"account_id"`
	AccountName             string           `json:"account_name"`
	PayeeID                 types.NullString `json:"payee_id"`
	PayeeName               string           `json:"payee_name"`
	CategoryID              types.NullString `json:"category_id"`
	CategoryName            types.NullString `json:"category_name"`              // If a split transaction, this will be 'Split'.
	TransferAccountID       types.NullString `json:"transfer_account_id"`        // If a transfer transaction, the account to which it transfers
	TransferTransactionID   types.NullString `json:"transfer_transaction_id"`    // If a transfer transaction, the id of transaction on the other side of the transfer
	MatchedTransactionID    types.NullString `json:"matched_transaction_id"`     // If transaction is matched, the id of the matched transaction
	ImportID                types.NullString `json:"import_id"`                  // If the transaction was imported, a unique (by account) import identifier
	ImportPayeeName         types.NullString `json:"import_payee_name"`          // If the transaction was imported, the payee name that was used when importing and before applying any payee rename rules
	ImportPayeeNameOriginal types.NullString `json:"import_payee_name_original"` // If the transaction was imported, the original payee name as it appeared on the statement
	DebtTransactionType     types.NullString `json:"debt_transaction_type"`      // If a debt/loan account transaction, the type of transaction
	Deleted                 bool             `json:"deleted"`
	Type                    string           `json:"type"`                  // Whether the hybrid transaction represents a regular transaction or a subtransaction
	ParentTransactionID     types.NullString `json:"parent_transaction_id"` // For subtransaction types, this is the id of the parent transaction. For transaction types, this will be null.
	Subtransactions         []Transaction    `json:"subtransactions"`
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
	Date       Date             `json:"date"`             // The scheduled transaction date in ISO format (e.g. 2016-12-01). Must be a future date no more than 5 years out.
	Amount     *int64           `json:"amount,omitempty"` // The scheduled transaction amount in milliunits format
	PayeeID    types.NullString `json:"payee_id"`         // To create a transfer, use the account transfer payee pointing to the target account.
	PayeeName  types.NullString `json:"payee_name"`       // If provided and payee_id is null, used to resolve the payee by same name or creation of a new payee.
	CategoryID types.NullString `json:"category_id"`      // Credit Card Payment categories are not permitted. Split scheduled transactions are not supported.
	Memo       types.NullString `json:"memo"`
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

type PlanService struct {
	client *Client
	// the plan ID
	id string
}

// BudgetService is deprecated. Use PlanService.
type BudgetService = PlanService

func (c *Client) PutResource(ctx context.Context, pathPart string, sid string, req any, resp any) error {
	sidPart := strings.Join([]string{pathPart, sid}, "/")
	return c.MakeRequest(ctx, "PUT", sidPart, nil, req, resp)
}

func (b *PlanService) CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*CreateTransactionResponse, error) {
	resp := new(CreateTransactionResponse)
	err := b.client.MakeRequest(ctx, "POST", "/plans/"+b.id+"/transactions", nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *PlanService) UpdateTransaction(ctx context.Context, transactionID string, req *UpdateTransactionRequest) (*TransactionResponse, error) {
	resp := new(TransactionResponse)
	err := b.client.PutResource(ctx, "/plans/"+b.id+"/transactions", transactionID, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *PlanService) DeleteTransaction(ctx context.Context, transactionID string) (*TransactionResponse, error) {
	resp := new(TransactionResponse)
	err := b.client.MakeRequest(ctx, "DELETE", "/plans/"+b.id+"/transactions/"+transactionID, nil, nil, resp)
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

// GetSettings returns the settings for this plan.
func (b *PlanService) GetSettings(ctx context.Context) (*BudgetSettingsResponse, error) {
	resp := new(BudgetSettingsResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/settings", nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateAccount creates a new account in this plan.
func (b *PlanService) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*AccountResponse, error) {
	resp := new(AccountResponse)
	err := b.client.MakeRequest(ctx, "POST", "/plans/"+b.id+"/accounts", nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetAccount returns a single account by ID.
func (b *PlanService) GetAccount(ctx context.Context, accountID string) (*AccountResponse, error) {
	resp := new(AccountResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/accounts/"+accountID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetCategory returns a single category by ID.
func (b *PlanService) GetCategory(ctx context.Context, categoryID string) (*CategoryResponse, error) {
	resp := new(CategoryResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/categories/"+categoryID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateCategory creates a category in this plan.
func (b *PlanService) CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*SaveCategoryResponse, error) {
	resp := new(SaveCategoryResponse)
	err := b.client.MakeRequest(ctx, "POST", "/plans/"+b.id+"/categories", nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateCategory updates a category.
func (b *PlanService) UpdateCategory(ctx context.Context, categoryID string, req *UpdateCategoryRequest) (*SaveCategoryResponse, error) {
	resp := new(SaveCategoryResponse)
	err := b.client.MakeRequest(ctx, "PATCH", "/plans/"+b.id+"/categories/"+categoryID, nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateCategoryGroup creates a category group in this plan.
func (b *PlanService) CreateCategoryGroup(ctx context.Context, req *CreateCategoryGroupRequest) (*SaveCategoryGroupResponse, error) {
	resp := new(SaveCategoryGroupResponse)
	err := b.client.MakeRequest(ctx, "POST", "/plans/"+b.id+"/category_groups", nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateCategoryGroup updates a category group.
func (b *PlanService) UpdateCategoryGroup(ctx context.Context, categoryGroupID string, req *UpdateCategoryGroupRequest) (*SaveCategoryGroupResponse, error) {
	resp := new(SaveCategoryGroupResponse)
	err := b.client.MakeRequest(ctx, "PATCH", "/plans/"+b.id+"/category_groups/"+categoryGroupID, nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Payees returns the list of payees for this plan.
func (b *PlanService) Payees(ctx context.Context, data url.Values) (*PayeeListResponse, error) {
	resp := new(PayeeListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/payees", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPayee returns a single payee by ID.
func (b *PlanService) GetPayee(ctx context.Context, payeeID string) (*PayeeResponse, error) {
	resp := new(PayeeResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/payees/"+payeeID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdatePayee updates a payee.
func (b *PlanService) UpdatePayee(ctx context.Context, payeeID string, req *UpdatePayeeRequest) (*SavePayeeResponse, error) {
	resp := new(SavePayeeResponse)
	err := b.client.MakeRequest(ctx, "PATCH", "/plans/"+b.id+"/payees/"+payeeID, nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// PayeeLocations returns all payee locations for this plan.
func (b *PlanService) PayeeLocations(ctx context.Context) (*PayeeLocationListResponse, error) {
	resp := new(PayeeLocationListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/payee_locations", nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPayeeLocation returns a single payee location by ID.
func (b *PlanService) GetPayeeLocation(ctx context.Context, locationID string) (*PayeeLocationResponse, error) {
	resp := new(PayeeLocationResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/payee_locations/"+locationID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// PayeeLocationsByPayee returns all payee locations for a specific payee.
func (b *PlanService) PayeeLocationsByPayee(ctx context.Context, payeeID string) (*PayeeLocationListResponse, error) {
	resp := new(PayeeLocationListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/payees/"+payeeID+"/payee_locations", nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Months returns the list of plan months for this plan.
func (b *PlanService) Months(ctx context.Context, data url.Values) (*MonthSummaryListResponse, error) {
	resp := new(MonthSummaryListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/months", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetMonth returns a single plan month.
// The month should be in ISO format (e.g., "2024-01-01") or "current".
func (b *PlanService) GetMonth(ctx context.Context, month string) (*MonthDetailResponse, error) {
	resp := new(MonthDetailResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/months/"+month, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// MoneyMovements returns all money movements for this plan.
func (b *PlanService) MoneyMovements(ctx context.Context, data url.Values) (*MoneyMovementsResponse, error) {
	resp := new(MoneyMovementsResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/money_movements", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// MonthMoneyMovements returns money movements for a specific month.
func (b *PlanService) MonthMoneyMovements(ctx context.Context, month string, data url.Values) (*MoneyMovementsResponse, error) {
	resp := new(MoneyMovementsResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/months/"+month+"/money_movements", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// MoneyMovementGroups returns all money movement groups for this plan.
func (b *PlanService) MoneyMovementGroups(ctx context.Context, data url.Values) (*MoneyMovementGroupsResponse, error) {
	resp := new(MoneyMovementGroupsResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/money_movement_groups", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// MonthMoneyMovementGroups returns money movement groups for a specific month.
func (b *PlanService) MonthMoneyMovementGroups(ctx context.Context, month string, data url.Values) (*MoneyMovementGroupsResponse, error) {
	resp := new(MoneyMovementGroupsResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/months/"+month+"/money_movement_groups", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetTransaction returns a single transaction by ID.
func (b *PlanService) GetTransaction(ctx context.Context, transactionID string) (*TransactionResponse, error) {
	resp := new(TransactionResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/transactions/"+transactionID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateTransactions bulk-updates multiple transactions.
func (b *PlanService) UpdateTransactions(ctx context.Context, req *UpdateTransactionsRequest) (*CreateTransactionResponse, error) {
	resp := new(CreateTransactionResponse)
	err := b.client.MakeRequest(ctx, "PATCH", "/plans/"+b.id+"/transactions", nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ImportTransactions imports transactions from linked accounts.
func (b *PlanService) ImportTransactions(ctx context.Context) (*TransactionsImportResponse, error) {
	resp := new(TransactionsImportResponse)
	err := b.client.MakeRequest(ctx, "POST", "/plans/"+b.id+"/transactions/import", nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// AccountTransactions returns the transactions for a specific account.
func (b *PlanService) AccountTransactions(ctx context.Context, accountID string, data url.Values) (*TransactionListResponse, error) {
	resp := new(TransactionListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/accounts/"+accountID+"/transactions", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CategoryTransactions returns the transactions for a specific category.
func (b *PlanService) CategoryTransactions(ctx context.Context, categoryID string, data url.Values) (*HybridTransactionListResponse, error) {
	resp := new(HybridTransactionListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/categories/"+categoryID+"/transactions", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// PayeeTransactions returns the transactions for a specific payee.
func (b *PlanService) PayeeTransactions(ctx context.Context, payeeID string, data url.Values) (*HybridTransactionListResponse, error) {
	resp := new(HybridTransactionListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/payees/"+payeeID+"/transactions", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// MonthTransactions returns the transactions for a specific month.
func (b *PlanService) MonthTransactions(ctx context.Context, month string, data url.Values) (*HybridTransactionListResponse, error) {
	resp := new(HybridTransactionListResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/months/"+month+"/transactions", data, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateScheduledTransaction creates a new scheduled transaction.
func (b *PlanService) CreateScheduledTransaction(ctx context.Context, req *CreateScheduledTransactionRequest) (*ScheduledTransactionResponse, error) {
	resp := new(ScheduledTransactionResponse)
	err := b.client.MakeRequest(ctx, "POST", "/plans/"+b.id+"/scheduled_transactions", nil, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetScheduledTransaction returns a single scheduled transaction by ID.
func (b *PlanService) GetScheduledTransaction(ctx context.Context, scheduledTransactionID string) (*ScheduledTransactionResponse, error) {
	resp := new(ScheduledTransactionResponse)
	err := b.client.MakeRequest(ctx, "GET", "/plans/"+b.id+"/scheduled_transactions/"+scheduledTransactionID, nil, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateScheduledTransaction updates an existing scheduled transaction.
func (b *PlanService) UpdateScheduledTransaction(ctx context.Context, scheduledTransactionID string, req *UpdateScheduledTransactionRequest) (*ScheduledTransactionResponse, error) {
	resp := new(ScheduledTransactionResponse)
	err := b.client.PutResource(ctx, "/plans/"+b.id+"/scheduled_transactions", scheduledTransactionID, req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// DeleteScheduledTransaction deletes a scheduled transaction.
func (b *PlanService) DeleteScheduledTransaction(ctx context.Context, scheduledTransactionID string) (*ScheduledTransactionResponse, error) {
	resp := new(ScheduledTransactionResponse)
	err := b.client.MakeRequest(ctx, "DELETE", "/plans/"+b.id+"/scheduled_transactions/"+scheduledTransactionID, nil, nil, resp)
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

func (c *Client) MakeRequest(ctx context.Context, method string, pathPart string, data url.Values, reqBody any, v any) error {
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

const Version = "1.7.0"

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
	client := restclient.NewBearerClient(token, "https://api.ynab.com/v1")
	c := &Client{Client: client}
	c.Plans = func(id string) *PlanService {
		return &PlanService{
			client: c,
			id:     id,
		}
	}
	c.Budgets = c.Plans
	return c
}
