# ynab-go

This is a You Need a Budget client but it's incomplete. Right now the only
supported endpoints are the ones necessary to do the Age of Money calculation.

### Age of Money

This command will print out more information about how old your money is.

Set `YNAB_TOKEN` to your API token in your environment. You can find your token
on the Settings page: https://app.youneedabudget.com/settings

Finally, run the binary:

```bash
go run ./cmd/ynab-age-of-money --budget='Personal Budget'
```

The output will look something like this:

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

The number in the far left hand column is the age of money, in days, for each
transaction. The thresholds show the upcoming age of money when you spend it.

The flags are:

```
  -budget-name string
    	Name of the budget to compute AOM for
  -debug
    	Enable debug
  -file string
    	Filename to read txns from
```

You will need to specify --budget-name if you have more than one budget.
`--debug` provides more information about all of the buckets you have as well as
all of the accounts you have. `--file` is useful if you are making a lot of
requests - save the JSON transaction data to a file and load it from there.

#### Errata

YNAB uses a weighted average to calculate age of money for a single transaction
if it spans multiple buckets. I choose the date of the bucket the last penny was
taken out of, so the numbers may be slightly lower here than in your dashboard.

YNAB averages the last ten transactions to get the Age of Money. I print
accurate results for each transaction in your account.

### Largest Inputs and Outputs

The `ynab-largest-inputs-outputs` command finds the largest inputs and outputs
to your Net Worth, optionally filtered by a month argument. Any income or
outflows that come into either your budget accounts or your tracking accounts
will appear here. One exception is that credit card spending is accounted at the
time of payment, not at the time the money is spent.

Pass the --month flag to filter by a given month. The flag accepts arguments in
the form of 'Jan 2006', e.g. --month='Aug 2019'.

### OpenAPI Spec

The YNAB OpenAPI spec is available at
https://api.ynab.com/papi/open_api_spec.yaml. A local copy is kept at
`open_api_spec.yaml`. Run `make update-spec` to refresh it.

### Disclaimer

I spot checked the results against my account, and they appeared to be accurate.
_They may not be correct for your account_. This software is provided "as is"
and I am not liable for any claim or damages arising from how you use this
library.

### License

MIT

### Support

You can hire me: https://burke.services

I maintain this software in my free time. Donations free up time to make
improvements to the library, and respond to bug reports. You can send donations
via Paypal's "Send Money" feature to kev@inburke.com. Donations are not tax
deductible in the USA.
