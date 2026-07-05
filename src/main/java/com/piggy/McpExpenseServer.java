package com.piggy;

import java.util.Arrays;
import java.util.List;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

public class McpExpenseServer {
    private final ExpenseTracker tracker;

    public McpExpenseServer() {
        this("expenses.json");
    }

    public McpExpenseServer(String storageFile) {
        this.tracker = new ExpenseTracker(storageFile);
    }

    public String handleRequest(String request) {
        if (request == null || request.trim().isEmpty()) {
            return errorResponse("Empty request");
        }

        String trimmed = request.trim();
        if (trimmed.startsWith("{")) {
            return handleJsonRequest(trimmed);
        }

        String normalized = trimmed.toLowerCase();
        if (normalized.startsWith("add")) {
            return addExpenseFromText(trimmed);
        }
        if (normalized.startsWith("summary")) {
            return summaryResponse();
        }
        if (normalized.startsWith("list")) {
            return listResponse();
        }

        return errorResponse("Unknown request. Try: add <description> <amount> <category> [date], summary, list, or a JSON-RPC style MCP request.");
    }

    private String handleJsonRequest(String request) {
        if (request.contains("\"method\":\"initialize\"")) {
            return initializeResponse();
        }

        if (request.contains("\"method\":\"tools/list\"")) {
            return toolsListResponse();
        }

        if (request.contains("\"method\":\"tools/call\"")) {
            String toolName = extractJsonString(request, "name");
            if ("add_expense".equals(toolName)) {
                String description = extractJsonMember(request, "description");
                String amount = extractJsonMember(request, "amount");
                String category = extractJsonMember(request, "category");
                String date = extractJsonMember(request, "date");

                if (description == null || amount == null || category == null) {
                    return errorResponse("Missing required arguments for add_expense");
                }

                double parsedAmount;
                try {
                    parsedAmount = Double.parseDouble(amount);
                } catch (NumberFormatException ex) {
                    return errorResponse("Amount must be a number");
                }

                tracker.addExpense(new Expense(description, parsedAmount, category, date == null ? "2026-07-06" : date));
                return successResponse("Added expense: " + description + " for " + parsedAmount + " in " + category);
            }

            if ("get_summary".equals(toolName)) {
                return successResponse(summaryResponse());
            }

            if ("list_expenses".equals(toolName)) {
                return successResponse(listResponse());
            }
        }

        return errorResponse("Unsupported MCP request");
    }

    private String extractJsonString(String request, String fieldName) {
        Pattern pattern = Pattern.compile("\\\"" + fieldName + "\\\"\\s*:\\s*\\\"([^\\\"]*)\\\"");
        Matcher matcher = pattern.matcher(request);
        if (matcher.find()) {
            return matcher.group(1);
        }
        return null;
    }

    private String extractJsonMember(String request, String fieldName) {
        String value = extractJsonString(request, fieldName);
        if (value != null) {
            return value;
        }
        Pattern pattern = Pattern.compile("\\\"" + fieldName + "\\\"\\s*:\\s*([^,}]+)");
        Matcher matcher = pattern.matcher(request);
        if (matcher.find()) {
            return matcher.group(1).trim().replaceAll("^\"|\"$", "");
        }
        return null;
    }

    private String initializeResponse() {
        return "{\"jsonrpc\":\"2.0\",\"result\":{\"protocolVersion\":\"2024-11-05\",\"serverInfo\":{\"name\":\"piggy-expense-mcp\",\"version\":\"1.0.0\"},\"capabilities\":{\"tools\":{}}}}";
    }

    private String successResponse(String text) {
        return "{\"jsonrpc\":\"2.0\",\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"" + escape(text) + "\"}]}}";
    }

    private String errorResponse(String text) {
        return "{\"jsonrpc\":\"2.0\",\"error\":{\"message\":\"" + escape(text) + "\"}}";
    }

    private String toolsListResponse() {
        return "{\"jsonrpc\":\"2.0\",\"result\":{\"tools\":[{\"name\":\"add_expense\",\"description\":\"Add a new expense\"},{\"name\":\"list_expenses\",\"description\":\"List all recorded expenses\"},{\"name\":\"get_summary\",\"description\":\"Show the total spend\"}]}}";
    }

    private String addExpenseFromText(String request) {
        List<String> parts = Arrays.asList(request.trim().split("\\s+"));
        if (parts.size() < 4) {
            return "Usage: add <description> <amount> <category> [date]";
        }

        String description = parts.get(1);
        double amount;
        try {
            amount = Double.parseDouble(parts.get(2));
        } catch (NumberFormatException ex) {
            return "Amount must be a number.";
        }

        String category = parts.get(3);
        String date = parts.size() > 4 ? parts.get(4) : "2026-07-06";

        tracker.addExpense(new Expense(description, amount, category, date));
        return "Added expense: " + description + " for " + amount + " in " + category;
    }

    public List<Expense> listExpenses() {
        return tracker.listExpenses();
    }

    private String summaryResponse() {
        return "Total spent: " + tracker.getTotal();
    }

    private String listResponse() {
        List<Expense> expenses = tracker.listExpenses();
        if (expenses.isEmpty()) {
            return "No expenses recorded yet.";
        }

        StringBuilder builder = new StringBuilder("Expenses:\n");
        for (Expense expense : expenses) {
            builder.append("- ")
                    .append(expense.getDescription())
                    .append(" | ")
                    .append(expense.getAmount())
                    .append(" | ")
                    .append(expense.getCategory())
                    .append(" | ")
                    .append(expense.getDate())
                    .append("\n");
        }
        return builder.toString();
    }

    private String escape(String value) {
        return value.replace("\\", "\\\\").replace("\"", "\\\"");
    }
}
