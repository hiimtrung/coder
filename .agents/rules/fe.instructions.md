---
applyTo: "**/*.{tsx,jsx,ts,js,css,scss}"
---

# Frontend Development Instructions

## Language-to-Skill Mapping

Before working on any frontend task, run `coder skill search` with the relevant skill:

| Technology / Context         | Skill to search                              |
| ---------------------------- | -------------------------------------------- |
| React / Next.js (web)        | `frontend`, `react-best-practices`           |
| React Native (mobile)        | `react-native-skills`                        |
| UI/UX design implementation  | `ui-ux-pro-max`                              |
| Styling / design system      | `web-design-guidelines`                      |
| Component architecture       | `composition-patterns`                       |
| Testing components           | `testing`                                    |
| General architecture         | `architecture`                               |

## Component Architecture

### Single Responsibility

- Each component has one clear responsibility
- Extract sub-components when a component exceeds ~150 lines
- Separate data-fetching components from presentational components

### Composition Patterns

```tsx
// ✅ Composition — flexible, extensible
<Card>
  <Card.Header>Title</Card.Header>
  <Card.Body>Content</Card.Body>
  <Card.Footer>Actions</Card.Footer>
</Card>

// ❌ Prop drilling — rigid, hard to extend
<Card title="Title" body="Content" footer="Actions" />
```

Run `coder skill search "composition-patterns"` for detailed composition guidance.

### Props & Types

- All component props must be fully typed (TypeScript interfaces/types)
- No `any` types in component props or state
- Use `React.FC` or explicit return type annotations
- Export prop types for reuse in parent components

```tsx
interface ButtonProps {
  label: string;
  variant: 'primary' | 'secondary' | 'danger';
  onClick: () => void;
  disabled?: boolean;
  loading?: boolean;
}
```

### State Management

- **Local state**: `useState` for UI-only state (open/closed, hover)
- **Shared state**: React Context or Zustand for cross-component state
- **Server state**: React Query / SWR for async data fetching and caching
- **Form state**: React Hook Form with Zod validation

## UI/UX Standards

### Accessibility (WCAG AA)

- All interactive elements must be keyboard navigable (Tab, Enter, Escape)
- Use semantic HTML: `<button>`, `<nav>`, `<main>`, `<article>`, `<section>`
- All images need `alt` text (empty `alt=""` for decorative images)
- Color contrast ratio ≥ 4.5:1 for normal text, ≥ 3:1 for large text
- ARIA labels for all non-obvious interactive elements
- Focus indicators must be visible (never `outline: none` without replacement)

```tsx
// ✅ Accessible
<button
  aria-label="Close dialog"
  aria-expanded={isOpen}
  onClick={onClose}
>
  <XIcon aria-hidden="true" />
</button>

// ❌ Not accessible
<div onClick={onClose}><XIcon /></div>
```

Run `coder skill search "ui-ux-pro-max"` for comprehensive UX patterns.

### Responsive Design

- Mobile-first: base styles for mobile, `md:` / `lg:` for larger screens
- Use design tokens for spacing, typography, colors — never hardcode values
- Test at 375px (mobile), 768px (tablet), 1280px (desktop) breakpoints
- Touch targets minimum 44×44px on mobile

### Loading & Error States

Every async operation needs all three states:

```tsx
if (isLoading) return <Skeleton />;
if (error) return <ErrorState message={error.message} onRetry={refetch} />;
return <DataComponent data={data} />;
```

- Use skeleton screens (not spinners) for content areas
- Optimistic updates for immediate feedback on mutations
- Error boundaries for unexpected component failures

## TypeScript Strict Typing for Components

### Component Return Types

```tsx
// ✅ Explicit return type
function UserCard({ user }: UserCardProps): React.ReactElement {
  return <div>{user.name}</div>;
}

// ✅ Nullable return
function OptionalBadge({ show }: { show: boolean }): React.ReactElement | null {
  if (!show) return null;
  return <Badge />;
}
```

### Event Handlers

```tsx
// ✅ Properly typed event handlers
const handleChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
  setValue(e.target.value);
};

const handleSubmit = (e: React.FormEvent<HTMLFormElement>): void => {
  e.preventDefault();
  // ...
};
```

### Generic Components

```tsx
// ✅ Generic list component
interface ListProps<T> {
  items: T[];
  renderItem: (item: T) => React.ReactElement;
  keyExtractor: (item: T) => string;
}

function List<T>({ items, renderItem, keyExtractor }: ListProps<T>) {
  return (
    <ul>
      {items.map(item => (
        <li key={keyExtractor(item)}>{renderItem(item)}</li>
      ))}
    </ul>
  );
}
```

## Build Tools & Package Management

### Package Management

- Use `yarn` as the primary package manager (unless project uses `npm`/`pnpm`)
- Commit `yarn.lock` / `package-lock.json` — never `.gitignore` lock files
- Pin exact versions for critical dependencies in `package.json`

### Next.js

- Use App Router (`app/`) for new Next.js 13+ projects
- Server Components by default; `'use client'` only when needed
- Use `next/image` for all images (automatic optimization)
- Use `next/font` for web fonts (eliminates FOUT)

### React Native / Expo

- Use Expo SDK for new projects (managed workflow preferred)
- Metro bundler — keep `metro.config.js` minimal
- Platform-specific code: `Component.ios.tsx` / `Component.android.tsx`

### Vite

- Use Vite for non-Next.js React projects
- Configure path aliases in `vite.config.ts`

## Styling Guidelines

Run `coder skill search "web-design-guidelines"` for project-specific styling standards.

### Design Tokens

- Never hardcode colors, spacing, or font sizes — use design tokens
- Tailwind: use config-defined colors (`text-primary` not `text-blue-600`)
- CSS Modules: import tokens from a shared constants file

### Class Organization (Tailwind)

```tsx
// ✅ Group by concern: layout → spacing → typography → visual → interactive
<div className="flex items-center gap-4 px-6 py-4 text-sm font-medium text-gray-900 bg-white rounded-lg shadow hover:shadow-md transition-shadow">
```

## Testing Components

- Use React Testing Library — test behavior, not implementation
- Query by accessible roles/labels: `getByRole`, `getByLabelText`
- Avoid `getByTestId` unless no semantic alternative exists
- Test user interactions: `userEvent.click`, `userEvent.type`
- Mock API calls with MSW (Mock Service Worker)

```tsx
// ✅ Test user behavior
it('submits form with valid data', async () => {
  render(<LoginForm />);
  await userEvent.type(screen.getByLabelText('Email'), 'user@example.com');
  await userEvent.type(screen.getByLabelText('Password'), 'password123');
  await userEvent.click(screen.getByRole('button', { name: 'Sign in' }));
  expect(mockLogin).toHaveBeenCalledWith({ email: 'user@example.com', password: 'password123' });
});
```
