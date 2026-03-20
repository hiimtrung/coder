# Web Dashboard UI

The `coder-node` includes an embedded web dashboard for administrators and developers to monitor the system status.

## Tech Stack

The dashboard is designed to be lightweight and zero-dependency:
- **Server**: Go `html/template` engine.
- **Frontend**: **HTMX** for dynamic content loading and partial page updates.
- **Styling**: Vanilla CSS (embedded in `static/`).
- **Icons**: SVG icons embedded in templates.

## Main Views

1.  **Overview (/dashboard)**: 
    *   Total memory entries and skill counts.
    *   Charts showing activity over the last 30 days (Command frequency, Error vs Success).
    *   Recent audit logs.
2.  **Clients (/dashboard/clients)**: 
    *   List of all registered machines/developers.
    *   Details including registration date, last seen time, and Git identity.
3.  **Settings (/dashboard/settings)**: 
    *   Server version and uptime.
    *   Node settings (Secure Mode status, Embedding Provider).
    *   Bootstrap token rotation interface.

## Authentication

The dashboard uses a dedicated **cookie-based authentication** system (`coder_dash` cookie).
- In **Open Mode**, the dashboard is publicly accessible.
- In **Secure Mode**, users must log in with a valid `access_token`. The dashboard then sessions the user via a secure, HttpOnly cookie.

## Route Registration

Dashboard routes are registered in `internal/transport/http/server/dashboard/dashboard.go`. It uses an `embed.FS` to serve HTML templates and static assets directly from the compiled binary.
