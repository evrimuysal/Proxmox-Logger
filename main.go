package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/streadway/amqp"
	"gopkg.in/yaml.v2"
)

// Config stores the application configuration
type Config struct {
	RabbitMQ struct {
		URI          string `yaml:"uri"`
		QueueName    string `yaml:"queue_name"`
		ExchangeName string `yaml:"exchange_name"`
		RoutingKey   string `yaml:"routing_key"`
	} `yaml:"rabbitmq"`
	Rsyslog struct {
		Protocol string `yaml:"protocol"`
		Port     string `yaml:"port"`
	} `yaml:"rsyslog"`
	Logging struct {
		LogFile  string `yaml:"log_file"`
		LogLevel string `yaml:"log_level"`
	} `yaml:"logging"`
}

// Default configuration values
const (
	DEFAULT_CONFIG_PATH  = "/etc/proxmox-logger/config.yml"
	DEFAULT_PROTOCOL     = "tcp"
	DEFAULT_PORT         = "18006"
	DEFAULT_RABBITMQ_URI = "amqp://guest:guest@localhost:5672/"
	DEFAULT_QUEUE_NAME   = "proxmox_logs"
	DEFAULT_LOG_FILE     = "/var/log/proxmox-logger.log"
)

var proxmoxLogPattern = regexp.MustCompile(`<\d+>.*pvedaemon\[\d+\].*UPID:.*`)
var config Config
var logger *log.Logger

func main() {
	// Initialize logging
	initLogging()

	// Load configuration
	err := loadConfig()
	if err != nil {
		logger.Printf("Warning: Could not load configuration: %v. Using defaults.", err)
	}

	// Connect to RabbitMQ
	conn, err := amqp.Dial(config.RabbitMQ.URI)
	if err != nil {
		logger.Fatalf("Failed to connect to RabbitMQ: %s", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		logger.Fatalf("Failed to create channel: %s", err)
	}
	defer ch.Close()

	// Declare queue
	q, err := ch.QueueDeclare(
		config.RabbitMQ.QueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Fatalf("Failed to create queue: %s", err)
	}

	// If exchange is specified, declare it and bind the queue
	if config.RabbitMQ.ExchangeName != "" {
		err = ch.ExchangeDeclare(
			config.RabbitMQ.ExchangeName,
			"topic", // Exchange type
			true,    // Durable
			false,   // Auto-deleted
			false,   // Internal
			false,   // No-wait
			nil,     // Arguments
		)
		if err != nil {
			logger.Fatalf("Failed to declare exchange: %s", err)
		}

		err = ch.QueueBind(
			q.Name,                       // Queue name
			config.RabbitMQ.RoutingKey,   // Routing key
			config.RabbitMQ.ExchangeName, // Exchange
			false,
			nil,
		)
		if err != nil {
			logger.Fatalf("Failed to bind queue: %s", err)
		}
	}

	logger.Printf("Starting Proxmox Logger. Listening on %s:%s, forwarding to RabbitMQ",
		config.Rsyslog.Protocol, config.Rsyslog.Port)

	go startRsyslogListener(config.Rsyslog.Protocol, config.Rsyslog.Port, ch, q.Name)

	// Keep the application running
	select {}
}

// Loads configuration from file or environment variables
func loadConfig() error {
	// Set default values
	config.Rsyslog.Protocol = DEFAULT_PROTOCOL
	config.Rsyslog.Port = DEFAULT_PORT
	config.RabbitMQ.URI = DEFAULT_RABBITMQ_URI
	config.RabbitMQ.QueueName = DEFAULT_QUEUE_NAME
	config.Logging.LogFile = DEFAULT_LOG_FILE

	// Check environment variables for config path
	configPath := os.Getenv("PROXMOX_LOGGER_CONFIG")
	if configPath == "" {
		configPath = DEFAULT_CONFIG_PATH
	}

	// Try to load config file
	data, err := ioutil.ReadFile(configPath)
	if err == nil {
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return fmt.Errorf("error parsing config file: %v", err)
		}
		logger.Printf("Configuration loaded from %s", configPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error reading config file: %v", err)
	}

	// Override with environment variables if they exist
	if val := os.Getenv("PROXMOX_LOGGER_RABBITMQ_URI"); val != "" {
		config.RabbitMQ.URI = val
	}
	if val := os.Getenv("PROXMOX_LOGGER_QUEUE_NAME"); val != "" {
		config.RabbitMQ.QueueName = val
	}
	if val := os.Getenv("PROXMOX_LOGGER_EXCHANGE"); val != "" {
		config.RabbitMQ.ExchangeName = val
	}
	if val := os.Getenv("PROXMOX_LOGGER_ROUTING_KEY"); val != "" {
		config.RabbitMQ.RoutingKey = val
	}
	if val := os.Getenv("PROXMOX_LOGGER_PROTOCOL"); val != "" {
		config.Rsyslog.Protocol = val
	}
	if val := os.Getenv("PROXMOX_LOGGER_PORT"); val != "" {
		config.Rsyslog.Port = val
	}
	if val := os.Getenv("PROXMOX_LOGGER_LOG_FILE"); val != "" {
		config.Logging.LogFile = val
	}

	return nil
}

// Set up logging to file and/or stdout
func initLogging() {
	// Default to stdout initially
	logger = log.New(os.Stdout, "", log.LstdFlags)

	// If log file path is provided via env var, use it immediately
	logFile := os.Getenv("PROXMOX_LOGGER_LOG_FILE")
	if logFile == "" {
		// Use default path
		logFile = DEFAULT_LOG_FILE
	}

	// Try to create log file directory if it doesn't exist
	if logFile != "" && logFile != "stdout" {
		logDir := filepath.Dir(logFile)
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			err := os.MkdirAll(logDir, 0755)
			if err != nil {
				logger.Printf("Warning: Could not create log directory %s: %v", logDir, err)
			}
		}

		// Open log file
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logger.Printf("Warning: Could not open log file %s: %v", logFile, err)
		} else {
			// Use MultiWriter to write to both stdout and file
			mw := io.MultiWriter(os.Stdout, f)
			logger = log.New(mw, "", log.LstdFlags)
			logger.Printf("Logging to %s and stdout", logFile)
		}
	}
}

func startRsyslogListener(protocol, port string, ch *amqp.Channel, queueName string) {
	addr, err := net.ResolveTCPAddr(protocol, ":"+port)
	if err != nil {
		logger.Fatalf("Failed to resolve address: %v", err)
	}

	listener, err := net.ListenTCP(protocol, addr)
	if err != nil {
		logger.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	logger.Printf("Rsyslog listener started on %s:%s (for Proxmox pvedaemon messages)", protocol, port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleRsyslogConnection(conn, ch, queueName)
	}
}

func handleRsyslogConnection(conn net.Conn, ch *amqp.Channel, queueName string) {
	defer conn.Close()

	buffer := make([]byte, 4096)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			logger.Printf("Read error: %v", err)
			return
		}

		if n > 0 {
			message := string(buffer[:n])

			logs := strings.Split(message, "\n")
			for _, logMsg := range logs {
				if logMsg = strings.TrimSpace(logMsg); logMsg != "" {

					if strings.Contains(logMsg, "pvedaemon") && proxmoxLogPattern.MatchString(logMsg) {
						logger.Printf("Proxmox log found: %s", logMsg)
						sendToRabbitMQ(logMsg, ch, queueName)
					}
				}
			}
		}
	}
}

func sendToRabbitMQ(logMessage string, ch *amqp.Channel, queueName string) {
	body := fmt.Sprintf("[proxmox] %s", logMessage)

	publishing := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(body),
		Timestamp:   time.Now(),
	}

	// If exchange is configured, publish to the exchange
	if config.RabbitMQ.ExchangeName != "" {
		err := ch.Publish(
			config.RabbitMQ.ExchangeName,
			config.RabbitMQ.RoutingKey,
			false,
			false,
			publishing,
		)
		if err != nil {
			logger.Printf("RabbitMQ publish error: %v", err)
		}
	} else {
		// Otherwise publish directly to the queue
		err := ch.Publish(
			"",
			queueName,
			false,
			false,
			publishing,
		)
		if err != nil {
			logger.Printf("RabbitMQ publish error: %v", err)
		}
	}
}
