# Piggy MCP Expense Tracker

Piggy is a local Go expense tracker with an MCP stdio server. An AI client can call the tools and update a local SQLite database for you, so you can message naturally instead of manually entering every expense.

## Run the CLI

```bash
go run ./cmd/piggy add -description "Lunch" -amount 12.50 -category Food
go run ./cmd/piggy list
go run ./cmd/piggy summary
```

The default store is `expenses.db` in the current directory. Use `-store` to choose another SQLite database file.

```bash
go run ./cmd/piggy -store /Users/abhishek/Piggy/mcp/piggy-mcp/expenses.db summary
```

Build a reusable binary:

```bash
go build -o bin/piggy ./cmd/piggy
```

## Run as an MCP server

Use this command in an MCP-capable AI client:

```bash
/Users/abhishek/Piggy/mcp/piggy-mcp/bin/piggy -store /Users/abhishek/Piggy/mcp/piggy-mcp/expenses.db mcp
```

The server exposes these tools:

- `add_expense`: add an expense with `description`, `amount`, `category`, and optional `date`
- `edit_expense`: update description, amount, category, or date by expense id
- `delete_expense`: delete an expense by id
- `search_expenses`: search/filter by text, category, date range, amount range, and limit
- `list_expenses`: list saved expenses
- `get_summary`: total spend, count, and totals by category
- `monthly_report`: report for a month like `2026-07`
- `category_report`: report for a category like `Food`
- `highest_expense`: highest single expense
- `lowest_expense`: lowest single expense
- `backup_database`: copy the SQLite database to a backup file
- `reset_database`: delete all expenses, requiring `confirm: true`

## Example MCP client config

```json
{
  "mcpServers": {
    "piggy": {
      "command": "/Users/abhishek/Piggy/mcp/piggy-mcp/bin/piggy",
      "args": [
        "-store",
        "/Users/abhishek/Piggy/mcp/piggy-mcp/expenses.db",
        "mcp"
      ]
    }
  }
}
```

After that, you can ask your AI client things like:

> Add 250 rupees for dinner under Food.

> How much have I spent by category?

> Show my latest tracked expenses.

> Find my Food expenses in July.

> Backup my expense database.
