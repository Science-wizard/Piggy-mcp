package com.piggy;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

import java.io.File;
import org.junit.Test;

public class ExpenseTrackerTest {

    @Test
    public void shouldAddExpensesAndSummarizeTotals() {
        String storePath = tempStorePath("tracker-summary");
        new File(storePath).delete();
        ExpenseTracker tracker = new ExpenseTracker(storePath);

        tracker.addExpense(new Expense("Coffee", 4.50, "Food", "2026-07-06"));
        tracker.addExpense(new Expense("Taxi", 18.00, "Transport", "2026-07-06"));

        assertEquals(2, tracker.listExpenses().size());
        assertEquals(22.50, tracker.getTotal(), 0.001);
        assertEquals(4.50, tracker.getCategoryTotal("Food"), 0.001);
    }

    @Test
    public void shouldHandleMcpToolCallRequests() {
        String storePath = tempStorePath("mcp-tool-call");
        new File(storePath).delete();
        McpExpenseServer server = new McpExpenseServer(storePath);
        String response = server.handleRequest("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"add_expense\",\"arguments\":{\"description\":\"Lunch\",\"amount\":\"12.75\",\"category\":\"Food\",\"date\":\"2026-07-06\"}}}");

        assertTrue(response.contains("Added expense"));
    }

    @Test
    public void shouldPersistExpensesAcrossServerInstances() {
        String storePath = tempStorePath("persisted-expenses");
        new File(storePath).delete();

        McpExpenseServer first = new McpExpenseServer(storePath);
        first.handleRequest("add Dinner 15.50 Food 2026-07-06");

        McpExpenseServer second = new McpExpenseServer(storePath);
        assertEquals(1, second.listExpenses().size());
    }

    private String tempStorePath(String name) {
        return "target/" + name + ".json";
    }
}
