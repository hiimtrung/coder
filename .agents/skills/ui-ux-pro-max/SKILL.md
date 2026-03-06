---
name: ui-ux-pro-max
description: Comprehensive design guide for web and mobile applications. Contains 67 styles, 96 color palettes, 57 font pairings, 99 UX guidelines, and 25 chart types across 13 technology stacks. Searchable database with priority-based recommendations.
---

# UI/UX Pro Max

UI/UX design intelligence. 50 styles, 21 palettes, 50 font pairings, 20 charts, 9 stacks.

## Prerequisites

Python 3.x is required for the search script.

```bash
python3 --version || python --version
```

---

## How to Use This Skill

When working on UI/UX tasks (design, build, review, improve), follow this workflow:

### Step 1: Analyze Requirements
Identify product type (SaaS, dashboard, etc.), industry, and desired style (minimal, elegant, etc.).

### Step 2: Generate Design System (REQUIRED)
**Always run this first** to get comprehensive recommendations:

```bash
python3 .agents/skills/ui-ux-pro-max/scripts/search.py "<product_type> <industry> <keywords>" --design-system
```

### Step 3: Detailed Searches
Supplement with domain-specific searches:

```bash
# Get UX guidelines
python3 .agents/skills/ui-ux-pro-max/scripts/search.py "accessibility animation" --domain ux

# Get charts for dashboards
python3 .agents/skills/ui-ux-pro-max/scripts/search.py "dashboard analytics" --domain chart
```

### Step 4: Implementation
Use the generated design system and guidelines to build the UI. Default to `html-tailwind` unless specified.

---

## Search Reference

| Domain | Use For |
|--------|---------|
| `product` | Product type recommendations |
| `style` | UI styles, colors, effects |
| `typography` | Font pairings, Google Fonts |
| `ux` | Best practices, anti-patterns |
| `chart` | Chart types, data visualization |

---

## Pre-Delivery Checklist

- [ ] No emojis used as icons (use SVG instead)
- [ ] Hover states provide visual feedback without layout shift
- [ ] All clickable elements have `cursor-pointer`
- [ ] Transitions are smooth (150-300ms)
- [ ] Light/Dark mode contrast is sufficient
- [ ] Responsive at all common breakpoints (375px to 1440px)
