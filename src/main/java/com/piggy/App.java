package com.piggy;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;

public class App {
    public static void main(String[] args) throws IOException {
        McpExpenseServer server = new McpExpenseServer("expenses.json");
        System.out.println("🐷 Piggy MCP Server");
        System.out.println("Starting...");
        System.out.println("Waiting for AI requests...");

        if (args.length > 0) {
            System.out.println(server.handleRequest(String.join(" ", args)));
            return;
        }

        BufferedReader reader = new BufferedReader(new InputStreamReader(System.in));
        String line;
        while ((line = reader.readLine()) != null) {
            if (line.trim().isEmpty()) {
                continue;
            }
            System.out.println(server.handleRequest(line));
        }
    }
}
