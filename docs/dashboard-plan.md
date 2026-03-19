# Dashboard Feature — Design Plan

> **Status**: DRAFT — pending review before implementation
> **Stack**: Go `html/template` · HTMX · Tailwind CSS · Chart.js
> **Target**: coder-node HTTP server (`/dashboard/*`)

---

## 1. Overview

The Dashboard is a web UI embedded inside `coder-node` that lets a server admin:

- Log in using the bootstrap token (same mechanism as `coder login`)
- View live statistics: total clients, command counts, active repos, daily activity
- Manage clients: list, inspect activity history, revoke access
- Regenerate the bootstrap token when it needs to be rotated
- Browse the full activity log with filters

Auth model mirrors the CLI client model exactly — the dashboard receives the same kind of access token, stored in an HTTP-only cookie. In open mode (`SECURE_MODE=false`) the entire dashboard is public.

---

## 2. Auth Flow

```
Admin browser                 coder-node
      │                           │
      │  GET /dashboard/          │
      │ ──────────────────────────▶
      │                           │  secure mode? cookie present?
      │                           │  NO → redirect /dashboard/login
      │◀──────────────────────────
      │
      │  GET /dashboard/login
      │ ──────────────────────────▶ serve login page (always public)
      │◀──────────────────────────
      │
      │  POST /dashboard/login
      │  { bootstrap_token: "..." }
      │ ──────────────────────────▶ mgr.RegisterClient(token, "Dashboard Admin", "dashboard@local")
      │                           │  → raw access token
      │                           │  Set-Cookie: coder_dash=<token>; HttpOnly; SameSite=Strict; Path=/dashboard
      │◀── 302 /dashboard/ ───────
      │
      │  GET /dashboard/          │
      │  Cookie: coder_dash=<token>
      │ ──────────────────────────▶ middleware: ValidateToken(cookie)
      │                           │  → OK → serve overview page
      │◀──────────────────────────
```

**Key decisions:**

| Decision | Choice | Reason |
|----------|--------|--------|
| Session storage | HTTP-only cookie | No JS access, safe against XSS; no extra session table needed |
| Token type | Same `coder_clients` row as CLI | Zero new auth code; dashboard = another registered client |
| Open mode | All dashboard pages public, no cookie check | Consistent with rest of API |
| Cookie name | `coder_dash` | Scoped to `/dashboard`, no conflict with other cookies |
| Cookie path | `/dashboard` | Limits scope, good practice |

---

## 3. Pages & Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/dashboard/` | ✓ | Redirect → `/dashboard/overview` |
| `GET` | `/dashboard/login` | public | Login page |
| `POST` | `/dashboard/login` | public | Validate bootstrap token, set cookie, redirect |
| `GET` | `/dashboard/logout` | — | Clear cookie, redirect to login |
| `GET` | `/dashboard/overview` | ✓ | Stats cards + 7-day activity chart |
| `GET` | `/dashboard/clients` | ✓ | Registered client list |
| `DELETE` | `/dashboard/clients/{id}` | ✓ | Revoke client (HTMX swap) |
| `GET` | `/dashboard/clients/{id}/activity` | ✓ | Per-client activity modal (HTMX) |
| `GET` | `/dashboard/activity` | ✓ | Global activity log (paginated, filterable) |
| `GET` | `/dashboard/settings` | ✓ | Bootstrap token management |
| `POST` | `/dashboard/settings/regenerate` | ✓ | Regenerate bootstrap token (HTMX swap) |
| `GET` | `/dashboard/stats/chart` | ✓ | JSON data for Chart.js (commands/day) |
| `GET` | `/dashboard/stats/commands` | ✓ | JSON: top commands breakdown |

`✓` = requires valid `coder_dash` cookie in secure mode; public in open mode.

---

## 4. UI Wireframes

### 4.1 Login page `/dashboard/login`

```
┌─────────────────────────────────────────────────┐
│                                                 │
│                  ██ coder                       │
│           Admin Dashboard                       │
│                                                 │
│  ┌───────────────────────────────────────────┐  │
│  │  Bootstrap Token                          │  │
│  │  ┌─────────────────────────────────────┐  │  │
│  │  │ ••••••••••••••••••••••••••••••••••  │  │  │
│  │  └─────────────────────────────────────┘  │  │
│  │                                           │  │
│  │           [ Sign In ]                     │  │
│  └───────────────────────────────────────────┘  │
│                                                 │
│  Open mode: no login required                   │
└─────────────────────────────────────────────────┘
```

### 4.2 Overview `/dashboard/overview`

```
┌──────────┬────────────────────────────────────────────────────┐
│          │  Overview                              v0.3.5  🟢  │
│ Overview │                                                     │
│ Clients  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────┐ │
│ Activity │  │ Clients  │ │ Commands │ │  Active  │ │ Repos │ │
│ Settings │  │    12    │ │   4,821  │ │  today 3 │ │   7   │ │
│          │  └──────────┘ └──────────┘ └──────────┘ └───────┘ │
│          │                                                     │
│ [Logout] │  Commands per day (last 30 days)                    │
│          │  ┌───────────────────────────────────────────────┐  │
│          │  │  📈  Chart.js line chart                      │  │
│          │  └───────────────────────────────────────────────┘  │
│          │                                                     │
│          │  Top Commands              Recent Activity          │
│          │  ┌─────────────────────┐  ┌──────────────────────┐ │
│          │  │ memory search  62%  │  │ dev@co.  skill search│ │
│          │  │ skill search   28%  │  │ 2m ago   main        │ │
│          │  │ memory store    8%  │  │ bob@co.  memory store│ │
│          │  │ skill ingest    2%  │  │ 5m ago   feat/auth   │ │
│          │  └─────────────────────┘  └──────────────────────┘ │
└──────────┴────────────────────────────────────────────────────┘
```

### 4.3 Clients `/dashboard/clients`

```
┌──────────┬────────────────────────────────────────────────────┐
│          │  Clients                                           │
│ Overview │                                                     │
│ Clients  │  ┌────────────────────────────────────────────┐    │
│ Activity │  │ Name         Email          Last seen  Act  │    │
│ Settings │  ├────────────────────────────────────────────┤    │
│          │  │ Trung Tran   dev@co.com     2m ago     [↗] [🗑]│
│          │  │ Bob Nguyen   bob@co.com     1h ago     [↗] [🗑]│
│          │  │ Dashboard    dashboard@..   just now   [↗] [🗑]│
│          │  │ Alice Le     alice@co.com   3d ago     [↗] [🗑]│
│          │  └────────────────────────────────────────────┘    │
│          │                                                     │
│          │  [↗] → opens activity drawer (HTMX)                │
│          │  [🗑] → DELETE with confirm dialog (HTMX)          │
└──────────┴────────────────────────────────────────────────────┘
```

Activity drawer (HTMX out-of-band swap):
```
┌──── Activity: dev@co.com ──────────────────┐
│  command         repo          branch  time │
│  memory search   github.com/.. main    2m   │
│  skill search    github.com/.. main    5m   │
│  memory store    github.com/.. feat/x  1h   │
│                                             │
│  [ Load more ]                              │
└─────────────────────────────────────────────┘
```

### 4.4 Activity log `/dashboard/activity`

```
┌──────────┬────────────────────────────────────────────────────┐
│          │  Activity Log                                      │
│          │                                                     │
│          │  Filter: [All clients ▾]  [All commands ▾]  [🔍]   │
│          │                                                     │
│          │  ┌──────────────────────────────────────────────┐  │
│          │  │ Time    Client       Command        Repo      │  │
│          │  ├──────────────────────────────────────────────┤  │
│          │  │ 2m ago  dev@co.com   memory search  repo/x   │  │
│          │  │ 5m ago  dev@co.com   skill search   repo/x   │  │
│          │  │ 1h ago  bob@co.com   memory store   repo/y   │  │
│          │  │ ...                                           │  │
│          │  └──────────────────────────────────────────────┘  │
│          │                                                     │
│          │  [ ← Prev ]  Page 1 / 12  [ Next → ]               │
│          │  (HTMX infinite scroll or pagination)               │
└──────────┴────────────────────────────────────────────────────┘
```

### 4.5 Settings `/dashboard/settings`

```
┌──────────┬────────────────────────────────────────────────────┐
│          │  Settings                                          │
│          │                                                     │
│          │  ┌─ Bootstrap Token ──────────────────────────┐    │
│          │  │                                            │    │
│          │  │  Status: ● Configured                      │    │
│          │  │                                            │    │
│          │  │  ⚠ Rotating the token does NOT revoke     │    │
│          │  │    existing client access tokens.          │    │
│          │  │                                            │    │
│          │  │  [ Regenerate Bootstrap Token ]            │    │
│          │  │                                            │    │
│          │  │  ┌─ New token (shown once) ─────────────┐  │    │
│          │  │  │ a3f9c2e1d4b87f...          [📋 Copy] │  │    │
│          │  │  └──────────────────────────────────────┘  │    │
│          │  │  (appears here after regenerate via HTMX)  │    │
│          │  └────────────────────────────────────────────┘    │
│          │                                                     │
│          │  ┌─ Server Info ──────────────────────────────┐    │
│          │  │  Version     v0.3.5                        │    │
│          │  │  Secure Mode true                          │    │
│          │  │  gRPC Port   50051                         │    │
│          │  │  HTTP Port   8080                          │    │
│          │  └────────────────────────────────────────────┘    │
└──────────┴────────────────────────────────────────────────────┘
```

---

## 5. File Structure

```
internal/transport/http/server/dashboard/
├── dashboard.go          router: registers all /dashboard/* routes
├── middleware.go         cookie auth middleware (dashboardAuth)
├── handler_login.go      GET /dashboard/login  +  POST /dashboard/login
├── handler_overview.go   GET /dashboard/overview  +  GET /dashboard/stats/*
├── handler_clients.go    GET /dashboard/clients  +  DELETE /dashboard/clients/{id}
│                         GET /dashboard/clients/{id}/activity
├── handler_activity.go   GET /dashboard/activity  (paginated, filtered)
├── handler_settings.go   GET /dashboard/settings  +  POST /dashboard/settings/regenerate
└── templates/
    ├── layout.html       base HTML: sidebar nav, head (Tailwind CDN, HTMX CDN)
    ├── login.html        login card
    ├── overview.html     stats cards + chart placeholder
    ├── clients.html      client table + activity drawer partial
    ├── activity.html     activity log table + pagination partial
    ├── settings.html     bootstrap token card + server info card
    └── partials/
        ├── stats_cards.html        HTMX swap target for overview cards
        ├── client_row.html         single <tr> for HTMX OOB swap after delete
        ├── activity_drawer.html    slide-in drawer content
        ├── activity_rows.html      <tbody> rows for infinite scroll / pagination
        ├── token_result.html       new token display after regenerate
        └── toast.html              success/error toast notification
```

---

## 6. Domain Changes Required

### 6.1 New methods on `AuthManager` interface

```go
// RevokeClient removes a client and all its activity records.
RevokeClient(ctx context.Context, clientID string) error

// GetAllActivities returns activity records across all clients.
// Supports pagination and optional filter by clientID or command.
GetAllActivities(ctx context.Context, filter ActivityFilter) ([]Activity, int, error)

// GetActivityStats returns aggregated data for dashboard charts.
GetActivityStats(ctx context.Context, days int) (ActivityStats, error)
```

### 6.2 New types in `internal/domain/auth/entity.go`

```go
// ActivityFilter for paginated activity queries.
type ActivityFilter struct {
    ClientID string    // "" = all clients
    Command  string    // "" = all commands
    Limit    int       // default 50
    Offset   int
}

// DailyCount is one data point for the commands-per-day chart.
type DailyCount struct {
    Date  string `json:"date"`  // "2026-03-19"
    Count int    `json:"count"`
}

// CommandCount is one slice of the top-commands breakdown.
type CommandCount struct {
    Command string  `json:"command"`
    Count   int     `json:"count"`
    Percent float64 `json:"percent"`
}

// ActivityStats bundles all chart data into one query result.
type ActivityStats struct {
    TotalClients    int            `json:"total_clients"`
    TotalCommands   int            `json:"total_commands"`
    ActiveToday     int            `json:"active_today"`
    UniqueRepos     int            `json:"unique_repos"`
    CommandsPerDay  []DailyCount   `json:"commands_per_day"`
    TopCommands     []CommandCount `json:"top_commands"`
    RecentActivity  []Activity     `json:"recent_activity"`
}
```

### 6.3 New methods on `AuthRepository` interface

```go
GetAllActivities(ctx context.Context, filter ActivityFilter) ([]Activity, int, error)
GetActivityStats(ctx context.Context, days int) (ActivityStats, error)
```

### 6.4 New SQL queries (PostgreSQL)

```sql
-- Commands per day (last N days)
SELECT DATE(timestamp)::text AS date, COUNT(*) AS count
FROM coder_client_activity
WHERE timestamp > NOW() - INTERVAL '$1 days'
GROUP BY DATE(timestamp)
ORDER BY DATE(timestamp);

-- Top commands
SELECT command, COUNT(*) AS count
FROM coder_client_activity
WHERE timestamp > NOW() - INTERVAL '$1 days'
GROUP BY command
ORDER BY count DESC
LIMIT 10;

-- Active clients today
SELECT COUNT(DISTINCT client_id)
FROM coder_client_activity
WHERE timestamp > CURRENT_DATE;

-- Unique repos
SELECT COUNT(DISTINCT repo)
FROM coder_client_activity
WHERE repo != '' AND timestamp > NOW() - INTERVAL '$1 days';

-- Paginated all-clients activity
SELECT a.id, a.client_id, a.command, a.repo, a.branch, a.timestamp,
       c.git_email
FROM coder_client_activity a
JOIN coder_clients c ON c.id = a.client_id
WHERE ($1 = '' OR a.client_id = $1)
  AND ($2 = '' OR a.command = $2)
ORDER BY a.timestamp DESC
LIMIT $3 OFFSET $4;
```

---

## 7. Dashboard-specific Auth Middleware

```go
// internal/transport/http/server/dashboard/middleware.go

// dashboardAuth wraps dashboard handlers.
// In open mode: always passes through (no cookie check).
// In secure mode: validates coder_dash cookie → 401 or redirect to /dashboard/login.
func dashboardAuth(mgr authdomain.AuthManager) func(http.Handler) http.Handler
```

Cookie spec:
```
Name:     coder_dash
Value:    <raw access token>
HttpOnly: true
SameSite: Strict
Path:     /dashboard
MaxAge:   86400 * 7  (7 days; auto-renews on each request)
Secure:   false (server has no TLS built-in; reverse proxy handles it)
```

The `dashboardAuth` middleware is **separate** from the main API `httpmiddleware.Auth`.
Dashboard cookie ≠ API Bearer header — they use separate mechanisms but validate via the same `authMgr.ValidateToken`.

---

## 8. Frontend Tech Decisions

| Concern | Choice | Notes |
|---------|--------|-------|
| Interactivity | **HTMX 1.9** via CDN | Replaces JSON fetch + JS rendering; server renders HTML partials |
| Styling | **Tailwind CSS** via CDN (play CDN) | Acceptable for admin tool; no build step; ~30 KB |
| Charts | **Chart.js 4** via CDN | Lightweight; HTMX triggers JSON fetch → `new Chart(...)` |
| Icons | **Heroicons** inline SVG | No extra dependency; embed in template |
| Theme | Dark (slate-900 bg) | Suits developer tooling |
| Templating | Go `html/template` | stdlib; partials via `{{template}}` |
| Template embed | `go:embed templates/*` | Baked into binary; no external files at runtime |

### HTMX patterns used

| Pattern | Where |
|---------|-------|
| `hx-delete` + `hx-confirm` | Revoke client button |
| `hx-swap="outerHTML"` | Replace client row after revoke |
| `hx-get` + `hx-target="#drawer"` | Open activity drawer |
| `hx-post` + `hx-swap="#token-result"` | Regenerate bootstrap token |
| `hx-get` + `hx-trigger="intersect"` | Infinite scroll on activity log |
| `hx-push-url="true"` | Update URL on filter change |
| `hx-indicator` | Loading spinner on chart refresh |

### Chart.js integration

Chart data fetched via a separate JSON endpoint (`/dashboard/stats/chart`) and rendered client-side. HTMX alone cannot render `<canvas>` charts — we use a minimal JS block:

```html
<canvas id="activityChart"></canvas>
<script>
  htmx.on("htmx:afterSettle", function() {
    fetch("/dashboard/stats/chart")
      .then(r => r.json())
      .then(data => {
        new Chart(document.getElementById("activityChart"), {
          type: "line",
          data: { labels: data.labels, datasets: [{ data: data.values }] }
        });
      });
  });
</script>
```

---

## 9. Security Model

| Scenario | Behaviour |
|----------|-----------|
| `SECURE_MODE=false` | Dashboard fully public; no cookie; all pages accessible |
| `SECURE_MODE=true`, no cookie | Redirect `GET` → `/dashboard/login`; `DELETE`/`POST` → 401 JSON |
| Valid cookie | Normal operation |
| Expired/revoked cookie | Clear cookie + redirect to login |
| Bootstrap token used for login | Creates a new `coder_clients` row (`git_email=dashboard@local`) |
| Multiple logins | Each login creates a new row; old tokens still valid until revoked |
| CSRF | `SameSite=Strict` cookie prevents cross-origin POST; sufficient for non-financial admin tool |
| XSS | `html/template` auto-escapes all output; HTMX content also server-rendered |

> **No dedicated dashboard role** — the dashboard client is a regular entry in `coder_clients`. Admins who need to identify it can filter by `git_email = 'dashboard@local'`.

---

## 10. Integration Points in `main.go`

```go
// Register dashboard routes (new block, after auth endpoints)
dashboardServer := dashboard.NewDashboardServer(authMgr, version.Version)
dashboardServer.RegisterHandlers(httpMux)

// No changes needed to the main auth middleware —
// dashboard uses its own cookie middleware internally.
```

The dashboard handler subtree wraps its own `dashboardAuth` middleware:

```go
func (d *DashboardServer) RegisterHandlers(mux *http.ServeMux) {
    // Public
    mux.HandleFunc("/dashboard/login",  d.handleLogin)
    mux.HandleFunc("/dashboard/logout", d.handleLogout)

    // Protected — wrap with dashboardAuth
    protected := dashboardAuth(d.mgr)
    mux.Handle("/dashboard/",           protected(http.HandlerFunc(d.handleRoot)))
    mux.Handle("/dashboard/overview",   protected(http.HandlerFunc(d.handleOverview)))
    mux.Handle("/dashboard/clients",    protected(http.HandlerFunc(d.handleClients)))
    mux.Handle("/dashboard/activity",   protected(http.HandlerFunc(d.handleActivity)))
    mux.Handle("/dashboard/settings",   protected(http.HandlerFunc(d.handleSettings)))
    mux.Handle("/dashboard/stats/",     protected(http.HandlerFunc(d.handleStats)))
}
```

---

## 11. Implementation Phases

### Phase 1 — Skeleton + Auth (prerequisite)
- [ ] `internal/domain/auth/entity.go` — add `ActivityFilter`, `ActivityStats`, `DailyCount`, `CommandCount`
- [ ] `internal/domain/auth/port.go` — add `RevokeClient`, `GetAllActivities`, `GetActivityStats`
- [ ] `internal/infra/postgres/auth.go` — implement new queries
- [ ] `internal/usecase/auth/manager.go` — implement new methods
- [ ] `dashboard/middleware.go` — cookie auth middleware
- [ ] `dashboard/handler_login.go` — login/logout
- [ ] Base layout template + Tailwind + HTMX wired
- [ ] `main.go` — register dashboard routes

### Phase 2 — Overview
- [ ] `dashboard/handler_overview.go` — stats query + template render
- [ ] `templates/overview.html` — stat cards + chart
- [ ] `/dashboard/stats/chart` JSON endpoint
- [ ] Chart.js integration

### Phase 3 — Client Management
- [ ] `dashboard/handler_clients.go` — list + revoke + activity drawer
- [ ] `templates/clients.html` + `partials/client_row.html`
- [ ] `partials/activity_drawer.html`
- [ ] Revoke confirm dialog (HTMX `hx-confirm`)

### Phase 4 — Activity Log
- [ ] `dashboard/handler_activity.go` — paginated + filtered
- [ ] `templates/activity.html` + `partials/activity_rows.html`
- [ ] Filter dropdowns (HTMX `hx-get` on change)

### Phase 5 — Settings
- [ ] `dashboard/handler_settings.go`
- [ ] `templates/settings.html`
- [ ] Token regenerate HTMX flow + copy-to-clipboard
- [ ] Server info section

---

## 12. Open Questions (needs decision before Phase 1)

| # | Question | Options | Recommendation |
|---|----------|---------|----------------|
| 1 | Tailwind CDN vs pre-built | CDN (easy) vs CLI build + embed | **CDN** for v1; switch to embed if offline/airgap needed |
| 2 | Infinite scroll vs pagination | Scroll (modern) vs buttons (simpler) | **Pagination** for v1 (simpler HTMX, easier testing) |
| 3 | Dashboard client identity | `git_email=dashboard@local` vs separate `role` column | **`git_email=dashboard@local`** — no schema change |
| 4 | Session max-age | 1 day vs 7 days vs no expiry | **7 days**, renewable on each request |
| 5 | Chart refresh interval | Manual only vs auto-poll | **Manual** (`hx-trigger="click"` refresh button) for v1 |
| 6 | Revoke own token | Allow or block | **Block** — dashboard should check `client.ID != current session ID` |
| 7 | Template location | Embed in binary vs read from disk | **Embed** via `go:embed` — single binary |
| 8 | Multi-admin support | One bootstrap login at a time vs multiple | **Multiple** — each login creates a new client row, all valid simultaneously |

---

## 13. What Does NOT Change

- Existing `/v1/auth/*` API endpoints — untouched
- gRPC interceptors — untouched
- `coder` CLI behaviour — untouched
- Database schema — only new queries on existing tables (except `ActivityFilter` + new aggregate queries)
- Open mode behaviour for CLI — untouched

---

## Review Checklist

Before implementation begins, please confirm:

- [ ] Auth flow (cookie-based session) is acceptable
- [ ] No dedicated `role` column in `coder_clients` — dashboard identified by `git_email=dashboard@local`
- [ ] Tailwind + HTMX + Chart.js via CDN is acceptable for v1
- [ ] Pagination (not infinite scroll) for activity log
- [ ] Phase order is correct
- [ ] Open questions 1–8 resolved
