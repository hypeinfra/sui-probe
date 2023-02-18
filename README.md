# Sui Probe project

A health-checker for your Sui Node, checks sync status and more.

Specifically made easy to deploy on any Golang supported platforms, including Plan9 and Illumos.

## Usage

Run the executable and navigate to `http://localhost:1323` to see the main page.

It also works without Javascript.

## Installation

You can install pre-compiled binaries from the releases page or compile this project from the source.

### Compilation

You need to have Golang, Git installed on your system.

Run it directly from the source code:

```bash
go run github.com/hypepartners/cmd/suiprobe@latest
```

Remember about `GOPROXY`, it will cache it for some time.
Change it to `direct` if you want to get the latest version.

## Hacking

Directory structure:

- `cmd/suiprobe` - main executable, contains the web server
- `sui` - Sui client library (API)
- `templates` - HTML templates
- `static` - static assets (CSS, JS, images)