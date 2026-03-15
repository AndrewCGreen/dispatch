# Contributing

We welcome contributions of all kinds — bug reports, documentation improvements, new backends, SDK improvements, and core features.

---

## Code of Conduct

Be excellent to each other. We follow the [Contributor Covenant](https://www.contributor-covenant.org/).

---

## Ways to Contribute

### Report Bugs

Open an issue on GitHub with:
- Dispatch version (`dispatch version`)
- Operating system and architecture
- Steps to reproduce
- Expected behavior vs actual behavior
- Relevant log output (redact any secrets)

### Suggest Features

Open a GitHub Discussion or Issue describing:
- The problem you're trying to solve
- Your proposed solution
- Alternatives you considered

### Improve Documentation

Documentation PRs are always welcome. The docs live in `docs/`. Fix typos, improve clarity, add examples.

### Submit Code

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Add tests for new functionality
5. Run the test suite: `go test ./...`
6. Submit a pull request

---

## Priority Contributions (Roadmap Help Wanted)

These are high-value areas where contributions are most welcome:

| Area | Description | Skill Level |
|------|-------------|-------------|
| SES backend | Amazon SES implementation | Intermediate |
| Listmonk backend | Listmonk API integration | Intermediate |
| Python SDK | Full Python client library | Intermediate |
| Go SDK | Full Go client library | Intermediate |
| Webhook delivery | Implement webhook firing | Intermediate |
| Double opt-in flow | Confirmation email + token verification | Intermediate |
| Admin web UI | Optional web dashboard | Advanced |
| Prometheus metrics | `/metrics` endpoint | Beginner |
| `dispatch doctor` | Diagnostic checks | Beginner |
| Test coverage | More unit/integration tests | Beginner |

---

## Pull Request Guidelines

- **One change per PR** — keep PRs focused
- **Tests required** for new features and bug fixes
- **Docs required** for new features and API changes
- **Backwards compatible** — don't break existing API or config without a major version bump
- **Clean commits** — meaningful commit messages (`feat: add SES backend`, `fix: handle nil suppression check`, `docs: add webhook signature example`)

### Commit Message Format

```
type(scope): short description

Longer explanation if needed. Wrap at 72 characters.

Fixes #123
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `perf`

---

## License

By contributing, you agree that your contributions will be licensed under the project's license (TBD — likely AGPL-3.0 or MIT).
