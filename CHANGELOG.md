# Changelog & Upgrade Guide

### v1.5.0 (2026-02-18)

Add endpoints covering most of the remaining YNAB API surface.

New BudgetService methods:

- `GetBudget` - full budget export with all months and categories
- `GetSettings` - budget date/currency format settings
- `GetAccount`, `CreateAccount` - single account operations
- `GetCategory`, `UpdateCategory` - single category operations
- `GetMonthCategory`, `UpdateMonthCategory` - per-month category budgeting
- `Payees`, `GetPayee`, `UpdatePayee` - payee operations
- `PayeeLocations`, `GetPayeeLocation`, `PayeeLocationsByPayee` - payee location operations
- `Months`, `GetMonth` - budget month operations
- `GetTransaction`, `UpdateTransactions` (bulk), `ImportTransactions` - additional transaction operations
- `AccountTransactions`, `CategoryTransactions`, `PayeeTransactions`, `MonthTransactions` - filtered transaction lists
- `GetScheduledTransaction`, `CreateScheduledTransaction`, `UpdateScheduledTransaction`, `DeleteScheduledTransaction` - scheduled transaction CRUD

New Client method:

- `GetUser` - get the authenticated user

New types: `MonthDetail`, `MonthSummary`, `BudgetDetail`, `HybridTransaction`, `Payee`, `PayeeLocation`, `BudgetSettings`, `DateFormat`, `CurrencyFormat`, `User`, `SaveScheduledTransaction`, and related request/response wrappers.

Other changes:

- Add `Balance` field to `Category`
- Add `PayeeID` field to `Transaction`
- Add doc comments to struct fields throughout
- Fix `MakeRequest` to set Content-Type for PATCH requests

### v1.4.0 (2026-01-16)

- Add `DeleteTransaction` endpoint
- Add `NewTransferTransaction` and `UpdateTransactionToTransfer` helpers for creating and converting transfers
- Add `TransferAccountID` field to `NewTransaction`
- Add `TransferPayeeID` field to `Account`
- Add `Error` type for client-side errors
- Change `Transaction.Cleared` from `string` to `ClearedStatus`

### v1.3.0 (2025-08-07)

Make the User-Agent header modifiable via `Client.UserAgent`.

### v1.2.0 (2025-07-22)

Change `Date` to a non-optional (non-pointer) field in transaction types.

### v1.1.0 (2025-07-22)

Add CreateTransaction endpoint.

### v1.0.0 (2025-07-22)

The main change is that BudgetService now requires a budget ID at creation time, simplifying the API by removing the need to pass budget IDs to individual methods.

Breaking Changes & Upgrade Instructions

1. Budget Service Access Pattern

Before:
client.Budgets.GetAccounts(ctx, budgetID, data)

After:
client.Budgets(budgetID).Accounts(ctx, data)

2. Budget Listing

Before:
budgets, err := client.Budgets.GetPage(ctx, data)

After:
budgets, err := client.GetBudgets(ctx, data)

3. Account Operations

Before:
accounts, err := client.Budgets.GetAccounts(ctx, budgetID, data)

After:
accounts, err := client.Budgets(budgetID).Accounts(ctx, data)

4. Transaction Operations

Before:
// Getting transactions
transactions, err := client.Budgets.GetTransactions(ctx, budgetID, data)

// Updating transactions
resp, err := client.Transactions.UpdateTransaction(ctx, budgetID, transactionID, req)

After:
// Getting transactions
transactions, err := client.Budgets(budgetID).Transactions(ctx, data)

// Updating transactions
resp, err := client.Budgets(budgetID).UpdateTransaction(ctx, transactionID, req)

5. Scheduled Transactions

Before:
scheduled, err := client.Budgets.GetScheduledTransactions(ctx, budgetID, data)

After:
scheduled, err := client.Budgets(budgetID).ScheduledTransactions(ctx, data)

6. Categories

Before:
categories, err := client.Budgets.GetCategories(ctx, budgetID, data)

After:
categories, err := client.Budgets(budgetID).Categories(ctx, data)

Summary of Method Changes

| Old Method                                                  | New Method                                             | Notes                        |
|-------------------------------------------------------------|--------------------------------------------------------|------------------------------|
| client.Budgets.GetPage()                                    | client.GetBudgets()                                    | Moved to client directly     |
| client.Budgets.GetAccounts(budgetID, ...)                   | client.Budgets(budgetID).Accounts(...)                 | Budget ID now in constructor |
| client.Budgets.GetTransactions(budgetID, ...)               | client.Budgets(budgetID).Transactions(...)             | Budget ID now in constructor |
| client.Budgets.GetScheduledTransactions(budgetID, ...)      | client.Budgets(budgetID).ScheduledTransactions(...)    | Budget ID now in constructor |
| client.Budgets.GetCategories(budgetID, ...)                 | client.Budgets(budgetID).Categories(...)               | Budget ID now in constructor |
| client.Transactions.UpdateTransaction(budgetID, txnID, ...) | client.Budgets(budgetID).UpdateTransaction(txnID, ...) | Moved to BudgetService       |
