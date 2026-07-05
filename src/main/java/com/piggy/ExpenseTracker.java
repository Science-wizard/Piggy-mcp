package com.piggy;

import java.util.List;

public class ExpenseTracker {
    private final JsonExpenseStore store;

    public ExpenseTracker(String storageFile) {
        this.store = new JsonExpenseStore(storageFile);
    }

    public ExpenseTracker() {
        this("expenses.json");
    }

    public void addExpense(Expense expense) {
        store.addExpense(expense);
    }

    public List<Expense> listExpenses() {
        return store.listExpenses();
    }

    public double getTotal() {
        return store.getTotal();
    }

    public double getCategoryTotal(String category) {
        return store.getCategoryTotal(category);
    }
}
