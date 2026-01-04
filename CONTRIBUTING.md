# Contributing to aws-go-tools

Thank you for your interest in contributing to this project!

## Development Setup

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/aws-go-tools.git`
3. Install dependencies: `go mod download`
4. Create a feature branch: `git checkout -b feature/your-feature-name`

## Making Changes

1. Make your changes in your feature branch
2. Test your changes locally:
   ```bash
   go build -o ec2-ssm-connector
   ./ec2-ssm-connector --version
   ```
3. Ensure code follows Go conventions: `go fmt ./...`
4. Test thoroughly with different scenarios

## Pull Request Labels for Version Bumping

When creating a pull request, add the appropriate label to indicate the type of change:

### Version Bump Labels

- **Major version bump** (breaking changes): Add `major` or `breaking` label
  - Breaking API changes
  - Removed functionality
  - Incompatible changes

- **Minor version bump** (new features): Add `minor` or `feature` label
  - New features
  - New functionality
  - Backward-compatible additions

- **Patch version bump** (bug fixes): Add `patch` label or no label (default)
  - Bug fixes
  - Documentation updates
  - Minor improvements

### Example PR Workflow

1. Create a pull request with your changes
2. Add appropriate label(s):
   - Breaking change? → Add `breaking` label
   - New feature? → Add `feature` label
   - Bug fix? → No label needed (defaults to patch)
3. Wait for review
4. Once approved and merged, automatic release will be created

## Commit Message Convention

While version bumping is now based on PR labels, we still encourage clear commit messages:

### Commit Message Format

Follow this format for clear, descriptive commits:

```
type: short description

Optional longer description explaining the change in more detail.

- Bullet points for specific changes
- Make it clear what was changed and why
```

**Types:**
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

## Pull Request Process

1. Update the README.md with details of changes if applicable
2. Update CHANGELOG.md with your changes under "Unreleased"
3. Ensure your code builds successfully
4. Submit a pull request to the `main` branch
5. Wait for review and address any feedback

## Testing

Before submitting:

1. Build the binary: `go build -o ec2-ssm-connector`
2. Run tests: `make test` or `go test -v ./...`
3. Check code formatting: `make fmt` or `go fmt ./...`
4. Test EC2 SSM mode: `./ec2-ssm-connector --mode ec2 --profile test-profile`
5. Test RDS mode: `./ec2-ssm-connector --mode rds --profile test-profile`
6. Test version flag: `./ec2-ssm-connector --version`

### Automated PR Checks

When you create a pull request, GitHub Actions will automatically:
- ✅ Check code formatting with `go fmt`
- ✅ Run `go vet` for common mistakes
- ✅ Run `golangci-lint` for comprehensive linting
- ✅ Execute all tests with race detection
- ✅ Validate builds for all platforms (Linux, macOS, Windows - AMD64 & ARM64)
- ✅ Generate test coverage reports

All results will be posted as a comment on your PR.

## Code Style

- Follow standard Go conventions
- Run `go fmt ./...` before committing
- Keep functions focused and single-purpose
- Add comments for complex logic
- Error messages should be clear and actionable

## Questions?

Feel free to open an issue for:
- Bug reports
- Feature requests
- Questions about usage
- Suggestions for improvements
