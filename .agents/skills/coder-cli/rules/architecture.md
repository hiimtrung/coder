# Rule: coder CLI Architecture

## Layer Separation (MUST follow)

```
cmd/coder/*.go          ← CLI only: flag parsing, user I/O, call HTTP client
                            ❌ NO direct DB access
                            ❌ NO direct Ollama calls
                            ✅ YES: httpclient.*, os.ReadFile, bufio.Scanner

internal/transport/http/client/*.go
                        ← HTTP client wrappers for coder-node API
                            Each command has its own client struct
                            Use ChatStream() for streaming responses

internal/transport/http/server/*.go
                        ← HTTP handlers in coder-node
                            Register with httpMux.HandleFunc
                            Thin: decode request → call use case → encode response

internal/usecase/*/manager.go
                        ← Business logic: prompt building, context injection, parsing
                            No framework dependencies
                            Returns domain types (not HTTP types)

internal/domain/*/entity.go + port.go
                        ← Pure types and interfaces
                            No imports from infra or transport

internal/infra/llm/ + postgres/ + embedding/
                        ← All external I/O: Ollama, PostgreSQL, pgvector
```

## Adding a New AI Command

When adding a new command (e.g. `coder summarize`):

1. **Domain layer** (`internal/domain/summarize/`):
   - `entity.go` — request/response types
   - `port.go` — Manager interface

2. **Use case layer** (`internal/usecase/summarize/manager.go`):
   - Inject context (memory + skill search, 300ms parallel goroutines)
   - Build prompt, call LLMProvider
   - Return domain type

3. **Transport: server** (`internal/transport/http/server/summarize.go`):
   - `POST /v1/summarize` handler
   - Register in `cmd/coder-node/main.go`

4. **Transport: client** (`internal/transport/http/client/summarize.go`):
   - Typed HTTP client calling `/v1/summarize`

5. **CLI** (`cmd/coder/cmd_summarize.go`):
   - `runSummarize(args []string)`
   - Flag parsing with `flag.NewFlagSet`
   - Call `logActivity("summarize")`
   - Add to `cmd/coder/main.go` switch

6. **Wire in main** (`cmd/coder-node/main.go`):
   - Create manager, server, register handlers

## Naming Conventions

| Layer | Pattern | Example |
|-------|---------|---------|
| Command func | `runXxx(args []string)` | `runReview` |
| HTTP handler struct | `XxxServer` | `ReviewServer` |
| Use case manager | `Manager` in `usecase/xxx` | `ucreview.Manager` |
| HTTP client | `XxxClient` | `ReviewClient` |
| Domain request | `XxxRequest` | `ReviewRequest` |
| Domain result | `XxxResult` | `ReviewResult` |
