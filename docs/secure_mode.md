# Secure Mode & Authentication

The `coder-node` provides a built-in security layer called **Secure Mode**. When enabled, every gRPC and HTTP request must carry a valid `Bearer` access token.

## How it works

1.  **SECURE_MODE Toggle**: Enabled via the `SECURE_MODE=true` environment variable in `docker-compose.yml`.
2.  **Bootstrap Token**: On the very first run of a secure node, the server generates a cryptographically secure `bootstrap_token`. This token is printed to `stdout` and is required for the first client to register.
3.  **Client Registration**: 
    *   Developers register their machine via `coder login`.
    *   The CLI sends the `bootstrap_token` (or a valid access token if rotating) along with the machine's Git identity (Name/Email).
    *   The server registers the client and returns a unique `access_token`.
4.  **Token Validation**:
    *   **gRPC**: Handled by `UnaryAuth` and `StreamAuth` interceptors.
    *   **HTTP**: Handled by the `Auth` middleware.
    *   Tokens are validated against the PostgreSQL `clients` table.
5.  **Context Propagation**: Once validated, the `*authdomain.Client` entity is attached to the request context, allowing downstream services to identify the developer or machine making the request.

## Public Endpoints

Even in secure mode, the following endpoints remain public to allow for health checks and registration:
- `/health`: Node health and secure mode status.
- `/v1/auth/register-client`: For registration via bootstrap token.
- `/v1/auth/bootstrap/status`: To check if a bootstrap token is still pending.

## Token Rotation

For improved security, clients can rotate their access tokens at any time using `coder token rotate`. This invalidates the old token and issues a new one immediately.
