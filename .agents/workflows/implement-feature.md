---
description: Developer process — TDD wave-by-wave implementation following confirmed requirements and design documents.
---

# Workflow: Implement Feature

This workflow governs all code implementation. It requires a confirmed requirements doc and design doc before the first line of code is written.

## When to Use

- Requirements confirmed: `docs/requirements/<feature>.md` exists and is approved
- Design confirmed: `docs/design/<feature>.md` exists and is approved
- Implementing a new feature, module, or significant change

## Step 1 — Context Load (MANDATORY)

```bash
coder skill resolve "<language> <feature domain>" --trigger initial --budget 3
coder memory search "<feature name>"
```

Use `coder memory recall "<feature name>"` when the implementation wave needs a narrower active memory working set.
Use `coder memory active` or `.coder/context-state.json` to inspect the current local context before resuming the next wave.

Then read:

- `docs/requirements/<feature>.md` — what to build and acceptance criteria
- `docs/design/<feature>.md` — how to build it
- Existing source files for the target module — understand current patterns

Use `coder skill resolve ... --format raw` if the current wave needs markdown-preserving skill context in the LLM prompt.

## Step 2 — Plan Implementation Waves

Decompose the implementation into independent, committable waves. Each wave must:

- Compile and pass all tests
- Be independently revertable
- Represent one logical slice of functionality

Before starting each wave or spawning a worker for a subtask, re-run:

```bash
coder skill resolve "<wave or subtask>" --trigger execution --budget 3
```

If a worker/subagent owns that wave, the worker must update the corresponding `.coder/` task or checkpoint before returning control.

**Example wave breakdown for a CRUD feature:**

```
Wave 1: Domain layer
  - Entity, value objects, domain exceptions
  - Unit tests for entity behavior

Wave 2: Repository interface + infrastructure
  - Repository interface in domain layer
  - Repository implementation (DB adapter)
  - Integration test for repository

Wave 3: Use cases
  - Create/Update/Delete use cases
  - Unit tests with mocked repository

Wave 4: Controller + DTOs
  - Request/response DTOs with validation
  - Controller with route registration
  - E2E test for happy path

Wave 5: Error paths + edge cases
  - Tests for all error scenarios from requirements
  - Verify all acceptance criteria pass
```

Present the wave plan to the user and wait for confirmation before executing.

## Step 3 — Execute Waves (One at a Time)

For each wave, follow this sequence strictly:

### 3a. Write tests first (TDD — Red phase)

Write the test(s) that describe the expected behavior. They must fail at this point.

```typescript
// Example: entity unit test (Wave 1)
describe("FeatureEntity", () => {
  it("should create with valid data", () => {
    const entity = FeatureEntity.create({ name: "test", companyId: "co-1" });
    expect(entity.name).toBe("test");
    expect(entity.companyId).toBe("co-1");
  });

  it("should throw VAL_INVALID_NAME when name is empty", () => {
    expect(() => FeatureEntity.create({ name: "", companyId: "co-1" })).toThrow(
      "VAL_INVALID_NAME",
    );
  });
});
```

### 3b. Implement (Green phase)

Write the minimum code to make the tests pass. Follow Clean Architecture strictly:

**Domain Layer** (no framework imports):

```typescript
// entities/feature.entity.ts
export class FeatureEntity {
  private constructor(
    public readonly id: string,
    public readonly name: string,
    public readonly companyId: string,
    public readonly createdAt: Date,
  ) {}

  static create(props: CreateFeatureProps): FeatureEntity {
    if (!props.name || props.name.trim().length === 0) {
      throw new DomainException("VAL_INVALID_NAME", "Name cannot be empty");
    }
    return new FeatureEntity(
      generateId(),
      props.name.trim(),
      props.companyId,
      new Date(),
    );
  }
}
```

**Application Layer** (use case, no DB imports):

```typescript
// use-cases/create-feature.use-case.ts
export class CreateFeatureUseCase {
  constructor(private readonly repo: IFeatureRepository) {}

  async execute(cmd: CreateFeatureCommand): Promise<FeatureResponseDto> {
    const entity = FeatureEntity.create(cmd);
    await this.repo.save(entity);
    // Publish event AFTER save
    await this.events.publish(new FeatureCreatedEvent(entity.id));
    return FeatureResponseDto.fromEntity(entity);
  }
}
```

**Infrastructure Layer** (implements domain interfaces):

```typescript
// repositories/feature.repository.ts — implements IFeatureRepository
```

### 3c. Run quality gates

```bash
# Lint
yarn lint
# or: go vet ./... | gradle checkstyleMain

# Build
yarn build
# or: go build ./... | gradle build

# Test
yarn test --testPathPattern="feature"
# or: go test ./... | gradle test
```

All must pass before committing.

### 3d. Commit the wave

```bash
git add <specific files — never git add -A>
git commit -m "feat(<scope>): <wave description>

- <what was added>
- <tests added>
- Passes: <test count> tests"
```

### 3e. Signal checkpoint

After each wave commit:

```
Wave N complete. Committed: <commit hash>

Tests passing: <N>
Next wave: <description>

Type "continue" to proceed to Wave N+1.
```

## Step 4 — Verify All Acceptance Criteria

After the final wave, run the full test suite and verify each acceptance criterion from the requirements doc:

```
Acceptance Criteria Verification:
[ ] Given <context> When <action> Then <outcome> — PASS/FAIL
[ ] Given <context> When <action> Then <outcome> — PASS/FAIL
```

All criteria must pass before declaring the feature complete.

## Step 5 — Gate Out (MANDATORY)

```bash
coder memory store "Implementation: <Feature Name>" "Waves: <N waves>. Patterns used: <patterns>. Non-obvious decisions: <decisions>. Test count: <N>. Files modified: <list>." --tags "implementation,<feature>,<language>,<domain>"
```

---

## Clean Architecture Rules (Non-Negotiable)

```
Controller  → only DTOs, calls use case
Use Case    → only domain interfaces, no DB imports
Entity      → no framework imports, pure business logic
Repository  → implements domain interface, all DB code here
```

Error escalation:

- DTO validation → `VAL_*` (400)
- Use case business rule → `BIZ_*` (400/404/409)
- Repository failure → `INF_*` (500/502/503)
- System/config → `SYS_*` (500)

Multi-tenancy (non-negotiable):

- `company_id` comes from JWT, never from request body
- Every repository query includes `company_id` filter
- Validated in use case before any DB operation

## Checklist

- [ ] `coder skill resolve` run
- [ ] `coder skill resolve "<wave or subtask>" --trigger execution --budget 3` run when scope narrows or changes
- [ ] `coder memory search` run
- [ ] Requirements doc read
- [ ] Design doc read
- [ ] Wave plan confirmed with user
- [ ] Each wave: tests written first (Red)
- [ ] Each wave: implementation done (Green)
- [ ] Each wave: lint + build + test all pass
- [ ] Each wave: committed with clear message
- [ ] All acceptance criteria verified
- [ ] `coder memory store` run
