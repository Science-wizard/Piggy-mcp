package expenses

type Expense struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Date        string  `json:"date"`
}

type Summary struct {
	Total      float64            `json:"total"`
	Count      int                `json:"count"`
	ByCategory map[string]float64 `json:"byCategory"`
}

type SearchOptions struct {
	Query     string
	Category  string
	StartDate string
	EndDate   string
	MinAmount *float64
	MaxAmount *float64
	Limit     int
}

type MonthlyReport struct {
	Month      string             `json:"month"`
	Total      float64            `json:"total"`
	Count      int                `json:"count"`
	ByCategory map[string]float64 `json:"byCategory"`
	Expenses   []Expense          `json:"expenses"`
}

type CategoryReport struct {
	Category string    `json:"category"`
	Total    float64   `json:"total"`
	Count    int       `json:"count"`
	Expenses []Expense `json:"expenses"`
}
