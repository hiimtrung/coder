---
name: error-code-standards
description: Standardized error codes and exception handling patterns across all languages and services. Ensures consistent error reporting and recovery strategies.
applyTo: "**/{domain,application,infrastructure}/**"

---

# Error Codes & Exception Handling Standards

## Error Code Format & Categories

**Format**: `{CATEGORY}_{DESCRIPTIVE_NAME}`

All error codes must follow this pattern across **all languages and services** (NestJS, Java, Go, Rust, Spring Boot).

### Categories & HTTP Status Codes:

| Category   | HTTP Status | Use Case                                      | Recovery                          |
| ---------- | ----------- | --------------------------------------------- | --------------------------------- |
| `AUTH_*`   | 401, 403    | Authentication & Authorization failures       | Re-login, request permission      |
| `VAL_*`    | 400         | Input validation failures                     | Correct input and retry           |
| `BIZ_*`    | 400, 404, 409 | Business logic violations, entity not found   | Verify data, check permissions    |
| `INF_*`    | 500, 502, 503 | Infrastructure/external service failures      | Retry with backoff, check status  |
| `SYS_*`    | 500         | System/configuration errors                   | Check logs, restart/scale service |

---

## Authentication & Authorization (40x)

| Code                     | HTTP | When Thrown                                 | Recovery Action                  |
| ------------------------ | ---- | ------------------------------------------- | -------------------------------- |
| `AUTH_INVALID_TOKEN`     | 401  | JWT expired, invalid signature, or tampered | Refresh token or re-authenticate |
| `AUTH_PERMISSION_DENIED` | 403  | User lacks permission for resource          | Check role/permissions           |
| `AUTH_COMPANY_NOT_FOUND` | 404  | Company doesn't exist in Auth Server         | Verify company ID in JWT         |
| `AUTH_USER_NOT_FOUND`    | 404  | User doesn't exist in Auth Server            | Verify user exists in Console    |

### Pattern (All Languages):

**TypeScript/NestJS**:
```typescript
if (!user) throw new UserNotFoundException(userId);
if (!hasPermission) throw new PermissionDeniedException('admin_access_required');
````

**Java**:

```java
if (user == null) throw new UserNotFoundException(userId);
if (!hasPermission) throw new PermissionDeniedException("admin_access_required");
```

**Go**:

```go
if user == nil {
  return nil, &DomainError{Code: "AUTH_USER_NOT_FOUND", ...}
}
```

---

## Validation (400)

| Code                         | HTTP | When Thrown                        | Recovery Action              |
| ---------------------------- | ---- | ---------------------------------- | ---------------------------- |
| `VAL_MISSING_REQUIRED_FIELD` | 400  | Required field in DTO is missing   | Add missing field            |
| `VAL_INVALID_EMAIL_FORMAT`   | 400  | Email doesn't match email pattern  | Use format: user@example.com |
| `VAL_INVALID_PHONE_FORMAT`   | 400  | Phone doesn't match expected regex | Use international format     |
| `VAL_INVALID_CHANNEL_TYPE`   | 400  | Channel type doesn't exist         | Use valid type from enum     |
| `VAL_INVALID_MESSAGE_TYPE`   | 400  | Message type doesn't exist         | Use valid type from enum     |
| `VAL_INVALID_JSON`           | 400  | Request body not valid JSON        | Check JSON syntax            |
| `VAL_FIELD_TOO_LONG`         | 400  | Field exceeds max length           | Reduce field length          |
| `VAL_FIELD_TOO_SHORT`        | 400  | Field below min length             | Increase field length        |

### When Thrown:

- **DTO Validation** (@Validated, @Valid in Java; class-validator in NestJS)
- **Business Constraint Violation** (email uniqueness, phone format, enum validation)
- **Parsing Errors** (invalid JSON, malformed request)

### Pattern:

```typescript
// TypeScript
if (!dto.name?.trim()) throw new InvalidNameException();
if (!isValidEmail(dto.email)) throw new InvalidEmailFormatException();

// Java
if (dto.name() == null || dto.name().isBlank())
  throw new InvalidNameException();
if (!isValidEmail(dto.email())) throw new InvalidEmailFormatException();
```

---

## Business Logic (400, 404, 409)

| Code                           | HTTP | When Thrown                                    | Recovery Action                 |
| ------------------------------ | ---- | ---------------------------------------------- | ------------------------------- |
| `BIZ_ENTITY_NOT_FOUND`         | 404  | Entity (Channel, Contact, etc.) doesn't exist  | Verify ID, list entities        |
| `BIZ_ENTITY_ALREADY_EXISTS`    | 409  | Entity with same value already exists          | Use existing or delete original |
| `BIZ_INVALID_STATE_TRANSITION` | 400  | Cannot transition to requested state           | Check current state, use valid  |
| `BIZ_INSUFFICIENT_CREDENTIALS` | 400  | Channel missing API credentials                | Configure channel               |
| `BIZ_CHANNEL_DISABLED`         | 400  | Channel is disabled or archived                | Enable channel or use active    |
| `BIZ_CONVERSATION_CLOSED`      | 400  | Conversation is closed, no new messages        | Reopen or create new            |
| `BIZ_INSUFFICIENT_PERMISSIONS` | 403  | User lacks permission within company scope     | Check user role assignments     |
| `BIZ_QUOTA_EXCEEDED`           | 429  | Usage quota exceeded (messages, contacts, etc) | Upgrade plan or wait for reset  |
| `BIZ_DUPLICATE_EXTERNAL_ID`    | 409  | External ID already mapped to other entity     | Use unique external ID          |

### When Thrown:

- **Repository Query Empty** (findById returns null)
- **Unique Constraint** (duplicate email, duplicate external ID)
- **State Machine Violation** (invalid state transition)
- **Business Rule Check** (disabled channel, closed conversation)
- **Multi-Tenant Isolation** (access to another company's data)

### Pattern:

```typescript
// TypeScript
const channel = await this.repo.findById(companyId, channelId);
if (!channel) throw new ChannelNotFoundException(channelId);

// Java
Channel channel = repository.findByIdAndCompanyId(channelId, companyId)
  .orElseThrow(() -> new ChannelNotFoundException(channelId));

// Go
channel, err := repo.FindByIDAndCompanyID(ctx, channelId, companyId)
if channel == nil {
  return nil, &ChannelNotFound{ChannelID: channelId}
}
```

---

## Infrastructure (500, 502, 503)

| Code                      | HTTP | When Thrown                             | Recovery Action                |
| ------------------------- | ---- | --------------------------------------- | ------------------------------ |
| `INF_DATABASE_ERROR`      | 500  | PostgreSQL query/connection failed      | Check DB health, retry         |
| `INF_MONGODB_ERROR`       | 500  | MongoDB query/connection failed         | Check MongoDB health, retry    |
| `INF_REDIS_ERROR`         | 503  | Redis cache unavailable                 | Check Redis, restart if needed |
| `INF_EXTERNAL_API_ERROR`  | 502  | Facebook, WhatsApp, LINE API failed     | Check API status, retry        |
| `INF_ENCRYPTION_ERROR`    | 500  | Credential encryption/decryption failed | Check encryption keys          |
| `INF_MESSAGE_QUEUE_ERROR` | 500  | RabbitMQ, Bull, Kafka queue failed      | Check broker health            |
| `INF_NETWORK_ERROR`       | 503  | Network request timeout or connection   | Check network, retry with prep |
| `INF_TIMEOUT`             | 504  | Operation exceeded timeout              | Increase timeout, optimize     |
| `INF_OUT_OF_MEMORY`       | 500  | Server out of memory                    | Restart or scale up            |

### When Thrown:

- **Database Operations** (connection pool exhausted, query timeout)
- **External APIs** (third-party service down)
- **Network Issues** (timeout, connection refused)
- **Resource Exhaustion** (memory, file descriptors)

### Pattern (All Languages):

```typescript
// TypeScript - Always catch and wrap
try {
  const response = await this.externalApi.call();
} catch (error) {
  const msg = error instanceof Error ? error.message : 'Unknown error';
  throw new ExternalApiException(`External API failed: ${msg}`);
}

// Java
try {
  response = channelClient.send(request);
} catch (Exception e) {
  throw new ExternalApiException("External API failed: " + e.getMessage(), e);
}

// Go
response, err := client.Send(ctx, request)
if err != nil {
  return nil, fmt.Errorf("send to external API: %w", err)
}
```

---

## System (500)

| Code                      | HTTP | When Thrown                        | Recovery Action              |
| ------------------------- | ---- | ---------------------------------- | ---------------------------- |
| `SYS_CONFIGURATION_ERROR` | 500  | Missing/invalid environment config | Set required env variables   |
| `SYS_INTERNAL_ERROR`      | 500  | Unhandled exception occurred       | Check logs, report bug       |
| `SYS_NOT_IMPLEMENTED`     | 501  | Feature not yet implemented        | Use alternative, wait update |
| `SYS_MAINTENANCE_MODE`    | 503  | Server in maintenance window       | Check status page            |

### When Thrown:

- **Missing Env Variables** (database URL, API keys)
- **Unexpected Exceptions** (unhandled code path)
- **Not Implemented** (feature flag disabled, work in progress)

---

## Error Code Definition Checklist

When creating a new error code:

- [ ] Choose correct category (AUTH, VAL, BIZ, INF, SYS)
- [ ] Define error code as enum value
- [ ] Create specific exception class
- [ ] Include message with context
- [ ] Set correct HTTP status code
- [ ] Add details map with debugging info
- [ ] Document in error codes registry
- [ ] Update client API documentation
- [ ] Add test case for error scenario

### Implementation Example (TypeScript):

```typescript
// 1. Define in enum
export enum ErrorCode {
  BIZ_CHANNEL_DISABLED = 'BIZ_CHANNEL_DISABLED',
}

// 2. Create exception class
export class ChannelDisabledException extends DomainException {
  constructor(channelId: string) {
    super(
      ErrorCode.BIZ_CHANNEL_DISABLED,
      `Channel '${channelId}' is disabled`,
      400,
      { channelId },
    );
  }
}

// 3. Throw from use case
async execute(companyId: string, channelId: string): Promise<void> {
  const channel = await this.repo.findById(companyId, channelId);
  if (!channel) throw new ChannelNotFoundException(channelId);
  if (!channel.isActive) throw new ChannelDisabledException(channelId);
  // ... rest of logic
}

// 4. Controller auto-catches via DomainExceptionFilter
@Post('channels/:id/activate')
async activate(@Param('id') channelId: string) {
  const result = await this.useCase.execute(channelId);
  return { data: result };
}
```

### Implementation Example (Java):

```java
// 1. Define exception
public class ChannelDisabledException extends DomainException {
  public ChannelDisabledException(String channelId) {
    super(
      "BIZ_CHANNEL_DISABLED",
      "Channel '" + channelId + "' is disabled",
      400,
      Map.of("channelId", channelId)
    );
  }
}

// 2. Throw from service
@Service
public class ActivateChannelService {
  public void execute(String companyId, String channelId) {
    Channel channel = repository.findByIdAndCompanyId(channelId, companyId)
      .orElseThrow(() -> new ChannelNotFoundException(channelId));

    if (!channel.isActive()) {
      throw new ChannelDisabledException(channelId);
    }
  }
}

// 3. Controller (auto-catches via GlobalExceptionHandler)
@PostMapping("channels/{id}/activate")
public ResponseEntity<ActivateResponse> activate(@PathVariable String id) {
  activateService.execute(companyId, id);
  return ResponseEntity.ok(new ActivateResponse());
}
```

---

## Standard Error Response Format

### Success Response:

```json
{
  "statusCode": 200,
  "data": {
    "id": "ch_abc123",
    "name": "My Channel"
  }
}
```

### Error Response:

```json
{
  "errorCode": "BIZ_ENTITY_NOT_FOUND",
  "message": "Channel 'ch_abc123' not found",
  "statusCode": 404,
  "details": {
    "channelId": "ch_abc123",
    "companyId": "comp_xyz789"
  },
  "timestamp": "2026-02-25T10:30:45.123Z"
}
```

### Response Fields:

- `errorCode`: Machine-readable error code (enum)
- `message`: Human-readable description
- `statusCode`: HTTP status code
- `details`: Context for debugging (user ID, entity ID, etc.)
- `timestamp`: ISO 8601 timestamp (when error occurred)

---

## Error Handling by Category

### AUTH\_\* Handling:

```typescript
// Controller or Filter
if (
  error instanceof DomainException &&
  error.getErrorCode().startsWith("AUTH_")
) {
  // Log security event
  logger.warn(`Authentication failed: ${error.message}`, { userId });
  // Return 401/403 with minimal details
  return { errorCode: error.getErrorCode(), message: "Authentication failed" };
}
```

### VAL\_\* Handling:

```typescript
// Show validation details to user
if (
  error instanceof DomainException &&
  error.getErrorCode().startsWith("VAL_")
) {
  // Include field info for form validation
  return {
    errorCode: error.getErrorCode(),
    message: error.message,
    details: error.details, // Include which field failed
  };
}
```

### BIZ\_\* Handling:

```typescript
// Log business rule violations (for analytics)
if (
  error instanceof DomainException &&
  error.getErrorCode().startsWith("BIZ_")
) {
  logger.info(`Business rule violation: ${error.message}`, error.details);
  return {
    errorCode: error.getErrorCode(),
    message: error.message,
    statusCode,
  };
}
```

### INF\_\* Handling:

```typescript
// Alert ops for infrastructure issues
if (
  error instanceof DomainException &&
  error.getErrorCode().startsWith("INF_")
) {
  logger.error(`Infrastructure error: ${error.message}`, error);
  monitoring.alert(`Infrastructure issue: ${error.getErrorCode()}`);
  // Suggest retry to user
  return {
    errorCode: error.getErrorCode(),
    message: "Service temporarily unavailable",
  };
}
```

---

## Monitoring & Alerting Rules

### 🔴 Critical (Alert Immediately):

- `INF_DATABASE_ERROR` - Data persistence critical
- `INF_EXTERNAL_API_ERROR` - May block message delivery
- `SYS_INTERNAL_ERROR` - Unhandled exception
- Any error spike (threshold: 5+ occurrences/minute)

### 🟡 Warning (Flag & Monitor):

- `BIZ_ENTITY_NOT_FOUND` - High volume = access pattern issue
- `AUTH_PERMISSION_DENIED` - Repeated = permission issues
- `BIZ_QUOTA_EXCEEDED` - Capacity planning needed

### 🟢 Info (Log Only):

- `VAL_MISSING_REQUIRED_FIELD` - Normal user input issues
- `BIZ_ENTITY_ALREADY_EXISTS` - Expected conflicts

---

## Error Code Reference

**For complete error code details, see**:

- Full registry: `system/context/error-codes.md`
- Enum definition: `src/shared/exceptions/error-code.enum.ts`
- Base exception: `src/shared/exceptions/domain-exception.ts`
- Filters: `src/shared/filters/domain-exception.filter.ts`
- Specific exceptions: `src/modules/*/domain/exceptions/`

---

## Common Error Scenarios

### Scenario: User tries to access another company's channel

```
Error Code: BIZ_ENTITY_NOT_FOUND (or AUTH_PERMISSION_DENIED)
HTTP: 404 (or 403)
Root Cause: Multi-tenant isolation check
Fix: Add companyId filter to all queries
```

### Scenario: External API timeout

```
Error Code: INF_TIMEOUT (or INF_EXTERNAL_API_ERROR)
HTTP: 504 (or 502)
Root Cause: External service slow
Fix: Increase timeout, check service status, implement circuit breaker
```

### Scenario: Duplicate webhook request

```
Error Code: BIZ_ENTITY_ALREADY_EXISTS
HTTP: 409
Root Cause: Webhook idempotency key not tracked
Fix: Implement idempotency key tracking in message queue
```

```

