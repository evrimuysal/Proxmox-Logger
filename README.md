# Proxmox Logger

A lightweight Go application that captures Proxmox pvedaemon logs and forwards them to RabbitMQ.

## Overview

This application listens for Proxmox logs sent via rsyslog over TCP, filters for specific pvedaemon task messages, and forwards them to a RabbitMQ message queue for further processing.

## Features

- TCP listener for rsyslog messages
- Pattern matching for Proxmox pvedaemon task logs (UPID messages)
- RabbitMQ integration for message queue publishing
- Lightweight and efficient

## Requirements

- Go 1.18+ (for building from source)
- A RabbitMQ server
- Proxmox VE server with rsyslog

## Installation

### Using apt (Debian/Ubuntu)

1. Add the repository:
```bash
# Add the GPG key
curl -fsSL https://your-github-repo.github.io/proxmox-logger/gpg | sudo gpg --dearmor -o /usr/share/keyrings/proxmox-logger-archive-keyring.gpg

# Add the repository
echo "deb [signed-by=/usr/share/keyrings/proxmox-logger-archive-keyring.gpg] https://your-github-repo.github.io/proxmox-logger/deb stable main" | sudo tee /etc/apt/sources.list.d/proxmox-logger.list

# Update package list
sudo apt update
```

2. Install the package:
```bash
sudo apt install proxmox-logger
```

3. Start the service:
```bash
sudo systemctl enable proxmox-logger
sudo systemctl start proxmox-logger
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

### Building Debian Package

1. Install build dependencies:
```bash
sudo apt install devscripts debhelper golang-go
```

2. Build the package:
```bash
dpkg-buildpackage -us -uc
```

The package will be created in the parent directory.

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