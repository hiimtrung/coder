---
name: testing-standards
description: Project-specific testing standards, mocking strategies, and setup patterns
applyTo: "**/*.spec.ts, **/*.test.ts"
---

# Testing Standards & Patterns

## 1. Unit Testing (Jest)

**Importance**: 🔴 CRITICAL

### Pattern: Arrange-Act-Assert (AAA)
Tests must follow the AAA structure for clarity:
```typescript
it("should [expected behavior] when [condition]", async () => {
  // Arrange: Mock dependencies, prepare input
  // Act: Call the method under test
  // Assert: Verify results and side effects
});
```

### Mocking Dependencies
Use `jest.Mocked<T>` for type-safe dependencies:
```typescript
let mockRepository: jest.Mocked<IRepository>;
mockRepository = {
  find: jest.fn().mockResolvedValue(testData),
  save: jest.fn(),
} as any;
```

### Exception Testing
Always verify both error messages and `ErrorCode`:
```typescript
await expect(useCase.execute(dto)).rejects.toThrow(
  expect.objectContaining({
    errorCode: ErrorCode.VAL_MISSING_REQUIRED_FIELD,
  }),
);
```

---

## 2. Integration & E2E Testing (NestJS)

**Importance**: 🟢 HIGH

### Test Database Lifecycle
- Use a dedicated test database (e.g., `_test` suffix).
- Use `synchronize: true` and `dropSchema: true` in `beforeAll` for an isolated environment.
- Clear tables in `beforeEach` to ensure test independence.

### TestingModule Configuration
```typescript
const moduleFixture: TestingModule = await Test.createTestingModule({
  imports: [AppModule],
}).compile();
app = moduleFixture.createNestApplication();
await app.init();
```

---

## 3. Test Data Strategy

### Factories
Centralize object creation in `test/fixtures/` using the Factory pattern:
```typescript
export class EntityFactory {
  static create(overrides?: Partial<Entity>): Entity {
    return Object.assign(new Entity(), { ...fakerData, ...overrides });
  }
}
```

### Faking Data
Use `@faker-js/faker` for generating realistic, randomized test data to prevent pattern-specific bugs.

---

## 4. Quality Mandates

- **Coverage**: Aim for 100% on Use Cases, Repositories, and Event Handlers.
- **Isolation**: Unit tests MUST NOT touch the database or network.
- **Naming**: Test names must be descriptive and documentation-like.
- **Cleanup**: Always close app/database connections in `afterAll`.
