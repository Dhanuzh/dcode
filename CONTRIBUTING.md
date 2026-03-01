# Contributing

Thanks for wanting to contribute! This project follows a simple workflow:

- Open an issue for discussion before implementing large features.
- Fork the repository and create a feature branch.
- Run tests and linters before submitting a PR.

Local development

- Go tools: go 1.21+, go test ./..., go vet ./..., go fmt ./...
- Node SDK: cd sdk && npm ci

Code style

- Follow gofmt formatting. Use golangci-lint to catch common issues.

Running CI

- The repository includes a GitHub Actions workflow (.github/workflows/ci.yml) that runs formatting checks, vet, tests, and installs the SDK.
