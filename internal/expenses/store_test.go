package expenses

import (
	"path/filepath"
	"testing"
)

func TestStoreAddsPersistsAndSummarizesExpenses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "expenses.db")

	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add("Coffee", 4.5, "Food", "2026-07-06"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add("Taxi", 18, "Travel", "2026-07-06"); err != nil {
		t.Fatal(err)
	}

	summary := store.Summary()
	if summary.Count != 2 {
		t.Fatalf("count = %d, want 2", summary.Count)
	}
	if summary.Total != 22.5 {
		t.Fatalf("total = %f, want 22.5", summary.Total)
	}
	if summary.ByCategory["Food"] != 4.5 {
		t.Fatalf("food total = %f, want 4.5", summary.ByCategory["Food"])
	}

	reloaded, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(reloaded.List()); got != 2 {
		t.Fatalf("reloaded expense count = %d, want 2", got)
	}
}

func TestStoreValidatesInput(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "expenses.db"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Add("", 10, "Food", "2026-07-06"); err == nil {
		t.Fatal("expected empty description to fail")
	}
	if _, err := store.Add("Lunch", 0, "Food", "2026-07-06"); err == nil {
		t.Fatal("expected zero amount to fail")
	}
	if _, err := store.Add("Lunch", 10, "", "2026-07-06"); err == nil {
		t.Fatal("expected empty category to fail")
	}
	if _, err := store.Add("Lunch", 10, "Food", "06-07-2026"); err == nil {
		t.Fatal("expected invalid date to fail")
	}
}

func TestStoreEditsDeletesSearchesAndReports(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "expenses.db"))
	if err != nil {
		t.Fatal(err)
	}
	coffee, err := store.Add("Coffee", 4.5, "Food", "2026-07-06")
	if err != nil {
		t.Fatal(err)
	}
	taxi, err := store.Add("Taxi", 18, "Travel", "2026-07-07")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add("Rent", 800, "Bills", "2026-07-01"); err != nil {
		t.Fatal(err)
	}

	amount := 5.25
	updated, err := store.Edit(coffee.ID, nil, &amount, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Amount != 5.25 {
		t.Fatalf("updated amount = %f, want 5.25", updated.Amount)
	}

	searchResults, err := store.Search(SearchOptions{Query: "tax", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(searchResults) != 1 || searchResults[0].ID != taxi.ID {
		t.Fatalf("unexpected search results: %#v", searchResults)
	}

	monthly, err := store.MonthlyReport("2026-07")
	if err != nil {
		t.Fatal(err)
	}
	if monthly.Count != 3 {
		t.Fatalf("monthly count = %d, want 3", monthly.Count)
	}

	category, err := store.CategoryReport("Food")
	if err != nil {
		t.Fatal(err)
	}
	if category.Count != 1 || category.Total != 5.25 {
		t.Fatalf("unexpected category report: %#v", category)
	}

	highest, err := store.HighestExpense()
	if err != nil {
		t.Fatal(err)
	}
	if highest.Description != "Rent" {
		t.Fatalf("highest = %s, want Rent", highest.Description)
	}

	lowest, err := store.LowestExpense()
	if err != nil {
		t.Fatal(err)
	}
	if lowest.Description != "Coffee" {
		t.Fatalf("lowest = %s, want Coffee", lowest.Description)
	}

	deleted, err := store.Delete(taxi.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.ID != taxi.ID {
		t.Fatalf("deleted id = %s, want %s", deleted.ID, taxi.ID)
	}

	if _, err := store.Reset(false); err == nil {
		t.Fatal("expected reset without confirmation to fail")
	}
	removed, err := store.Reset(true)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 {
		t.Fatalf("reset removed = %d, want 2", removed)
	}
}
