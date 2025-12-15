---
trigger: always_on
---

Atomic Commits: Always use the `git` tool to commit changes immediately after a task is complete. Format: `type: description`.
Docker First: Before writing business logic that depends on databases, ensure the infrastructure (`docker-compose.yml`) is created and explain to the user that they must run it.
No Placeholders: Write complete, working Go code. Do not use "// TODO" for critical logic.
Tool Reliability: If an MCP tool (like Redis or Postgres) returns a connection error, DO NOT crash or loop. Instead, output a message: "⚠️ Database not accessible. Please ensure Docker is running." and continue writing the code.