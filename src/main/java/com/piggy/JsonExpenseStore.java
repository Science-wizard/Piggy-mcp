package com.piggy;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;

public class JsonExpenseStore {
    private final Path storagePath;
    private final ObjectMapper objectMapper;
    private List<Expense> expenses = new ArrayList<>();

    public JsonExpenseStore(String storageFile) {
        this.storagePath = Paths.get(storageFile);
        this.objectMapper = new ObjectMapper();
        load();
    }

    public synchronized void addExpense(Expense expense) {
        expenses.add(expense);
        persist();
    }

    public synchronized List<Expense> listExpenses() {
        return new ArrayList<>(expenses);
    }

    public synchronized double getTotal() {
        double total = 0.0;
        for (Expense expense : expenses) {
            total += expense.getAmount();
        }
        return total;
    }

    public synchronized double getCategoryTotal(String category) {
        double total = 0.0;
        for (Expense expense : expenses) {
            if (expense.getCategory().equalsIgnoreCase(category)) {
                total += expense.getAmount();
            }
        }
        return total;
    }

    private void load() {
        if (!Files.exists(storagePath)) {
            expenses = new ArrayList<>();
            return;
        }

        try {
            String content = new String(Files.readAllBytes(storagePath), StandardCharsets.UTF_8);
            if (content.trim().isEmpty()) {
                expenses = new ArrayList<>();
                return;
            }
            expenses = objectMapper.readValue(content, new TypeReference<List<Expense>>() {
            });
        } catch (IOException ex) {
            expenses = new ArrayList<>();
        }
    }

    private void persist() {
        try {
            Path parent = storagePath.toAbsolutePath().getParent();
            if (parent != null) {
                Files.createDirectories(parent);
            }
            objectMapper.writeValue(storagePath.toFile(), expenses);
        } catch (IOException ex) {
            throw new IllegalStateException("Unable to persist expenses", ex);
        }
    }
}
