# Git Workflow & Commit Conventions

Commit and branching standards for the `gocore` repository.

---

## 1. Commit Message Convention (Conventional Commits)

Commit messages must follow this structure:
```text
<type>(<scope>): <short description>
```

### Accepted `type` values:
* `feat`: Introduce a new package or feature (e.g., `feat(stderr): implement i18n error formatting`).
* `fix`: Bug fix (e.g., `fix(cache): resolve redis reconnection pool leak`).
* `refactor`: Code change that neither fixes a bug nor adds a feature.
* `test`: Add or update unit tests / benchmarks.
* `docs`: Documentation updates, README, or OpenSpecs.
* `perf`: Performance improvements, memory allocation reduction.
* `chore`: Maintain dependencies (`go.mod`), Makefile, CI/CD, or tooling.

### Guidelines for `short description`:
* Use the imperative mood (e.g., `add`, `fix`, `implement` instead of `added`, `fixes`).
* Keep the first letter lowercase, no trailing period.
* Use English across all commit messages.

---

## 2. Branching & Pull Requests

* **`main`**: Production-ready, always passes all tests and linting.
* **`feature/<name>`** or **`pkg/<name>`**: Package or feature development branch (e.g., `feature/stderr-foundation`).
* **`fix/<issue>`**: Bug fix branch.
* PR Requirements before merge:
  1. `make test` passes 100% with `-race`.
  2. `make lint` reports zero issues.
  3. No breaking changes to established public APIs of stable packages.
