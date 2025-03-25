# Proxmox Logger

A lightweight Go application that captures Proxmox pvedaemon logs and forwards them to RabbitMQ.

## Overview

Proxmox Logger is a service that listens for Proxmox logs sent via rsyslog over TCP, filters for specific pvedaemon task messages, and forwards them to a RabbitMQ message queue for further processing. It's designed to be lightweight, efficient, and easy to integrate with existing Proxmox VE environments.

## Features

- TCP listener for rsyslog messages
- Pattern matching for Proxmox pvedaemon task logs (UPID messages)
- RabbitMQ integration for message queue publishing
- YAML configuration file support
- Environment variable configuration
- RabbitMQ exchange and routing key support
- Integrated rsyslog restart on service restart
- Lightweight and efficient
- Automated Debian package building with GitHub Actions
- Fully integrated systemd service management
- Configuration validation and RabbitMQ connection testing

## Requirements

- Go 1.20+ (for building from source)
- A RabbitMQ server
- Proxmox VE server with rsyslog

## Installation

### Using apt (Debian/Ubuntu)

```bash
# Add repository
echo "deb [trusted=yes] https://evrimuysal.github.io/Proxmox-Logger/deb stable main" | sudo tee /etc/apt/sources.list.d/proxmox-logger.list

# Update package list
sudo apt update

# Install package
sudo apt install proxmox-logger
```

After installation, you need to configure the application:

1. Create the configuration directory:
```bash
sudo mkdir -p /etc/proxmox-logger
```

2. Create the configuration file:
```bash
sudo nano /etc/proxmox-logger/config.yml
```

3. Add your RabbitMQ connection details:
```yaml
rabbitmq:
  uri: "amqp://username:password@host:5672/"  # Replace with your RabbitMQ connection string
  queue_name: "proxmox_logs"
  exchange_name: ""  # Optional
  routing_key: "proxmox.logs"  # Used with exchange

rsyslog:
  protocol: "tcp"
  port: "18006"
```

4. Start the service:
```bash
sudo systemctl start proxmox-logger
```

The service will validate the configuration and test the RabbitMQ connection before starting. If there are any issues, check the logs:
```bash
sudo journalctl -u proxmox-logger
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

Follow the same configuration steps as described above.

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

4. Install as a systemd service:
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
EnvironmentFile=-/etc/default/proxmox-logger
Restart=on-failure
User=root
Group=root
ExecReload=/bin/kill -HUP \$MAINPID
ExecStop=/bin/kill -TERM \$MAINPID

[Install]
WantedBy=multi-user.target
EOF

# Create configuration directory and file
sudo mkdir -p /etc/proxmox-logger
sudo nano /etc/proxmox-logger/config.yml

# Reload systemd, enable and start the service
sudo systemctl daemon-reload
sudo systemctl enable proxmox-logger
sudo systemctl start proxmox-logger
```

## Configuration

Proxmox Logger can be configured using:

1. Configuration file at `/etc/proxmox-logger/config.yml`
2. Environment variables in `/etc/default/proxmox-logger`

### Configuration File

The default configuration file is located at `/etc/proxmox-logger/config.yml`:

```yaml
# RabbitMQ Connection Settings
rabbitmq:
  uri: "amqp://guest:guest@localhost:5672/"  # Replace with your RabbitMQ connection string
  queue_name: "proxmox_logs"
  exchange_name: ""  # Optional
  routing_key: "proxmox.logs"  # Used with exchange

# Rsyslog Listener Settings
rsyslog:
  protocol: "tcp"
  port: "18006"
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

# Configuration file location
PROXMOX_LOGGER_CONFIG="/etc/proxmox-logger/config.yml"
```

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

The application will validate the configuration and test the RabbitMQ connection before starting. If there are any issues, it will display appropriate error messages.

## Log Format

The application captures logs in the following format:

```
<30>Mar 24 12:04:10 Axoft pvedaemon[669766]: <root@pam> end task UPID:Axoft:0017ABAC:026FA8FF:67E12009:qmstart:103:root@pam: OK
```

## License

[Apache License 2.0](LICENSE) 