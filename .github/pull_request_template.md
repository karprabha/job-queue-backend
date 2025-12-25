## Task Context

This PR implements the following task:

- Goal:
- Scope:
- Non-goals:

## Constraints (Must Not Be Violated)

- Standard library only (no frameworks)
- No global variables
- No panics in server code
- Explicit error handling
- Context must be respected
- No business logic in main.go

## Self-Review Checklist

- [ ] Code builds with `go build ./...`
- [ ] No ignored errors
- [ ] Handlers are small and focused
- [ ] Shutdown logic uses context
- [ ] Naming follows Go conventions

## Areas I’m Unsure About

-

## Reviewer Notes

Copilot / reviewers:

- Please review against the constraints above
- Focus on idiomatic Go and design, not feature completeness
- Do not rewrite code; comment only
