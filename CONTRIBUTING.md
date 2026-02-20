# Contributing to Stocks Notifier

Thanks for contributing.

## Local setup

1. Clone the repository.
2. Copy `stocks.sample.json` to `stocks.json`.
3. Run the app:
   - CLI: `go run . .`
   - Local UI: `go run . . --web`
4. Run tests: `go test ./...`

## Pull request guidelines

1. Keep PRs focused and small.
2. Include what changed and why.
3. Include validation steps (`go test ./...`, build, or manual checks).
4. Update docs when behavior changes.

## Good first contributions

- Improve docs clarity and examples.
- Add tests for edge cases and error paths.
- Improve UI polish while keeping it lightweight.
- Improve reliability around provider failures and retries.

## Code style

- Follow existing Go style in this repo.
- Prefer clear names over short names.
- Avoid unrelated refactors in feature/fix PRs.
