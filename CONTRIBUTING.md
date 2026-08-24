# Contributing to mailme

Thank you for your interest in improving `mailme`! Contributions are welcome.

## Getting Started

1. Fork the repository and clone your fork locally.
2. Make sure you have [Go 1.22+](https://go.dev/dl/) installed.
3. Create a branch for your change:
   ```sh
   git checkout -b feature/my-new-feature
   ```

## Making Changes

```sh
go build ./...   # Compile all packages
go vet ./...     # Static analysis & vetting
gofmt -w .       # Format all files
```

Keep commits focused and write clear commit messages. Match the existing code style.

## Submitting a Pull Request

1. Push your branch to your fork.
2. Open a pull request against `main` with a short description of what changed and why.
3. Make sure `go build ./...`, `go vet ./...`, and `gofmt -l .` are clean before submitting.

## Reporting Issues

Open a GitHub issue with steps to reproduce, expected behavior, and actual behavior. Include your OS and `mailme` version where relevant.
