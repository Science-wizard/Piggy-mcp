package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"piggy-mcp/internal/expenses"
)

const protocolVersion = "2024-11-05"

type Server struct {
	store *expenses.Store
}

func NewServer(store *expenses.Store) *Server {
	return &Server{store: store}
}

func (s *Server) Serve(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	encoder := json.NewEncoder(w)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		response, shouldRespond := s.Handle(line)
		if !shouldRespond {
			continue
		}
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) Handle(payload []byte) (map[string]any, bool) {
	var request rpcRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return errorResponse(nil, -32700, "Parse error"), true
	}

	if request.ID == nil {
		s.handleNotification(request)
		return nil, false
	}

	switch request.Method {
	case "initialize":
		return resultResponse(request.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"serverInfo": map[string]any{
				"name":    "piggy-expense-mcp",
				"version": "1.0.0",
			},
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
		}), true
	case "tools/list":
		return resultResponse(request.ID, map[string]any{
			"tools": toolDefinitions(),
		}), true
	case "tools/call":
		return s.handleToolCall(request), true
	default:
		return errorResponse(request.ID, -32601, "Method not found"), true
	}
}

func (s *Server) handleNotification(request rpcRequest) {
	_ = request
}

func (s *Server) handleToolCall(request rpcRequest) map[string]any {
	var params toolCallParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return errorResponse(request.ID, -32602, "Invalid tools/call params")
	}

	switch params.Name {
	case "add_expense":
		var args addExpenseArgs
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return toolErrorResponse(request.ID, "Invalid add_expense arguments")
		}
		expense, err := s.store.Add(args.Description, args.Amount, args.Category, args.Date)
		if err != nil {
			return toolErrorResponse(request.ID, err.Error())
		}
		return toolResultResponse(request.ID, map[string]any{
			"message": "Expense added",
			"expense": expense,
		})
	case "edit_expense":
		var args editExpenseArgs
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return toolErrorResponse(request.ID, "Invalid edit_expense arguments")
		}
		expense, err := s.store.Edit(args.ID, args.Description, args.Amount, args.Category, args.Date)
		if err != nil {
			return toolErrorResponse(request.ID, err.Error())
		}
		return toolResultResponse(request.ID, map[string]any{
			"message": "Expense updated",
			"expense": expense,
		})
	case "delete_expense":
		var args deleteExpenseArgs
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return toolErrorResponse(request.ID, "Invalid delete_expense arguments")
		}
		expense, err := s.store.Delete(args.ID)
		if err != nil {
			return toolErrorResponse(request.ID, err.Error())
		}
		return toolResultResponse(request.ID, map[string]any{
			"message": "Expense deleted",
			"expense": expense,
		})
	case "search_expenses":
		var args searchExpenseArgs
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return toolErrorResponse(request.ID, "Invalid search_expenses arguments")
		}
		expenses, err := s.store.Search(expenses.SearchOptions{
			Query:     args.Query,
			Category:  args.Category,
			StartDate: args.StartDate,
			EndDate:   args.EndDate,
			MinAmount: args.MinAmount,
			MaxAmount: args.MaxAmount,
			Limit:     args.Limit,
		})
		if err != nil {
			return toolErrorResponse(request.ID, err.Error())
		}
		return toolResultResponse(request.ID, map[string]any{
			"expenses": expenses,
			"count":    len(expenses),
		})
	case "list_expenses":
		return toolResultResponse(request.ID, map[string]any{
			"expenses": s.store.List(),
		})
	case "get_summary":
		return toolResultResponse(request.ID, s.store.Summary())
	case "monthly_report":
		var args monthlyReportArgs
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return toolErrorResponse(request.ID, "Invalid monthly_report arguments")
		}
		report, err := s.store.MonthlyReport(args.Month)
		if err != nil {
			return toolErrorResponse(request.ID, err.Error())
		}
		return toolResultResponse(request.ID, report)
	case "category_report":
		var args categoryReportArgs
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return toolErrorResponse(request.ID, "Invalid category_report arguments")
		}
		report, err := s.store.CategoryReport(args.Category)
		if err != nil {
			return toolErrorResponse(request.ID, err.Error())
		}
		return toolResultResponse(request.ID, report)
	case "highest_expense":
		expense, err := s.store.HighestExpense()
		if err != nil {
			return toolErrorResponse(request.ID, err.Error())
		}
		return toolResultResponse(request.ID, map[string]any{"expense": expense})
	case "lowest_expense":
		expense, err := s.store.LowestExpense()
		if err != nil {
			return toolErrorResponse(request.ID, err.Error())
		}
		return toolResultResponse(request.ID, map[string]any{"expense": expense})
	case "backup_database":
		var args backupDatabaseArgs
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return toolErrorResponse(request.ID, "Invalid backup_database arguments")
		}
		path, err := s.store.Backup(args.Destination)
		if err != nil {
			return toolErrorResponse(request.ID, err.Error())
		}
		return toolResultResponse(request.ID, map[string]any{
			"message": "Database backed up",
			"path":    path,
		})
	case "reset_database":
		var args resetDatabaseArgs
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return toolErrorResponse(request.ID, "Invalid reset_database arguments")
		}
		deleted, err := s.store.Reset(args.Confirm)
		if err != nil {
			return toolErrorResponse(request.ID, err.Error())
		}
		return toolResultResponse(request.ID, map[string]any{
			"message": "Database reset",
			"deleted": deleted,
		})
	default:
		return toolErrorResponse(request.ID, fmt.Sprintf("Unknown tool: %s", params.Name))
	}
}

func toolDefinitions() []map[string]any {
	noArgs := map[string]any{"type": "object", "properties": map[string]any{}}
	return []map[string]any{
		{
			"name":        "add_expense",
			"description": "Add one expense to the local expense tracker.",
			"inputSchema": schema([]string{"description", "amount", "category"}, map[string]any{
				"description": textProp("Short expense description, for example Lunch or Metro ticket."),
				"amount":      numberProp("Expense amount as a positive number."),
				"category":    textProp("Expense category, for example Food, Travel, Bills, Shopping."),
				"date":        textProp("Optional date in YYYY-MM-DD format. Defaults to today."),
			}),
		},
		{
			"name":        "edit_expense",
			"description": "Edit an existing expense by id. Provide only the fields that should change.",
			"inputSchema": schema([]string{"id"}, map[string]any{
				"id":          textProp("Expense id from list_expenses or search_expenses."),
				"description": textProp("New description."),
				"amount":      numberProp("New positive amount."),
				"category":    textProp("New category."),
				"date":        textProp("New date in YYYY-MM-DD format."),
			}),
		},
		{
			"name":        "delete_expense",
			"description": "Delete an expense by id.",
			"inputSchema": schema([]string{"id"}, map[string]any{
				"id": textProp("Expense id from list_expenses or search_expenses."),
			}),
		},
		{
			"name":        "search_expenses",
			"description": "Search and filter expenses by text, category, date range, amount range, and limit.",
			"inputSchema": schema(nil, map[string]any{
				"query":      textProp("Text to search in description or category."),
				"category":   textProp("Exact category filter."),
				"start_date": textProp("Earliest date in YYYY-MM-DD format."),
				"end_date":   textProp("Latest date in YYYY-MM-DD format."),
				"min_amount": numberProp("Minimum amount."),
				"max_amount": numberProp("Maximum amount."),
				"limit":      map[string]any{"type": "integer", "description": "Maximum results, defaults to 100."},
			}),
		},
		{"name": "list_expenses", "description": "List recent tracked expenses.", "inputSchema": noArgs},
		{"name": "get_summary", "description": "Get total spend, expense count, and totals by category.", "inputSchema": noArgs},
		{
			"name":        "monthly_report",
			"description": "Get spending report for a month.",
			"inputSchema": schema(nil, map[string]any{
				"month": textProp("Month in YYYY-MM format. Defaults to current month."),
			}),
		},
		{
			"name":        "category_report",
			"description": "Get spending report for one category.",
			"inputSchema": schema([]string{"category"}, map[string]any{
				"category": textProp("Category name, for example Food."),
			}),
		},
		{"name": "highest_expense", "description": "Return the highest single expense.", "inputSchema": noArgs},
		{"name": "lowest_expense", "description": "Return the lowest single expense.", "inputSchema": noArgs},
		{
			"name":        "backup_database",
			"description": "Create a copy of the SQLite database.",
			"inputSchema": schema(nil, map[string]any{
				"destination": textProp("Optional backup file path. Defaults beside the database."),
			}),
		},
		{
			"name":        "reset_database",
			"description": "Delete all expenses. Requires confirm=true.",
			"inputSchema": schema([]string{"confirm"}, map[string]any{
				"confirm": map[string]any{"type": "boolean", "description": "Must be true to reset the database."},
			}),
		},
	}
}

func schema(required []string, properties map[string]any) map[string]any {
	result := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

func textProp(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func numberProp(description string) map[string]any {
	return map[string]any{"type": "number", "description": description}
}

func toolResultResponse(id any, data any) map[string]any {
	content, _ := json.MarshalIndent(data, "", "  ")
	return resultResponse(id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(content)},
		},
	})
}

func toolErrorResponse(id any, message string) map[string]any {
	return resultResponse(id, map[string]any{
		"isError": true,
		"content": []map[string]any{
			{"type": "text", "text": message},
		},
	})
}

func resultResponse(id any, result any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
}

func errorResponse(id any, code int, message string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type addExpenseArgs struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Date        string  `json:"date"`
}

type editExpenseArgs struct {
	ID          string   `json:"id"`
	Description *string  `json:"description"`
	Amount      *float64 `json:"amount"`
	Category    *string  `json:"category"`
	Date        *string  `json:"date"`
}

type deleteExpenseArgs struct {
	ID string `json:"id"`
}

type searchExpenseArgs struct {
	Query     string   `json:"query"`
	Category  string   `json:"category"`
	StartDate string   `json:"start_date"`
	EndDate   string   `json:"end_date"`
	MinAmount *float64 `json:"min_amount"`
	MaxAmount *float64 `json:"max_amount"`
	Limit     int      `json:"limit"`
}

type monthlyReportArgs struct {
	Month string `json:"month"`
}

type categoryReportArgs struct {
	Category string `json:"category"`
}

type backupDatabaseArgs struct {
	Destination string `json:"destination"`
}

type resetDatabaseArgs struct {
	Confirm bool `json:"confirm"`
}
