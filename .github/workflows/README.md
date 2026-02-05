# CI/CD Workflows

This directory contains GitHub Actions workflows for automated building, testing, and releasing of MainfluxLabs Docker images.

## Release Workflow

The `release.yml` workflow automates the process of building and pushing Docker images to Docker Hub when a new version tag is pushed.

### Trigger

The workflow is triggered when a tag matching the pattern `v*` is pushed:

```bash
git tag v0.0.<version>
git push origin v0.0.<version>
```

### Jobs

#### 1. Lint and Test

Runs on `ubuntu-latest` with Go 1.22:

- Code formatting check (`go fmt`)
- Static analysis (`go vet`)
- Unit tests (`go test`)

#### 2. Build and Push

Runs after successful lint and test:

- Logs into Docker Hub
- Builds Docker images for all services
- Pushes images with both version tag and `latest` tag

### Setup

Before using this workflow, configure the following repository secrets:

1. Go to **Settings** → **Secrets and variables** → **Actions**
2. Add the following secrets:

| Secret | Description |
|--------|-------------|
| `DOCKERHUB_USERNAME` | Your Docker Hub username |
| `DOCKERHUB_TOKEN` | Docker Hub access token with **Read & Write** permissions |

### Creating a Release

```bash
# Create a version tag
git tag v0.0.<version>

# Push the tag to trigger the workflow
git push origin v0.0.<version>
```

### Adding New Architectures

To build for additional architectures, modify the `Makefile` to include the desired platforms in the build configuration.
