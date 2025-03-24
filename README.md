# Proxmox Logger

A lightweight Go application that captures Proxmox pvedaemon logs and forwards them to RabbitMQ.

## Overview

This application listens for Proxmox logs sent via rsyslog over TCP, filters for specific pvedaemon task messages, and forwards them to a RabbitMQ message queue for further processing.

## Features

- TCP listener for rsyslog messages
- Pattern matching for Proxmox pvedaemon task logs (UPID messages)
- RabbitMQ integration for message queue publishing
- Lightweight and efficient
- Automated Debian package building with GitHub Actions

## Requirements

- Go 1.18+ (for building from source)
- A RabbitMQ server
- Proxmox VE server with rsyslog

## Installation

### Using apt (Debian/Ubuntu)

```bash
# Simplified installation with trusted repository
echo "deb [trusted=yes] https://evrimuysal.github.io/Proxmox-Logger/deb stable main" | sudo tee /etc/apt/sources.list.d/proxmox-logger.list
sudo apt update
sudo apt install proxmox-logger
```

If you prefer a more secure approach with GPG verification:

```bash
# Add GPG key
curl -fsSL https://evrimuysal.github.io/Proxmox-Logger/gpg-key.asc | sudo gpg --dearmor -o /usr/share/keyrings/proxmox-logger-archive-keyring.gpg

# Add repository
echo "deb [signed-by=/usr/share/keyrings/proxmox-logger-archive-keyring.gpg] https://evrimuysal.github.io/Proxmox-Logger/deb/ stable main" | sudo tee /etc/apt/sources.list.d/proxmox-logger.list

# Update package list
sudo apt update

# Install package
sudo apt install proxmox-logger
```

### Manual Installation

If you encounter any issues with the APT repository, you can manually install the package:

```bash
# Download the latest release
wget https://github.com/evrimuysal/Proxmox-Logger/releases/latest/download/proxmox-logger_*_amd64.deb

# Install the package
sudo dpkg -i proxmox-logger_*_amd64.deb

# Install dependencies (if needed)
sudo apt --fix-broken install
```

### Building from Source

1. Clone this repository:
```bash
git clone https://github.com/evrimuysal/Proxmox-Logger.git
cd proxmox-logger
```

2. Install dependencies:
```bash
go get github.com/streadway/amqp
```

3. Build the application:
```bash
go build -o proxmox-logger
```

4. Install as a systemd service (Debian/Ubuntu):
```bash
# Create directory for the application
sudo mkdir -p /opt/proxmox-logger

# Copy the binary and service file
sudo cp proxmox-logger /opt/proxmox-logger/
sudo cp proxmox-logger.service /etc/systemd/system/

# Reload systemd, enable and start the service
sudo systemctl daemon-reload
sudo systemctl enable proxmox-logger.service
sudo systemctl start proxmox-logger.service
```

### Building Debian Package Locally

There are two ways to build the Debian package locally:

#### Method 1: Using the build script

1. Install build dependencies:
```bash
sudo apt install golang debhelper
```

2. Run the build script:
```bash
./build/scripts/build-deb.sh
```

This will create a `.deb` package in the current directory.

#### Method 2: Using dpkg-buildpackage

1. Install build dependencies:
```bash
sudo apt install devscripts debhelper golang-go
```

2. Build the package:
```bash
dpkg-buildpackage -us -uc
```

The package will be created in the parent directory.

### Automated Package Building

This project uses GitHub Actions to automatically build Debian packages:

1. When a new tag is pushed (format: `v*`), a Debian package is built
2. The package is published to the APT repository on GitHub Pages
3. The package is attached to GitHub Releases (when a release is created)

## Releasing New Versions

### Automatic Release Creation

You can automatically create a new release with the following steps:

1. Go to the [Actions tab](https://github.com/evrimuysal/Proxmox-Logger/actions) in the GitHub repository
2. Select the "Create GitHub Release" workflow
3. Click "Run workflow"
4. Enter the version number (e.g., "1.0.0"), release title, and optional release notes
5. Click "Run workflow"

The workflow will:
- Create a new tag with the specified version number
- Create a new GitHub release
- Trigger the build-deb workflow to build and publish the Debian package

### Manual Release Creation

To manually release a new version:

1. Update version in `debian/changelog` and `build/debian/control`
2. Create a new tag: `git tag v1.0.0`
3. Push the tag: `git push --tags`
4. GitHub Actions will automatically build and publish the package

## Configuration

You can configure the application by modifying the constants at the top of `main.go`:

```go
const (
    RSYSLOG_PROTOCOL = "tcp"
    RSYSLOG_PORT     = "18006"
    RABBITMQ_URI     = "amqp://guest:guest@192.168.1.5:5672/"
)
```

- `RSYSLOG_PROTOCOL`: The protocol to use for the rsyslog listener (TCP or UDP)
- `RSYSLOG_PORT`: The port to listen on for rsyslog messages
- `RABBITMQ_URI`: The URI of your RabbitMQ server

## Configuring rsyslog on Proxmox

Edit your rsyslog configuration on the Proxmox server to forward pvedaemon logs to this application:

1. Create or edit a configuration file in `/etc/rsyslog.d/`:
```bash
sudo nano /etc/rsyslog.d/99-proxmox-logger.conf
```

2. Add the following line to forward pvedaemon logs:
```
if $programname == 'pvedaemon' then @@your_logger_ip:18006
```

3. Restart rsyslog:
```bash
sudo systemctl restart rsyslog
```

## Usage

### Running as a systemd service

```bash
# Start the service
sudo systemctl start proxmox-logger

# Check status
sudo systemctl status proxmox-logger

# Stop the service
sudo systemctl stop proxmox-logger

# View logs
sudo journalctl -u proxmox-logger
```

### Running manually

Run the application:

```bash
./proxmox-logger
```

The application will start listening for rsyslog messages on the configured port and forward matching logs to RabbitMQ.

## Log Format

The application captures logs in the following format:

```
<30>Mar 24 12:04:10 Axoft pvedaemon[669766]: <root@pam> end task UPID:Axoft:0017ABAC:026FA8FF:67E12009:qmstart:103:root@pam: OK
```

## License

[MIT License](LICENSE) 