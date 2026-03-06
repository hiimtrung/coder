---
name: c-development-rules
description: Procedural Clean Architecture and safety rules for C development
applyTo: "**/*.{c,h}"
---

# C Development Standards

## 1. Modular Architecture

- **Encapsulation**: Use Opaque Pointers (`struct x;`) in headers to hide implementation details.
- **Header Guards**: Use `#ifndef HEADER_H` or `#pragma once`.
- **Layers**:
  - `domain/`: Structs and "methods" (functions taking struct pointer) representing business logic.
  - `infra/`: Implementations of IO, DB, and network.

## 2. Memory Management

- **Ownership**: Every `malloc` must have a clearly defined owner responsible for `free`.
- **NULL Checks**: Always check the return value of memory allocation functions.
- **Buffer Safety**: NEVER use `gets`, `strcpy`, or `sprintf`. Use `fgets`, `strncpy`, and `snprintf`.

## 3. Error Handling

- Use return codes (`int` or `enum`) for function success/failure.
- Use an `out` parameter for returned data if the return value is an error code.
- Standardize on `ERR_SUCCESS`, `ERR_INVALID_PARAM`, etc.

## 4. Multi-Tenancy

- Pass a `tenant_context_t` struct pointer through function calls or use a thread-safe global context (in multi-threaded environments).

## 5. Violation Checklist

- [ ] Memory leaks (no `free` for a `malloc`)? (FAIL)
- [ ] Global variables without `static` keyword (polluting namespace)? (FAIL)
- [ ] No bounds checking on array/buffer operations? (FAIL)
