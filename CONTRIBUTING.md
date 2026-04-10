# Contributing to Flux

First off, thank you for considering contributing to Flux! It's people like you that make Flux a high-performance and developer-friendly framework.

## 1. Where do I go from here?

If you've noticed a bug or have a feature request, please **open an issue** before submitting a Pull Request. It's best to discuss the architecture or fix with the maintainers before writing a lot of code.

## 2. Setting up your environment

1. Fork the repo and clone it locally.
2. Ensure you have Go `1.21+` installed.
3. Install `air` (optional) for hot-reloading if you are testing example applications during development:
   ```bash
   go install github.com/air-verse/air@latest
   ```

## 3. Making Changes

- Create a new branch from `main` (`git checkout -b feature/my-awesome-feature`).
- Ensure your code follows standard Go formatting (`go fmt ./...`).
- Add comments to any exported functions or struct fields.
- **Do not break the OpenAPI generator.** If you modify routing (`router.go` or `flux.go`), manually verify that the examples still generate valid `/openapi.json` files.

## 4. Testing Your Changes

We use the example applications as our primary end-to-end integration test suite. Before submitting a PR, you MUST ensure all examples build and pass.

Run the test suite script from the root of the project:
```bash
bash RUN_EXAMPLES.sh
```
This script will:
- Compile all code.
- Spin up every example application.
- Fire mock HTTP requests to verify routing, validation, status codes, and panic recovery.
- Fail immediately if any regressions occurred.

## 5. Submitting a Pull Request

Fill out the Pull Request template completely. Ensure exactly what you changed and why is clearly documented. Once merged into `main`, our GitHub Actions will automatically test and generate a new version tag for your release!

## 6. PR Naming & Auto-Releases (Conventional Commits)

This repository strictly follows **Semantic Versioning** (`vMAJOR.MINOR.PATCH`). 
We use an automated GitHub Action that reads your merged Pull Request title to calculate the next module version. 

Please format your PR titles using **Conventional Commits**:
* **`fix: ...`** — Bumps the Patch version (e.g. `v1.0.0` -> `v1.0.1`). Use this for bug fixes.
* **`feat: ...`** — Bumps the Minor version (e.g. `v1.0.1` -> `v1.1.0`). Use this for new features or API additions.
* **`feat!: ...`** or **`BREAKING CHANGE: ...`** — Bumps the Major version (e.g. `v1.1.0` -> `v2.0.0`). Use this ONLY if your change will break existing users' code.

If you don't use a prefix, the system defaults to a safe `patch` bump.
