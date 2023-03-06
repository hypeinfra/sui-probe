<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/44648612/223167295-4ae8da21-4a8d-4f17-b518-3263bc6b539d.png">
  <img alt="Sui Probe Logo" src="https://user-images.githubusercontent.com/44648612/219971097-7983f048-7fb9-4200-a0ea-948be481b6b2.png">
</picture>

# Sui Probe project

A health-checker for your Sui Node, checks sync status and more.

Specifically made easy to deploy on any Golang supported platforms, including Plan9 and Illumos.

## Screenshots

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/44648612/223164504-3140b2c8-75ff-4ba0-95c1-e0a727ca243e.png">
    <img alt="Main page" src="https://user-images.githubusercontent.com/44648612/223162002-52e8f16e-dace-4049-8718-bece9048b03f.png" width="45%">
   </picture>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/44648612/223164391-53a7a111-5502-464c-9eaf-742d42e213c6.png">
    <img alt="Node statistics page" src="https://user-images.githubusercontent.com/44648612/223162006-a67d3647-ba94-4a68-a992-8320e71c650f.png" width="45%">
  </picture>
</p>

## Usage

Run the executable and navigate to `http://localhost:1323` to see the main page.

It also works without Javascript.

## Installation

You can install pre-compiled binaries from the releases page or compile this project from the source.

You can also check out [installation instructions] in the wiki.

### Compilation

You need to have Golang, Git installed on your system.

Run it directly from the source code:

```bash
go run github.com/hypeinfra/sui-probe/cmd/suiprobe@latest
```

Remember about `GOPROXY`, it will cache it for some time.
Change it to `direct` if you want to get the latest version.

### Systemd service

You can run it as a systemd service, check the [instructions] in the wiki.

## Hacking

Directory structure:

- `cmd/suiprobe` - main executable, contains the web server
- `sui` - Sui client library (API)
- `templates` - HTML templates
- `static` - static assets (CSS, JS, images)

[installation instructions]: https://github.com/hypeinfra/sui-probe/wiki/Installation
[instructions]: https://github.com/hypeinfra/sui-probe/wiki/Systemd
