# ynab-go

A complete, well-typed Go client for the [YNAB (You Need a Budget)
API](https://api.ynab.com), plus a handful of ready-to-run command line tools
for analyzing and exporting your budget.

- **Full API coverage.** Every YNAB endpoint and every response field is
  supported — accounts, transactions, scheduled transactions, categories,
  payees, payee locations, months, money movements, settings, and the user
  endpoint.
- **Strongly typed.** Money is handled in milliunits, dates round-trip through a
  dedicated `Date` type, and nullable fields use explicit `NullString` /
  `NullDate` types instead of bare pointers, so you never guess what a zero
  value means.
- **Batteries included.** Helpers like `NewTransferTransaction` and
  `UpdateTransactionToTransfer` handle the fiddly parts of the API (transfer
  payee IDs, preserving fields on conversion) for you.
- **Three CLI tools out of the box.** Compute your Age of Money per
  transaction, find the largest inflows/outflows to your net worth, and export
  transactions to CSV — no code required.
- **Stable and maintained.** Backwards-compatible aliases are kept when YNAB
  renames things (see Budgets vs. Plans below), and the client sets a versioned
  User-Agent by default.

## Why use it

If you want to script against your own budget, build a dashboard, or pull your
data out of YNAB, this library gives you typed access to the entire API with
sensible defaults. If you just want answers about your money, the bundled
commands work without writing any Go.

## Install

The library:

```bash
go get github.com/kevinburke/ynab-go
```

The command line tools:

```bash
go install github.com/kevinburke/ynab-go/ynab-age-of-money@latest
go install github.com/kevinburke/ynab-go/ynab-largest-inputs-outputs@latest
go install github.com/kevinburke/ynab-go/ynab-export-transactions@latest
```

All tools read your API token from the `YNAB_TOKEN` environment variable. Create
a Personal Access Token on the [YNAB settings
page](https://app.youneedabudget.com/settings) and export it:

```bash
export YNAB_TOKEN=your-token-here
```

## Library quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/kevinburke/ynab-go"
)

func main() {
	client := ynab.NewClient(os.Getenv("YNAB_TOKEN"))
	ctx := context.Background()

	// List your plans (budgets).
	plans, err := client.GetPlans(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, plan := range plans.Data.Plans {
		fmt.Printf("%s (%s)\n", plan.Name, plan.ID)

		// List the accounts in each plan.
		accounts, err := client.Plans(plan.ID).Accounts(ctx, url.Values{})
		if err != nil {
			log.Fatal(err)
		}
		for _, acct := range accounts.Data.Accounts {
			// Balances are in milliunits: 1000 milliunits = $1.00.
			fmt.Printf("  %-20s %.2f\n", acct.Name, float64(acct.Balance)/1000)
		}
	}
}
```

See [`example_test.go`](example_test.go) for runnable examples covering creating
transactions, transfers, and converting an existing transaction into a transfer.

### Budgets vs. Plans

YNAB now refers to budgets as *plans* in the API. New code should use
`Client.Plans`, `Client.GetPlans`, and the `Plan` types. The old `Budget` names
(`Client.Budgets`, `Client.GetBudgets`, `Budget`, ...) remain as
compatibility aliases where practical, so existing code keeps working.

## Command line tools

### Age of Money

`ynab-age-of-money` prints detailed Age of Money information for each
transaction, rather than the single averaged number YNAB shows on your
dashboard.

```bash
ynab-age-of-money --budget-name='Personal Budget'
```

The output looks like this:

```
 70 earned: 2019-02-07 spent: 2019-04-18     $25.00 Cash Exxon Mobil
 71 earned: 2019-02-07 spent: 2019-04-19      $3.00 Cash Shaska Cafe
 71 earned: 2019-02-07 spent: 2019-04-19      $2.00 Cash Shaska Cafe
 72 earned: 2019-02-07 spent: 2019-04-20      $4.00 Cash Aces
 75 earned: 2019-02-07 spent: 2019-04-23      $6.00 Cash Corner Store
 75 earned: 2019-02-07 spent: 2019-04-23      $3.00 Cash Lava Java
 76 earned: 2019-02-07 spent: 2019-04-24    $123.45 Shared Checking Transfer : Credit Card 1
 76 earned: 2019-02-07 spent: 2019-04-24    $456.78 Shared Checking Transfer : Credit Card 2
 77 earned: 2019-02-07 spent: 2019-04-25      $3.00 Cash Lava Java
 77 earned: 2019-02-07 spent: 2019-04-25      $2.00 Cash Corner Store

upcoming spending thresholds (and age if you spent today):
 78 2019-02-07   $1000.00 Personal Checking Employer 1
 74 2019-02-11   $2000.00 Shared Checking Employer 2
 73 2019-02-12   $2020.00 Personal Checking Venmo
 71 2019-02-14   $3020.00 Shared Checking Employer 2
 66 2019-02-19   $3060.00 Cash Laura Cash
 60 2019-02-25   $3065.00 Shared Checking Bank Interest
 60 2019-02-25   $3070.00 Personal Checking Bank Interest
```

The number in the far left column is the age of money, in days, for each
transaction. The thresholds show the upcoming age of money when you spend it.

The flags are:

```
  -budget-name string
    	Name of the budget to compute AOM for
  -debug
    	Enable debug
  -file string
    	Filename to read txns from
  -include-scheduled-income
    	Include scheduled income
```

You need to specify `--budget-name` if you have more than one budget. `--debug`
prints more information about all of your buckets and accounts. `--file` is
useful if you are making a lot of requests — save the JSON transaction data to a
file and load it from there.

**Errata:** YNAB uses a weighted average to calculate age of money for a single
transaction that spans multiple buckets. This tool chooses the date of the
bucket the last penny was taken out of, so its numbers may be slightly lower
than your dashboard. YNAB also averages the last ten transactions to get the Age
of Money; this tool prints accurate results for each transaction in your
account.

### Largest Inputs and Outputs

`ynab-largest-inputs-outputs` finds the largest inputs and outputs to your net
worth, optionally filtered by month. Any income or outflow that touches either
your budget accounts or your tracking accounts appears here. (One exception:
credit card spending is accounted at the time of payment, not at the time the
money is spent.)

```bash
ynab-largest-inputs-outputs --month='Aug 2019'
```

Pass `--month` to filter by a given month. The flag accepts arguments in the
form `Jan 2006`, e.g. `--month='Aug 2019'`. Use `--year` to filter by year,
`--exclude` to drop a comma-separated list of accounts, and `--budget-name` to
choose a budget.

### Export Transactions

`ynab-export-transactions` retrieves transactions and prints them to stdout in
CSV format.

```bash
ynab-export-transactions --budget-name='Personal Budget' > transactions.csv
```

Use `--start` (an RFC 3339 timestamp) and `--category` to filter the
transactions returned, and `--budget-name` to choose a budget.

## OpenAPI spec

The YNAB OpenAPI spec is available at
https://api.ynab.com/papi/open_api_spec.yaml. A local copy is kept at
[`open_api_spec.yaml`](open_api_spec.yaml). Run `make update-spec` to refresh
it.

## Disclaimer

I spot checked the results against my account and they appeared to be accurate.
_They may not be correct for your account._ This software is provided "as is" and
I am not liable for any claim or damages arising from how you use this library.

## License

MIT. See [LICENSE](LICENSE).

## Support

You can hire me: https://burke.services

I maintain this software in my free time. Donations free up time to make
improvements to the library and respond to bug reports. You can [Sponsor me
on Github][sponsor], or send donations via PayPal's "Send Money" feature to
kevin@burke.dev. Donations are not tax deductible in the USA.

[sponsor]: https://github.com/sponsors/kevinburke
