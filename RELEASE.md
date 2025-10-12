# Release Process

This document describes the CI/CD workflow for building, testing, and releasing `cec-controller`.

## Overview

The project uses GitHub Actions workflows to automate building, testing, and releasing binaries for multiple platforms:
- **Ubuntu** (amd64, arm64) - `.deb` packages
- **Fedora** (amd64, arm64) - `.rpm` packages

## Workflows

### 1. Build and Test Workflow (`build.yml`)

**Trigger:** Push to `main` branch or pull request to `main`

**Purpose:** Continuous integration for testing and building on every push

**Steps:**
1. Run tests on Ubuntu with libcec installed
2. Build Ubuntu amd64 and arm64 binaries (using cross-compilation for arm64)
3. Upload build artifacts

### 2. Release Workflow (`release.yml`)

**Triggers:** 
- Automatic: Push of a semver-compliant git tag (e.g., `v1.0.0`, `v2.1.3`)
- Manual: Workflow dispatch with optional dry-run mode

**Purpose:** Full build, test, and release pipeline

**Jobs:**

#### Job 1: Test
- Runs all Go tests on Ubuntu with libcec dependencies installed
- Ensures code quality before building release artifacts

#### Job 2: Build Ubuntu Binaries
- Builds for Ubuntu amd64 and arm64
- Uses native Ubuntu runner with libcec-dev installed
- Cross-compiles arm64 using `aarch64-linux-gnu-gcc`
- Creates binaries in goreleaser-compatible directory structure

#### Job 3: Build Fedora Binaries
- Builds for Fedora amd64 and arm64 inside Docker containers
- Uses official `fedora:latest` image
- Installs `libcec-devel` from Fedora repos (ensures correct library version linking)
- Cross-compiles arm64 using `gcc-aarch64-linux-gnu`
- Creates binaries in goreleaser-compatible directory structure

#### Job 4: Release
- Downloads all built binaries from previous jobs
- Uses GoReleaser to:
  - Create `.deb` packages for Ubuntu (amd64 and arm64)
  - Create `.rpm` packages for Fedora (amd64 and arm64)
  - Generate GitHub Release with all artifacts
  - Generate release notes

## Release Artifacts

Each release includes:
- 4 binary executables:
  - `cec-controller-ubuntu-amd64`
  - `cec-controller-ubuntu-arm64`
  - `cec-controller-fedora-amd64`
  - `cec-controller-fedora-arm64`
- 2 `.deb` packages:
  - `cec-controller_<version>_ubuntu_amd64.deb`
  - `cec-controller_<version>_ubuntu_arm64.deb`
- 2 `.rpm` packages:
  - `cec-controller-<version>-1.<dist>.x86_64.rpm`
  - `cec-controller-<version>-1.<dist>.aarch64.rpm`

## Creating a Release

### Automatic Release (Production)

To create a new release:

1. Ensure all changes are committed and pushed to `main`
2. Create and push a semver-compliant tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
3. GitHub Actions will automatically:
   - Run all tests
   - Build binaries for all platforms
   - Create packages
   - Publish a GitHub Release

### Manual Dry Run (Testing)

To test the release process without publishing:

1. Go to the Actions tab in GitHub
2. Select "Build, Test and Release" workflow
3. Click "Run workflow"
4. Keep "Dry run mode" checked (default: true)
5. Select the branch you want to test
6. Click "Run workflow"

The workflow will:
- Run all tests
- Build binaries for all platforms
- Create packages
- Upload packages as workflow artifacts (not published as a release)

To perform an actual release manually:
- Uncheck the "Dry run mode" option when running the workflow

## Technical Details

### Cross-Compilation for ARM64

- **Ubuntu builds:** Use `aarch64-linux-gnu-gcc` cross-compiler with `PKG_CONFIG_PATH` set to find arm64 libcec libraries
- **Fedora builds:** Use `gcc-aarch64-linux-gnu` inside Fedora Docker container

### Docker-Based Fedora Builds

Fedora binaries are built inside Docker containers to ensure they link against the correct version of `libcec` from Fedora repositories. This prevents compatibility issues when running on Fedora systems.

The Docker commands:
- Install Fedora's `libcec-devel` package
- Build using CGO with the correct compiler
- Output binaries to the host's `dist/` directory

### GoReleaser Integration

The `.goreleaser.yml` configuration defines:
- Build targets for all platforms
- Package metadata for `.deb` and `.rpm` files
- Changelog generation rules

The release workflow uses `goreleaser release --skip=build,validate` to:
- Skip building (already done in separate jobs)
- Use pre-built binaries from the `dist/` directory
- Generate packages using `nfpm`
- Create GitHub Release

## Troubleshooting

### Build Failures

If builds fail:
1. Check the GitHub Actions logs for the specific job that failed
2. Verify that dependencies (libcec) are available in the build environment
3. For Fedora builds, ensure Docker is working and can pull the `fedora:latest` image

### Package Issues

If packages are malformed:
1. Check the `.goreleaser.yml` configuration
2. Verify the directory structure in `dist/` matches goreleaser's expectations
3. Review the goreleaser logs in the release job

### Cross-Compilation Issues

If arm64 builds fail:
1. Ensure cross-compiler packages are installed (`gcc-aarch64-linux-gnu`)
2. Verify `PKG_CONFIG_PATH` is set correctly for Ubuntu builds
3. Check that libcec headers are available for the target architecture

## Development Testing

To test the build process locally:

### Ubuntu Build
```bash
# Install dependencies
sudo apt-get install libcec-dev libp8-platform-dev gcc-aarch64-linux-gnu

# Build amd64
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o cec-controller .

# Build arm64 (cross-compile)
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
  PKG_CONFIG_PATH=/usr/lib/aarch64-linux-gnu/pkgconfig \
  go build -ldflags="-s -w" -o cec-controller-arm64 .
```

### Fedora Build (Docker)
```bash
# Build amd64
docker run --rm -v $(pwd):/workspace -w /workspace fedora:latest bash -c "
  dnf install -y golang libcec-devel gcc && \
  CGO_ENABLED=1 go build -o cec-controller-fedora .
"

# Build arm64 (cross-compile in Docker)
docker run --rm -v $(pwd):/workspace -w /workspace fedora:latest bash -c "
  dnf install -y golang libcec-devel gcc-aarch64-linux-gnu && \
  CGO_ENABLED=1 GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -o cec-controller-fedora-arm64 .
"
```

### Test GoReleaser Config
```bash
# Validate configuration
curl -sfL https://goreleaser.com/static/run | bash -s -- check

# Test release (without publishing)
goreleaser release --snapshot --clean
```
