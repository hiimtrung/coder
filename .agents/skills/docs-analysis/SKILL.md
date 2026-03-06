---
name: docs-analysis
description: Expert in project documentation analysis and maintenance. Ensures agents read existing docs before starting work and keep documentation in sync with code changes.
---

# Documentation Analysis & Maintenance

This skill ensures that project documentation is treated as a first-class citizen alongside code. It focuses on systematic exploration of the `docs/` directory and proactive maintenance of documentation.

## Core Expertise

- **Context Discovery**: Systematically reading all files in the `docs/` folder to understand business logic, architecture, and constraints.
- **Documentation Sync**: Identifying where documentation needs to be updated following code or requirement changes.
- **Technical Writing**: Maintaining clear, concise, and up-to-date documentation in Markdown format.

## Instructions for the Agent

### 1. Read First
Before any implementation or analysis, list the contents of the `docs/` directory and read relevant files. This provides context that might not be evident from the code alone.

### 2. Trace Impact
For every code change, identify which documentation files are affected:
- API changes → `api-docs.md`, `swagger.json`
- Architecture changes → `architecture.md`, `design-decisions.md`
- Logic/Workflow changes → `business-rules.md`, `workflows.md`
- Data changes → `database-schema.md`

### 3. Continuous Maintenance
Update documentation **within the same iteration** as the code changes. Documentation should NEVER lag behind the implementation.

### 4. Quality Gate
Consider a task "Done" ONLY if all related documentation has been verified and updated.
