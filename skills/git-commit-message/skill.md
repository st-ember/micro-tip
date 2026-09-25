# Git Commit Message Generator

## Instructions

1. **Strict Context Isolation:** Disregard all previous conversation history and prior prompt turns. Base your analysis **solely** on the staged changes from `git diff --cached`.
2. **Exhaustive Scope Coverage:** Cover all major components, services, adapters, or configurations present in the diff without dropping files.
3. **Determine Type & Scope:** Choose a standard prefix (`feat`, `fix`, `refactor`, `docs`, `chore`, etc.) with an optional scope in parentheses (e.g., `feat(adapters):`).
4. **Format Output:**
   - **Subject Line:** Concise, imperative-present-tense summary capturing the overarching change (under 50 characters).
   - **Body:** Blank line followed by professional, natural descriptive bullet points. Avoid repetitive starting verbs like "Add"; vary your phrasing naturally (e.g., using component-first naming, implementation details, setup, or integration descriptions).
5. **Plain Text Execution:** Output the final commit message as clean plain text. **Do not** wrap the final commit message in markdown code block backticks.

## Example

### Input (`git diff --cached`)

_(Staged changes introducing config, redis, postgres repos, hasher, and server main)_

### Output

feat(adapters): implement driven database, cache, and security adapters

- Configuration loader supporting port formatting and duration parsing
- Redis cache adapter integrated with miniredis unit tests
- Postgres repository adapters, unit of work, and embedded-postgres integration tests
- Bcrypt crypto hasher adapter for secure credential handling
- Server initialization entry point established in cmd/server/main.go
- Initial SQL schema script provided for database tables
