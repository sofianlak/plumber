<p align="center">
  <img src="assets/plumber.svg" alt="Plumber" width="600">
</p>

<p align="center">
  <b>CI/CD compliance scanner for GitLab pipelines</b>
</p>

<p align="center">
  <a href="https://github.com/getplumber/plumber/actions"><img src="https://img.shields.io/github/actions/workflow/status/getplumber/plumber/release.yml?label=Build" alt="Build Status"></a>
  <a href="https://github.com/getplumber/plumber/releases"><img src="https://img.shields.io/github/v/release/getplumber/plumber" alt="Latest Release"></a>
  <img src="https://img.shields.io/github/go-mod/go-version/getplumber/plumber" alt="Go Version">
  <a href="https://github.com/getplumber/plumber/releases"><img src="https://img.shields.io/github/downloads/getplumber/plumber/total?label=Downloads" alt="GitHub Downloads"></a>
  <a href="https://hub.docker.com/r/getplumber/plumber"><img src="https://img.shields.io/docker/pulls/getplumber/plumber" alt="Docker Pulls"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MPL--2.0-blue" alt="License"></a>
</p>

<p align="center">
  <a href="https://getplumber.io">Website</a> •
  <a href="https://discord.gg/932xkSU24f">Discord</a> •
  <a href="https://github.com/getplumber/plumber/issues">Issues</a>
</p>

---

## 🤔 What is Plumber?

Plumber is a compliance scanner for GitLab. It reads your `.gitlab-ci.yml` and repository settings, then checks for security and compliance issues like:

- Container images using mutable tags (`latest`, `dev`)
- Container images from untrusted registries
- Unprotected branches

**How does it work?** Plumber connects to your GitLab instance via API, analyzes your pipeline configuration, and reports any issues it finds. You define what's allowed in a config file (`.plumber.yaml`), and Plumber tells you if your project complies.

<p align="center">
  <img src="assets/component.gif" alt="Plumber Demo" width="700">
</p>

## 🚀 Two Ways to Use Plumber

Choose **one** of these methods. You don't need both:

| Method | Best for | How it works |
|--------|----------|--------------|
| **[CLI](#option-1-cli)** | Quick evaluation, local testing, one-off scans | Install binary and run from terminal |
| **[GitLab CI Component](#option-2-gitlab-ci-component)** | Automated checks on every pipeline run | Add 2 lines to your `.gitlab-ci.yml` |

---

## 📖 Table of Contents

- [What is Plumber?](#-what-is-plumber)
- [CLI](#option-1-cli)
- [GitLab CI Component](#option-2-gitlab-ci-component)
- [Configuration](#%EF%B8%8F-configuration)
- [Installation](#-installation)
- [CLI Reference](#-cli-reference)
- [Self-Hosted GitLab](#%EF%B8%8F-self-hosted-gitlab)
- [Troubleshooting](#-troubleshooting)


---

## Option 1: CLI

**Try Plumber in 2 minutes!** No commits, no CI changes, just run it.

### Step 1: Install

```bash
# Homebrew (macOS/Linux)
brew tap getplumber/plumber
brew install plumber

# Mise
mise use -g ubi:getplumber/plumber

# Direct download (macOS/Linux)
curl -LO "https://github.com/getplumber/plumber/releases/latest/download/plumber-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')"
chmod +x plumber-* && sudo mv plumber-* /usr/local/bin/plumber
```

> 📦 See [Installation](#-installation) for Windows, Docker, or building from source.

### Step 2: Generate a Config File

```bash
plumber generate config
```

This creates `.plumber.yaml` with [default](./.plumber.yaml) compliance rules. You can customize it later.

### Step 3: Create & Set Your Token

1. In GitLab, go to **User Settings → Access Tokens** ([direct link](https://gitlab.com/-/user_settings/personal_access_tokens))
2. Create a Personal Access Token with `read_api` + `read_repository` scopes
3. Export it in your terminal:

```bash
export GITLAB_TOKEN=glpat-xxxx
```

> 💡 You can also use a project or group access token if you prefer scoped permissions.

### Step 4: Run Analysis

```bash
plumber analyze \
  --gitlab-url https://gitlab.com \
  --project mygroup/myproject
```

Plumber reads your `.plumber.yaml` config and outputs a compliance report. You can also tell it to store the output in JSON format with the `--output` flag.

> 💡 **Like what you see?** Add Plumber to your CI/CD with the [GitLab CI Component](#option-2-gitlab-ci-component) for automated checks on every pipeline.

---

## Option 2: GitLab CI Component

**Add Plumber to your GitLab pipeline**: it will run automatically on the default branch, tags and open merge requests.

> ⚠️ These instructions are for **gitlab.com**. Self-hosted? See [Self-Hosted GitLab](#%EF%B8%8F-self-hosted-gitlab).

### Step 1: Create a GitLab Token

1. In GitLab, go to **User Settings → Access Tokens** ([or create one here](https://gitlab.com/-/user_settings/personal_access_tokens))
2. Create a Personal Access Token with `read_api` + `read_repository` scopes
3. Go to your project's **Settings → CI/CD → Variables**
4. Add the token as `GITLAB_TOKEN` (masked recommended)

> 💡 You can also use a project or group access token if you prefer scoped permissions.

### Step 2: Add to Your Pipeline

Add this to your `.gitlab-ci.yml`:

```yaml
include:
  - component: gitlab.com/getplumber/plumber/plumber@~latest
```

### Step 3: Run Your Pipeline

That's it! Plumber will now run on every pipeline and report compliance issues.

> 💡 **Want to customize?** See [Configuration](#%EF%B8%8F-configuration) to set thresholds, enable/disable controls, and whitelist trusted images.

---

## ⚙️ Configuration

### GitLab CI Component Inputs

Override any input to fit your needs:

```yaml
include:
  - component: gitlab.com/getplumber/plumber/plumber@~latest
    inputs:
      threshold: 80                           # Minimum % to pass (default: 100)
      config_file: configs/my-plumber.yaml    # Custom config path
      server_url: https://gitlab.example.com  # Self-hosted GitLab
      branch: develop                         # Specific branch to analyze
      verbose: true                           # Debug output
```

<details>
<summary><b>All available inputs</b></summary>

| Input | Default | Description |
|-------|---------|-------------|
| `server_url` | `$CI_SERVER_URL` | GitLab instance URL |
| `project_path` | `$CI_PROJECT_PATH` | Project to analyze |
| `branch` | `$CI_COMMIT_REF_NAME` | Branch to analyze |
| `gitlab_token` | `$GITLAB_TOKEN` | GitLab API token (requires `read_api` + `read_repository`) |
| `threshold` | `100` | Minimum compliance % to pass |
| `config_file` | *(auto-detect)* | Path to config file (relative to repo root) |
| `output_file` | `plumber-report.json` | Path to write JSON results |
| `print_output` | `true` | Print text output to stdout |
| `stage` | `.pre` | Pipeline stage for the job |
| `image` | `getplumber/plumber:0.1` | Docker image to use |
| `allow_failure` | `false` | Allow job to fail without blocking |
| `verbose` | `false` | Enable debug output |

</details>

### Configuration File

Generate a default configuration file with:

```bash
plumber generate config

Flags:
  -f, --force           Overwrite existing file
  -o, --output string   Output file path (default ".plumber.yaml")
```

This creates `.plumber.yaml` with sensible [defaults](./.plumber.yaml). Customize it to fit your needs:


---

## 📦 Installation

### Homebrew

```bash
brew tap getplumber/plumber
brew install plumber
```

To install a specific version:

```bash
brew install getplumber/plumber/plumber@0.1.23
```

### Mise

```bash
mise use -g ubi:getplumber/plumber
```

### Binary Download

<details>
<summary><b>Linux (amd64)</b></summary>

```bash
curl -LO https://github.com/getplumber/plumber/releases/latest/download/plumber-linux-amd64
chmod +x plumber-linux-amd64
sudo mv plumber-linux-amd64 /usr/local/bin/plumber
```

</details>

<details>
<summary><b>Linux (arm64)</b></summary>

```bash
curl -LO https://github.com/getplumber/plumber/releases/latest/download/plumber-linux-arm64
chmod +x plumber-linux-arm64
sudo mv plumber-linux-arm64 /usr/local/bin/plumber
```

</details>

<details>
<summary><b>macOS (Apple Silicon)</b></summary>

```bash
curl -LO https://github.com/getplumber/plumber/releases/latest/download/plumber-darwin-arm64
chmod +x plumber-darwin-arm64
sudo mv plumber-darwin-arm64 /usr/local/bin/plumber
```

</details>

<details>
<summary><b>macOS (Intel)</b></summary>

```bash
curl -LO https://github.com/getplumber/plumber/releases/latest/download/plumber-darwin-amd64
chmod +x plumber-darwin-amd64
sudo mv plumber-darwin-amd64 /usr/local/bin/plumber
```

</details>

<details>
<summary><b>Windows (PowerShell)</b></summary>

```powershell
Invoke-WebRequest -Uri https://github.com/getplumber/plumber/releases/latest/download/plumber-windows-amd64.exe -OutFile plumber.exe
```

</details>

<details>
<summary><b>Verify checksum</b></summary>

```bash
curl -LO https://github.com/getplumber/plumber/releases/latest/download/checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

</details>

### Docker

```bash
docker pull getplumber/plumber:latest

docker run --rm \
  -e GITLAB_TOKEN=glpat-xxxx \
  getplumber/plumber:latest analyze \
  --gitlab-url https://your-gitlab-instance.com \ 
  --project mygroup/myproject
```

### Build from Source

> Requires Go 1.24+ and Make.

```bash
git clone https://github.com/getplumber/plumber.git
cd plumber
make build # or make install to build and copy to /usr/local/bin/
```

---

## 🔍 CLI Reference

### `plumber analyze`

Run compliance analysis on a GitLab project.

```bash
plumber analyze [flags]
```

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--gitlab-url` | Yes | — | GitLab instance URL |
| `--project` | Yes | — | Project path (e.g., `group/project`) |
| `--config` | No | `.plumber.yaml` | Path to config file |
| `--threshold` | No | `100` | Minimum compliance % to pass (0-100) |
| `--branch` | No | default | Branch to analyze |
| `--output` | No | — | Write JSON results to file |
| `--print` | No | `true` | Print text output to stdout |

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GITLAB_TOKEN` | Yes | GitLab API token with `read_api` + `read_repository` scopes |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Compliance ≥ threshold |
| `1` | Compliance < threshold or error |

### `plumber generate config`

Generate a default `.plumber.yaml` configuration file.

```bash
plumber generate config [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--output`, `-o` | `.plumber.yaml` | Output file path |
| `--force`, `-f` | `false` | Overwrite existing file |

**Examples:**

```bash
# Generate default config
plumber generate config

# Custom filename
plumber generate config --output my-plumber.yaml

# Overwrite existing file
plumber generate config --force
```

---

## ⚠️ Self-Hosted GitLab

If you're running a self-hosted GitLab instance, you'll need to host your own copy of the component.

<details>
<summary><b>Step-by-step setup</b></summary>

**Step 1: Import the repository**

- Go to **New Project → Import project → Repository by URL**
- URL: `https://gitlab.com/getplumber/plumber.git`
- Choose a group/project name (e.g., `infrastructure/plumber`)

**Step 2: Enable CI/CD Catalog**

- Go to **Settings → General**
- Make sure the project has a **description** (required for CI/CD Catalog)
- Expand **Visibility, project features, permissions**
- Toggle **CI/CD Catalog resource** to enabled
- Click **Save changes**

**Step 3: Create a release**

- Go to **Code → Tags → New tag**
- Enter a version (e.g., `1.0.0`)
- Click **Create tag**

**Step 4: Create a GitLab Token**

In the project you want to scan:

1. Go to **User Settings → Access Tokens** on your GitLab instance
2. Create a Personal Access Token with `read_api` + `read_repository` scopes
3. Go to the project's **Settings → CI/CD → Variables**
4. Add the token as `GITLAB_TOKEN` (masked recommended)

> 💡 You can also use a project or group access token if you prefer scoped permissions.

**Step 5: Use in your pipelines**

```yaml
include:
  - component: gitlab.example.com/infrastructure/plumber/plumber@1.0.0
```

> 💡 Format: `<your-gitlab-host>/<project-path>/plumber@<tag>`

</details>

---

## 🔧 Troubleshooting

| Issue | Solution |
|-------|----------|
| `GITLAB_TOKEN environment variable is required` | Set `GITLAB_TOKEN` in CI/CD Variables or export it locally |
| `401 Unauthorized` | Token needs `read_api` + `read_repository` scopes |
| `403 Forbidden` on MR settings | Expected on non-Premium GitLab; continues without that data |
| `404 Not Found` | Verify project path and GitLab URL are correct |
| Configuration file not found | Use absolute path in Docker, relative path otherwise |

> 💡 **Need help?** [Open an issue](https://github.com/getplumber/plumber/issues) or [join our Discord](https://discord.gg/932xkSU24f)

---

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on how to submit pull requests, report issues, and coding conventions.

## 📄 License

[Mozilla Public License 2.0 (MPL-2.0)](LICENSE)
