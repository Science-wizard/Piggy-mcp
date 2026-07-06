package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"piggy-mcp/internal/expenses"
	"piggy-mcp/internal/mcp"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("piggy", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	storePath := flags.String("store", defaultStorePath(), "path to SQLite database file")
	if err := flags.Parse(args); err != nil {
		return err
	}

	store, err := expenses.NewStore(*storePath)
	if err != nil {
		return err
	}
	defer store.Close()

	remaining := flags.Args()
	if len(remaining) == 0 || remaining[0] == "mcp" {
		return mcp.NewServer(store).Serve(os.Stdin, os.Stdout)
	}

	switch remaining[0] {
	case "add":
		return addExpense(store, remaining[1:])
	case "edit":
		return editExpense(store, remaining[1:])
	case "delete":
		return deleteExpense(store, remaining[1:])
	case "search":
		return searchExpenses(store, remaining[1:])
	case "list":
		return printJSON(map[string]any{"expenses": store.List()})
	case "summary":
		return printJSON(store.Summary())
	case "monthly-report":
		return monthlyReport(store, remaining[1:])
	case "category-report":
		return categoryReport(store, remaining[1:])
	case "highest":
		expense, err := store.HighestExpense()
		if err != nil {
			return err
		}
		return printJSON(map[string]any{"expense": expense})
	case "lowest":
		expense, err := store.LowestExpense()
		if err != nil {
			return err
		}
		return printJSON(map[string]any{"expense": expense})
	case "backup":
		destination := ""
		if len(remaining) > 1 {
			destination = remaining[1]
		}
		path, err := store.Backup(destination)
		if err != nil {
			return err
		}
		return printJSON(map[string]any{"message": "Database backed up", "path": path})
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\nRun: piggy help", remaining[0])
	}
}

func addExpense(store *expenses.Store, args []string) error {
	flags := flag.NewFlagSet("piggy add", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	description := flags.String("description", "", "expense description")
	amount := flags.Float64("amount", 0, "expense amount")
	category := flags.String("category", "", "expense category")
	date := flags.String("date", "", "optional date in YYYY-MM-DD format")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *description == "" && flags.NArg() >= 3 {
		*description = flags.Arg(0)
		parsed, err := strconv.ParseFloat(flags.Arg(1), 64)
		if err != nil {
			return fmt.Errorf("amount must be a number: %w", err)
		}
		*amount = parsed
		*category = flags.Arg(2)
		if flags.NArg() >= 4 {
			*date = flags.Arg(3)
		}
	}

	expense, err := store.Add(*description, *amount, *category, *date)
	if err != nil {
		return err
	}
	return printJSON(map[string]any{
		"message": "Expense added",
		"expense": expense,
	})
}

func editExpense(store *expenses.Store, args []string) error {
	flags := flag.NewFlagSet("piggy edit", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	id := flags.String("id", "", "expense id")
	description := flags.String("description", "", "new expense description")
	amount := flags.Float64("amount", 0, "new expense amount")
	category := flags.String("category", "", "new expense category")
	date := flags.String("date", "", "new date in YYYY-MM-DD format")
	if err := flags.Parse(args); err != nil {
		return err
	}

	var descriptionPtr *string
	var amountPtr *float64
	var categoryPtr *string
	var datePtr *string
	flags.Visit(func(flag *flag.Flag) {
		switch flag.Name {
		case "description":
			descriptionPtr = description
		case "amount":
			amountPtr = amount
		case "category":
			categoryPtr = category
		case "date":
			datePtr = date
		}
	})

	expense, err := store.Edit(*id, descriptionPtr, amountPtr, categoryPtr, datePtr)
	if err != nil {
		return err
	}
	return printJSON(map[string]any{"message": "Expense updated", "expense": expense})
}

func deleteExpense(store *expenses.Store, args []string) error {
	flags := flag.NewFlagSet("piggy delete", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	id := flags.String("id", "", "expense id")
	if err := flags.Parse(args); err != nil {
		return err
	}
	expense, err := store.Delete(*id)
	if err != nil {
		return err
	}
	return printJSON(map[string]any{"message": "Expense deleted", "expense": expense})
}

func searchExpenses(store *expenses.Store, args []string) error {
	flags := flag.NewFlagSet("piggy search", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	query := flags.String("query", "", "search query")
	category := flags.String("category", "", "category filter")
	startDate := flags.String("start-date", "", "start date in YYYY-MM-DD format")
	endDate := flags.String("end-date", "", "end date in YYYY-MM-DD format")
	minAmount := flags.Float64("min-amount", 0, "minimum amount")
	maxAmount := flags.Float64("max-amount", 0, "maximum amount")
	limit := flags.Int("limit", 100, "maximum results")
	if err := flags.Parse(args); err != nil {
		return err
	}

	var minPtr *float64
	var maxPtr *float64
	flags.Visit(func(flag *flag.Flag) {
		switch flag.Name {
		case "min-amount":
			minPtr = minAmount
		case "max-amount":
			maxPtr = maxAmount
		}
	})

	expenses, err := store.Search(expenses.SearchOptions{
		Query:     *query,
		Category:  *category,
		StartDate: *startDate,
		EndDate:   *endDate,
		MinAmount: minPtr,
		MaxAmount: maxPtr,
		Limit:     *limit,
	})
	if err != nil {
		return err
	}
	return printJSON(map[string]any{"expenses": expenses, "count": len(expenses)})
}

func monthlyReport(store *expenses.Store, args []string) error {
	month := ""
	if len(args) > 0 {
		month = args[0]
	}
	report, err := store.MonthlyReport(month)
	if err != nil {
		return err
	}
	return printJSON(report)
}

func categoryReport(store *expenses.Store, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("category is required")
	}
	report, err := store.CategoryReport(args[0])
	if err != nil {
		return err
	}
	return printJSON(report)
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printHelp() {
	fmt.Println(strings.TrimSpace(`
Piggy expense tracker

Usage:
  piggy mcp
  piggy -store expenses.db mcp
  piggy add -description "Lunch" -amount 12.50 -category Food -date 2026-07-06
  piggy add Lunch 12.50 Food 2026-07-06
  piggy edit -id 123 -amount 15
  piggy delete -id 123
  piggy search -query lunch -category Food
  piggy list
  piggy summary
  piggy monthly-report 2026-07
  piggy category-report Food
  piggy highest
  piggy lowest
  piggy backup backup.db

For AI clients, configure the command as:
  bin/piggy -store /absolute/path/to/expenses.db mcp
`))
}

func defaultStorePath() string {
	if value := strings.TrimSpace(os.Getenv("PIGGY_DB_PATH")); value != "" {
		return value
	}

	workingDir, err := os.Getwd()
	if err == nil {
		if fileExists(filepath.Join(workingDir, "go.mod")) {
			return filepath.Join(workingDir, "expenses.db")
		}
	}

	executable, err := os.Executable()
	if err != nil {
		return "expenses.db"
	}
	executablePath, err := filepath.EvalSymlinks(executable)
	if err != nil {
		executablePath = executable
	}

	executableDir := filepath.Dir(executablePath)
	if filepath.Base(executableDir) == "bin" {
		parent := filepath.Dir(executableDir)
		if fileExists(filepath.Join(parent, "go.mod")) || fileExists(filepath.Join(parent, "expenses.db")) {
			return filepath.Join(parent, "expenses.db")
		}
	}
	return filepath.Join(executableDir, "expenses.db")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
