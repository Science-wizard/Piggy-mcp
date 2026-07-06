package expenses

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	path string
	db   *sql.DB
}

func NewStore(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		path = "expenses.db"
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	store := &Store{path: path, db: db}
	if err := store.init(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Add(description string, amount float64, category string, date string) (Expense, error) {
	expense, err := validateExpense("", description, amount, category, date)
	if err != nil {
		return Expense{}, err
	}
	expense.ID = fmt.Sprintf("%d", time.Now().UnixNano())

	_, err = s.db.Exec(`
		INSERT INTO expenses (id, description, amount, category, date)
		VALUES (?, ?, ?, ?, ?)
	`, expense.ID, expense.Description, expense.Amount, expense.Category, expense.Date)
	if err != nil {
		return Expense{}, err
	}
	return expense, nil
}

func (s *Store) Edit(id string, description *string, amount *float64, category *string, date *string) (Expense, error) {
	current, err := s.Get(id)
	if err != nil {
		return Expense{}, err
	}

	nextDescription := current.Description
	nextAmount := current.Amount
	nextCategory := current.Category
	nextDate := current.Date
	if description != nil {
		nextDescription = *description
	}
	if amount != nil {
		nextAmount = *amount
	}
	if category != nil {
		nextCategory = *category
	}
	if date != nil {
		nextDate = *date
	}

	updated, err := validateExpense(current.ID, nextDescription, nextAmount, nextCategory, nextDate)
	if err != nil {
		return Expense{}, err
	}
	_, err = s.db.Exec(`
		UPDATE expenses
		SET description = ?, amount = ?, category = ?, date = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, updated.Description, updated.Amount, updated.Category, updated.Date, updated.ID)
	if err != nil {
		return Expense{}, err
	}
	return updated, nil
}

func (s *Store) Delete(id string) (Expense, error) {
	expense, err := s.Get(id)
	if err != nil {
		return Expense{}, err
	}
	result, err := s.db.Exec(`DELETE FROM expenses WHERE id = ?`, id)
	if err != nil {
		return Expense{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Expense{}, err
	}
	if affected == 0 {
		return Expense{}, fmt.Errorf("expense not found: %s", id)
	}
	return expense, nil
}

func (s *Store) Get(id string) (Expense, error) {
	row := s.db.QueryRow(`
		SELECT id, description, amount, category, date
		FROM expenses
		WHERE id = ?
	`, strings.TrimSpace(id))
	return scanExpense(row)
}

func (s *Store) List() []Expense {
	expenses, _ := s.Search(SearchOptions{Limit: 200})
	return expenses
}

func (s *Store) Search(options SearchOptions) ([]Expense, error) {
	where := []string{"1 = 1"}
	args := []any{}

	if strings.TrimSpace(options.Query) != "" {
		where = append(where, "(LOWER(description) LIKE LOWER(?) OR LOWER(category) LIKE LOWER(?))")
		like := "%" + strings.TrimSpace(options.Query) + "%"
		args = append(args, like, like)
	}
	if strings.TrimSpace(options.Category) != "" {
		where = append(where, "LOWER(category) = LOWER(?)")
		args = append(args, strings.TrimSpace(options.Category))
	}
	if strings.TrimSpace(options.StartDate) != "" {
		if err := validateDate(options.StartDate); err != nil {
			return nil, err
		}
		where = append(where, "date >= ?")
		args = append(args, options.StartDate)
	}
	if strings.TrimSpace(options.EndDate) != "" {
		if err := validateDate(options.EndDate); err != nil {
			return nil, err
		}
		where = append(where, "date <= ?")
		args = append(args, options.EndDate)
	}
	if options.MinAmount != nil {
		where = append(where, "amount >= ?")
		args = append(args, *options.MinAmount)
	}
	if options.MaxAmount != nil {
		where = append(where, "amount <= ?")
		args = append(args, *options.MaxAmount)
	}

	limit := options.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	args = append(args, limit)

	rows, err := s.db.Query(fmt.Sprintf(`
		SELECT id, description, amount, category, date
		FROM expenses
		WHERE %s
		ORDER BY date DESC, created_at DESC
		LIMIT ?
	`, strings.Join(where, " AND ")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanExpenses(rows)
}

func (s *Store) Summary() Summary {
	rows, err := s.db.Query(`SELECT category, COUNT(*), COALESCE(SUM(amount), 0) FROM expenses GROUP BY category`)
	if err != nil {
		return Summary{ByCategory: map[string]float64{}}
	}
	defer rows.Close()

	summary := Summary{ByCategory: map[string]float64{}}
	for rows.Next() {
		var category string
		var count int
		var total float64
		if err := rows.Scan(&category, &count, &total); err != nil {
			continue
		}
		summary.Count += count
		summary.Total += total
		summary.ByCategory[category] = total
	}
	return summary
}

func (s *Store) MonthlyReport(month string) (MonthlyReport, error) {
	month = strings.TrimSpace(month)
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	if _, err := time.Parse("2006-01", month); err != nil {
		return MonthlyReport{}, fmt.Errorf("month must use YYYY-MM format: %w", err)
	}
	start, _ := time.Parse("2006-01-02", month+"-01")
	end := start.AddDate(0, 1, -1)

	expenses, err := s.Search(SearchOptions{
		StartDate: start.Format(time.DateOnly),
		EndDate:   end.Format(time.DateOnly),
		Limit:     500,
	})
	if err != nil {
		return MonthlyReport{}, err
	}

	report := MonthlyReport{
		Month:      month,
		ByCategory: map[string]float64{},
		Expenses:   expenses,
	}
	for _, expense := range expenses {
		report.Count++
		report.Total += expense.Amount
		report.ByCategory[expense.Category] += expense.Amount
	}
	return report, nil
}

func (s *Store) CategoryReport(category string) (CategoryReport, error) {
	category = strings.TrimSpace(category)
	if category == "" {
		return CategoryReport{}, errors.New("category is required")
	}
	expenses, err := s.Search(SearchOptions{Category: category, Limit: 500})
	if err != nil {
		return CategoryReport{}, err
	}
	report := CategoryReport{Category: category, Expenses: expenses}
	for _, expense := range expenses {
		report.Count++
		report.Total += expense.Amount
	}
	return report, nil
}

func (s *Store) HighestExpense() (Expense, error) {
	return s.singleByAmount("DESC")
}

func (s *Store) LowestExpense() (Expense, error) {
	return s.singleByAmount("ASC")
}

func (s *Store) Backup(destination string) (string, error) {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		destination = fmt.Sprintf("%s.backup-%s", s.path, time.Now().Format("20060102-150405"))
	}
	if dir := filepath.Dir(destination); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
	}
	if _, err := s.db.Exec(`PRAGMA wal_checkpoint(FULL)`); err != nil {
		return "", err
	}

	src, err := os.Open(s.path)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(destination)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return destination, nil
}

func (s *Store) Reset(confirm bool) (int64, error) {
	if !confirm {
		return 0, errors.New("reset requires confirm=true")
	}
	result, err := s.db.Exec(`DELETE FROM expenses`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS expenses (
			id TEXT PRIMARY KEY,
			description TEXT NOT NULL,
			amount REAL NOT NULL CHECK(amount > 0),
			category TEXT NOT NULL,
			date TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(date);
		CREATE INDEX IF NOT EXISTS idx_expenses_category ON expenses(category);
		CREATE INDEX IF NOT EXISTS idx_expenses_amount ON expenses(amount);
	`)
	return err
}

func (s *Store) singleByAmount(direction string) (Expense, error) {
	if direction != "ASC" {
		direction = "DESC"
	}
	row := s.db.QueryRow(fmt.Sprintf(`
		SELECT id, description, amount, category, date
		FROM expenses
		ORDER BY amount %s, date DESC
		LIMIT 1
	`, direction))
	return scanExpense(row)
}

func validateExpense(id string, description string, amount float64, category string, date string) (Expense, error) {
	description = strings.TrimSpace(description)
	category = strings.TrimSpace(category)
	date = strings.TrimSpace(date)

	if description == "" {
		return Expense{}, errors.New("description is required")
	}
	if amount <= 0 {
		return Expense{}, errors.New("amount must be greater than zero")
	}
	if category == "" {
		return Expense{}, errors.New("category is required")
	}
	if date == "" {
		date = time.Now().Format(time.DateOnly)
	}
	if err := validateDate(date); err != nil {
		return Expense{}, err
	}

	return Expense{
		ID:          strings.TrimSpace(id),
		Description: description,
		Amount:      amount,
		Category:    category,
		Date:        date,
	}, nil
}

func validateDate(date string) error {
	if _, err := time.Parse(time.DateOnly, strings.TrimSpace(date)); err != nil {
		return fmt.Errorf("date must use YYYY-MM-DD format: %w", err)
	}
	return nil
}

func scanExpense(scanner interface {
	Scan(dest ...any) error
}) (Expense, error) {
	var expense Expense
	if err := scanner.Scan(&expense.ID, &expense.Description, &expense.Amount, &expense.Category, &expense.Date); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Expense{}, errors.New("expense not found")
		}
		return Expense{}, err
	}
	return expense, nil
}

func scanExpenses(rows *sql.Rows) ([]Expense, error) {
	expenses := []Expense{}
	for rows.Next() {
		expense, err := scanExpense(rows)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, expense)
	}
	return expenses, rows.Err()
}
