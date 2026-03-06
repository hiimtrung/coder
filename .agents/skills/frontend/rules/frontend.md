---
name: frontend-standards
description: Modern frontend standards - React/Next.js (RSC), Vue, Astro patterns, SSR/CSR strategies, and design system mandates.
applyTo: "**/*.{ts,tsx,vue,astro,js,jsx}"
---

# Enterprise Frontend Engineering Rules

## 1. Modern Framework Patterns

### React & Next.js (App Router)
- **Component Classification**: Explicitly use `'use client'` ONLY for components requiring hooks, browser APIs, or interactivity. Default to **React Server Components (RSC)**.
- **Data Fetching**: Use Server Components to fetch data directly. Sanitize and validate all server-side inputs.
- **Forms**: Use **Server Actions** with `useActionState` and `useFormStatus` for form handling. Validate schema on both client and server (e.g., Zod).
- **State Management**: Prefer **URL State** (search params) or **React Context** for global UI state. Only use external libraries (Zustand, Redux) for high-complexity state.

### Vue.js
- **Composition API**: Mandated usage of `<script setup>` and the Composition API.
- **Prop Logic**: Strict prop validation. Emits must be explicitly declared for event safety.

### Astro
- **Islands Architecture**: Optimize for zero-JS by default. Only enable hydration (`client:load`, `client:visible`) where strictly necessary.

---

## 2. Rendering & Performance (SSR/CSR)

- **SSR Protocol**: Critical for SEO and FCP. Ensure all common SEO metadata (Title, OpenGraph, Meta) is populated server-side.
- **Hydration Safety**: Prevent hydration mismatches by ensuring client-specific code (e.g., `localStorage`) is wrapped in `useEffect` or `onMounted`.
- **Image Optimization**: Always use framework-specific Image components (e.g., `next/image`, `astro:assets`) with proper aspect ratios and lazy loading.
- **Vitals Enforcement**: Maintain **LCP < 2.5s**, **CLS < 0.1**, and **INP < 200ms**.
- **Layout Shift Prevention**: Always reserve dimensions (aspect-ratio or explicit height) for async content and images to prevent Layout Shift.
- **Motion Accessibility**: Respect user motion preferences by checking `prefers-reduced-motion`. Disable or simplify non-essential animations.

---

## 3. Design System & Component Architecture

- **Atomic Design**: Organize components into Atoms (UI primitives), Molecules (groups of atoms), and Organisms (standalone sections).
- **Base Components**: Build a "headless" or "styled-system" base layer. Components must be **theme-aware** and **accessible**.
- **Touch Targets**: Minimum **44x44px** for all interactive elements and buttons on mobile-friendly views.
- **Z-Index Scale**: Standardize z-index using a consistent scale: `z-10` (backgrounds), `z-20` (interactives), `z-30` (overlays/dropdowns), `z-50` (sticky nav/modals).
- **Prop Standards**: Components should follow a predictable API pattern. Use discriminated unions for variant-based props (e.g., `variant: 'primary' | 'secondary'`).
- **Tailwind CSS (If used)**: Centralize magic numbers in `tailwind.config.ts`. Avoid arbitrary values (`h-[123px]`) in component files.

---

## 4. Security & Hardening

- **XSS Prevention**: Never use `dangerouslySetInnerHTML` or `v-html` without explicit sanitization using a library like `DOMPurify`.
- **CSRF Protection**: Ensure all state-changing requests (POST/PUT/PATCH) use proper CSRF tokens or modern SameSite cookie protections.
- **Dependency Audit**: Regularly check for vulnerabilities in `@npm` packages. Pin critical versions.
- **Environment Safety**: Never expose `process.env` secrets prefix-less (e.g., without `NEXT_PUBLIC_` or `VITE_`) if they are needed on the client.

---

## 5. Violation Checklist
- [ ] Is a client-side hook being used in a file without `'use client'`? (FAIL)
- [ ] Does a component contain arbitrary layout styles instead of using tokens? (FAIL)
- [ ] Is an API secret being exposed to the client bundle? (FAIL)
- [ ] Are hydration-unsafe browser APIs being used outside of lifecycle hooks? (FAIL)
