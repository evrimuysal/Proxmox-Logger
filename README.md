# Proxmox Logger

A lightweight Go application that captures Proxmox pvedaemon logs and forwards them to RabbitMQ.

## Overview

This application listens for Proxmox logs sent via rsyslog over TCP, filters for specific pvedaemon task messages, and forwards them to a RabbitMQ message queue for further processing.

## Features

- TCP listener for rsyslog messages
- Pattern matching for Proxmox pvedaemon task logs (UPID messages)
- RabbitMQ integration for message queue publishing
- YAML configuration file support
- Environment variable configuration
- Automatic log file creation
- RabbitMQ exchange and routing key support
- Integrated rsyslog restart on service restart
- Lightweight and efficient
- Automated Debian package building with GitHub Actions
- Fully integrated systemd service management

## Requirements

- Go 1.20+ (for building from source)
- A RabbitMQ server
- Proxmox VE server with rsyslog

## Installation

### Using apt (Debian/Ubuntu)

```bash
# Simplified installation with trusted repository
echo "deb [trusted=yes] https://evrimuysal.github.io/Proxmox-Logger/deb stable main" | sudo tee /etc/apt/sources.list.d/proxmox-logger.list
sudo apt update
sudo apt install proxmox-logger -y
```

If you encounter issues with `apt update` not finding the package, try these troubleshooting steps:

```bash
# Verify repository configuration
cat /etc/apt/sources.list.d/proxmox-logger.list

# Force update repository metadata
sudo apt update -o Acquire::AllowInsecureRepositories=true -o Acquire::AllowDowngradeToInsecureRepositories=true

# Install the package with verbose output
sudo apt install -V proxmox-logger

# If it's still not working, try downloading and installing directly
wget https://evrimuysal.github.io/Proxmox-Logger/deb/pool/main/proxmox-logger_1.2.0_amd64.deb
sudo dpkg -i proxmox-logger_1.2.0_amd64.deb
sudo apt --fix-broken install
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
# Copy the binary to the standard location
sudo cp proxmox-logger /usr/local/bin/
sudo chmod +x /usr/local/bin/proxmox-logger

# Create and install the systemd service file
cat << EOF | sudo tee /etc/systemd/system/proxmox-logger.service
[Unit]
Description=Proxmox Logger Service
After=network.target

[Service]
ExecStart=/usr/local/bin/proxmox-logger
Restart=on-failure
User=root
Group=root
ExecReload=/bin/kill -HUP \$MAINPID
ExecStop=/bin/kill -TERM \$MAINPID

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd, enable and start the service
sudo systemctl daemon-reload
sudo systemctl enable proxmox-logger
sudo systemctl start proxmox-logger
```

### Building Debian Package Locally

You can build the Debian package manually with these steps:

1. Install build dependencies:
```bash
sudo apt install golang dpkg
```

2. Create the package structure:
```bash
# Create package directories
mkdir -p pkg/DEBIAN
mkdir -p pkg/usr/local/bin
mkdir -p pkg/etc/systemd/system

# Build and copy the binary
go build -o pkg/usr/local/bin/proxmox-logger
chmod +x pkg/usr/local/bin/proxmox-logger

# Create the control file
cat > pkg/DEBIAN/control << EOF
Package: proxmox-logger
Version: 1.0.0
Section: base
Priority: optional
Architecture: amd64
Maintainer: Your Name <your.email@example.com>
Description: Logger for Proxmox, sends events to RabbitMQ.
EOF

# Create systemd service file
cat > pkg/etc/systemd/system/proxmox-logger.service << EOF
[Unit]
Description=Proxmox Logger Service
After=network.target

[Service]
ExecStart=/usr/local/bin/proxmox-logger
Restart=on-failure
User=root
Group=root
ExecReload=/bin/kill -HUP \$MAINPID
ExecStop=/bin/kill -TERM \$MAINPID

[Install]
WantedBy=multi-user.target
EOF

# Add lifecycle management scripts
cat > pkg/DEBIAN/postinst << EOF
#!/bin/bash
set -e
systemctl daemon-reload
systemctl enable proxmox-logger
systemctl restart proxmox-logger
EOF
chmod +x pkg/DEBIAN/postinst

cat > pkg/DEBIAN/prerm << EOF
#!/bin/bash
set -e
systemctl stop proxmox-logger
systemctl disable proxmox-logger
EOF
chmod +x pkg/DEBIAN/prerm

cat > pkg/DEBIAN/postrm << EOF
#!/bin/bash
set -e
systemctl daemon-reload
EOF
chmod +x pkg/DEBIAN/postrm

# Build the package
dpkg-deb --build pkg
mv pkg.deb proxmox-logger_1.0.0_amd64.deb
```

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

1. Create a new tag: `git tag v1.0.0`
2. Push the tag: `git push --tags`
3. GitHub Actions will automatically build and publish the package

## Configuration

Proxmox Logger can be configured using:

1. Configuration file at `/etc/proxmox-logger/config.yml`
2. Environment variables
3. Command-line flags (planned for future versions)

### Configuration File

The default configuration file is located at `/etc/proxmox-logger/config.yml`:

```yaml
# RabbitMQ Connection Settings
rabbitmq:
  uri: "amqp://guest:guest@localhost:5672/"
  queue_name: "proxmox_logs"
  exchange_name: "proxmox_exchange"  # Optional
  routing_key: "proxmox.logs"        # Used with exchange

# Rsyslog Listener Settings
rsyslog:
  protocol: "tcp"
  port: "18006"

# Logging Settings
logging:
  log_file: "/var/log/proxmox-logger.log"
  log_level: "info"
```

### Environment Variables

Environment variables can be set in `/etc/default/proxmox-logger`:

```
# RabbitMQ Settings
PROXMOX_LOGGER_RABBITMQ_URI="amqp://guest:guest@localhost:5672/"
PROXMOX_LOGGER_QUEUE_NAME="proxmox_logs"
PROXMOX_LOGGER_EXCHANGE=""
PROXMOX_LOGGER_ROUTING_KEY=""

# Rsyslog Settings
PROXMOX_LOGGER_PROTOCOL="tcp"
PROXMOX_LOGGER_PORT="18006"

# Logging Settings
PROXMOX_LOGGER_LOG_FILE="/var/log/proxmox-logger.log"

# Configuration file location
PROXMOX_LOGGER_CONFIG="/etc/proxmox-logger/config.yml"
```

## Log File

Proxmox Logger automatically logs to both stdout and a log file at `/var/log/proxmox-logger.log`. You can change the log file location by setting the `PROXMOX_LOGGER_LOG_FILE` environment variable or updating the configuration file.

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
/usr/local/bin/proxmox-logger
```

The application will start listening for rsyslog messages on the configured port and forward matching logs to RabbitMQ.

## Lifecycle Management

The Debian package includes proper service lifecycle management:

- **Installation**: The service is automatically enabled and started
- **Removal**: The service is properly stopped and disabled before removal
- **Upgrade**: The service is restarted after upgrade

## Log Format

The application captures logs in the following format:

```
<30>Mar 24 12:04:10 Axoft pvedaemon[669766]: <root@pam> end task UPID:Axoft:0017ABAC:026FA8FF:67E12009:qmstart:103:root@pam: OK
```

## License

[MIT License](LICENSE) 