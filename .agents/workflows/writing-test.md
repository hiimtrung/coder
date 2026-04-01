---
description: Test writing process — analyze feature requirements, write comprehensive unit, integration, and E2E tests with full coverage of happy paths, error paths, and edge cases.
---

# Workflow: Writing Tests

A systematic approach to writing tests for a feature or module. Produces tests that verify behavior, document intent, and protect against future regressions.

## When to Use

- Implementing a new feature (write tests first — TDD)
- Adding tests to untested existing code
- A code review identified missing test coverage
- Preparing for a refactor (tests must exist before changing behavior)

## Step 1 — Context Load (MANDATORY)

```bash
coder skill resolve "testing <language or framework>" --trigger execution --budget 3
coder memory search "<feature or module name>"
```

Use `coder memory recall "<feature or module name>"` when the testing scope is large and you need the active context narrowed to the current module.
Use `coder memory active` or `.coder/context-state.json` to inspect the current local context before writing tests.

## Step 2 — Gather Context

Before writing tests, read:

- `docs/requirements/<feature>.md` — user stories and acceptance criteria
- `docs/design/<feature>.md` — data flows, error codes, edge cases
- Existing test files for the module — understand current patterns and utilities

```bash
git diff --name-only main...HEAD
```

## Step 3 — Map Test Scenarios

For each user story and each edge case in the requirements, map it to a test:

| Story / Scenario             | Test Type   | Test Name Pattern                                        | File            |
| ---------------------------- | ----------- | -------------------------------------------------------- | --------------- |
| Create resource (happy path) | Unit + E2E  | `should create resource with valid data`                 | `*.spec.ts`     |
| Name too long (VAL error)    | Unit        | `should throw VAL_NAME_TOO_LONG when name exceeds limit` | `*.spec.ts`     |
| Resource not found (BIZ)     | Unit        | `should throw BIZ_NOT_FOUND when id does not exist`      | `*.spec.ts`     |
| Unauthorized access          | E2E         | `should return 401 when no JWT provided`                 | `*.e2e-spec.ts` |
| Cross-tenant isolation       | Integration | `should not return resources from other companies`       | `*.spec.ts`     |

Target coverage: 100% of acceptance criteria, all error paths, all boundary conditions.

## Step 4 — Write Unit Tests

Unit tests cover domain logic and use cases with mocked dependencies.

```typescript
// use-case.spec.ts — Unit test pattern
describe("CreateFeatureUseCase", () => {
  let useCase: CreateFeatureUseCase;
  let mockRepo: jest.Mocked<IFeatureRepository>;

  beforeEach(() => {
    mockRepo = {
      save: jest.fn(),
      findById: jest.fn(),
    } as jest.Mocked<IFeatureRepository>;
    useCase = new CreateFeatureUseCase(mockRepo);
  });

  describe("execute", () => {
    it("should create and save a feature with valid input", async () => {
      const command = { name: "Test Feature", companyId: "co-123" };
      mockRepo.save.mockResolvedValue(undefined);

      const result = await useCase.execute(command);

      expect(result.name).toBe("Test Feature");
      expect(mockRepo.save).toHaveBeenCalledOnce();
    });

    it("should throw BIZ_FEATURE_ALREADY_EXISTS when name is taken", async () => {
      mockRepo.findByName = jest.fn().mockResolvedValue(existingFeature);

      await expect(useCase.execute(command)).rejects.toThrow(
        "BIZ_FEATURE_ALREADY_EXISTS",
      );
    });

    it("should include company_id in saved entity", async () => {
      await useCase.execute({ name: "Feature", companyId: "co-456" });

      const savedEntity = mockRepo.save.mock.calls[0][0];
      expect(savedEntity.companyId).toBe("co-456");
    });
  });
});
```

## Step 5 — Write Integration Tests

Integration tests verify repository implementations against a real (or in-memory) database.

```typescript
// repository.spec.ts — Integration test pattern
describe("FeatureRepository", () => {
  let repo: FeatureRepository;
  let testDb: TestDatabase;

  beforeAll(async () => {
    testDb = await TestDatabase.create();
    repo = new FeatureRepository(testDb.connection);
  });

  afterAll(() => testDb.destroy());
  afterEach(() => testDb.clear("features"));

  it("should persist and retrieve a feature", async () => {
    const entity = FeatureEntity.create({ name: "Test", companyId: "co-1" });
    await repo.save(entity);

    const found = await repo.findById(entity.id, "co-1");
    expect(found?.name).toBe("Test");
  });

  it("should not return features from other companies", async () => {
    const entity = FeatureEntity.create({ name: "Private", companyId: "co-1" });
    await repo.save(entity);

    const found = await repo.findById(entity.id, "co-999");
    expect(found).toBeNull();
  });
});
```

## Step 6 — Write E2E Tests

E2E tests verify the full stack from HTTP request to database and back.

```typescript
// feature.e2e-spec.ts — E2E test pattern
describe("Feature API (e2e)", () => {
  let app: INestApplication;
  let authToken: string;

  beforeAll(async () => {
    app = await createTestApp();
    authToken = await loginAsTestUser(app, "co-1");
  });

  afterAll(() => app.close());

  describe("POST /api/v1/features", () => {
    it("should create a feature and return 201", async () => {
      const response = await request(app.getHttpServer())
        .post("/api/v1/features")
        .set("Authorization", `Bearer ${authToken}`)
        .send({ name: "New Feature" });

      expect(response.status).toBe(201);
      expect(response.body.name).toBe("New Feature");
      expect(response.body.id).toBeDefined();
    });

    it("should return 400 with VAL_INVALID_NAME when name is empty", async () => {
      const response = await request(app.getHttpServer())
        .post("/api/v1/features")
        .set("Authorization", `Bearer ${authToken}`)
        .send({ name: "" });

      expect(response.status).toBe(400);
      expect(response.body.error.code).toBe("VAL_INVALID_NAME");
    });

    it("should return 401 when no token provided", async () => {
      const response = await request(app.getHttpServer())
        .post("/api/v1/features")
        .send({ name: "Test" });

      expect(response.status).toBe(401);
    });
  });
});
```

## Step 7 — Run and Verify Coverage

```bash
# Run all tests
yarn test
yarn test:e2e

# Check coverage
yarn test --coverage

# View uncovered lines
open coverage/lcov-report/index.html
```

Identify any uncovered scenarios and add tests until all acceptance criteria are covered.

## Step 8 — Update Documentation

Update `docs/requirements/<feature>.md` to link to the test files:

```markdown
## Test Coverage

| Story   | Test File                         | Test Count |
| ------- | --------------------------------- | ---------- |
| Story 1 | `src/feature/feature.spec.ts`     | 8 tests    |
| Story 2 | `src/feature/feature.e2e-spec.ts` | 4 tests    |
```

## Step 9 — Gate Out (MANDATORY)

```bash
coder memory store "Testing: <Feature Name>" "Test types: unit, integration, E2E. Count: <N unit>, <N integration>, <N e2e>. Coverage: <N>%. Mocking strategy: <what was mocked and how>. Patterns established: <reusable test utilities or patterns>." --tags "testing,<feature>,<language>,<framework>"
```

---

## Checklist

- [ ] `coder skill resolve` run
- [ ] `coder memory search` run
- [ ] Requirements doc read — all user stories and acceptance criteria mapped to tests
- [ ] Test scenarios table written before coding tests
- [ ] Unit tests cover domain logic and use case behavior
- [ ] Integration tests verify repository with real DB layer
- [ ] E2E tests cover happy path and key error paths
- [ ] Multi-tenant isolation tested explicitly
- [ ] All tests pass
- [ ] Coverage verified against acceptance criteria
- [ ] `coder memory store` run with testing patterns
