package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/streadway/amqp"
	"gopkg.in/yaml.v2"
)

// Config defines application configuration settings including RabbitMQ and Rsyslog configurations.
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
}

// DEFAULT_CONFIG_PATH defines the default path to the configuration file.
// DEFAULT_PROTOCOL specifies the default protocol to be used.
// DEFAULT_PORT specifies the default port for the application.
// DEFAULT_RABBITMQ_URI provides the URI for connecting to RabbitMQ.
// DEFAULT_QUEUE_NAME defines the default name of the RabbitMQ queue.
const (
	DEFAULT_CONFIG_PATH  = "/etc/proxmox-logger/config.yml"
	DEFAULT_PROTOCOL     = "tcp"
	DEFAULT_PORT         = "18006"
	DEFAULT_RABBITMQ_URI = "amqp://guest:guest@localhost:5672/"
	DEFAULT_QUEUE_NAME   = "proxmox_logs"
)

// proxmoxLogPattern is a compiled regular expression used to match log messages from Proxmox's pvedaemon service.
var proxmoxLogPattern = regexp.MustCompile(`<\d+>.*pvedaemon\[\d+\].*UPID:.*`)

// config holds the application configuration, including RabbitMQ and Rsyslog settings loaded from YAML or environment.
var config Config

// logger is a centralized logging instance used throughout the application for logging messages and errors.
var logger *log.Logger

// validateConfig checks if the configuration is valid and required fields are set.
func validateConfig() error {
	if config.RabbitMQ.URI == "" {
		return fmt.Errorf("RabbitMQ URI is required")
	}
	if config.RabbitMQ.QueueName == "" {
		return fmt.Errorf("RabbitMQ queue name is required")
	}
	return nil
}

// testRabbitMQConnection tests the connection to RabbitMQ and returns an error if the connection fails.
func testRabbitMQConnection() error {
	conn, err := amqp.Dial(config.RabbitMQ.URI)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to create channel: %v", err)
	}
	defer ch.Close()

	// Test queue declaration
	_, err = ch.QueueDeclare(
		config.RabbitMQ.QueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	// Test exchange if configured
	if config.RabbitMQ.ExchangeName != "" {
		err = ch.ExchangeDeclare(
			config.RabbitMQ.ExchangeName,
			"topic",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange: %v", err)
		}

		err = ch.QueueBind(
			config.RabbitMQ.QueueName,
			config.RabbitMQ.RoutingKey,
			config.RabbitMQ.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue: %v", err)
		}
	}

	return nil
}

// main initializes logging, loads configuration, connects to RabbitMQ, sets up queues, and starts the Rsyslog listener.
func main() {
	logger = log.New(os.Stdout, "", log.LstdFlags)

	// Check if config file exists
	if _, err := os.Stat(DEFAULT_CONFIG_PATH); os.IsNotExist(err) {
		logger.Printf("Configuration file not found at %s", DEFAULT_CONFIG_PATH)
		logger.Printf("Please create the configuration file with RabbitMQ connection details")
		logger.Printf("Example configuration:")
		logger.Printf(`
rabbitmq:
  uri: "amqp://username:password@host:5672/"
  queue_name: "proxmox_logs"
  exchange_name: ""  # Optional
  routing_key: "proxmox.logs"  # Used with exchange

rsyslog:
  protocol: "tcp"
  port: "18006"
`)
		os.Exit(1)
	}

	err := loadConfig()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := validateConfig(); err != nil {
		logger.Fatalf("Invalid configuration: %v", err)
	}

	// Test RabbitMQ connection
	if err := testRabbitMQConnection(); err != nil {
		logger.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	logger.Printf("Successfully connected to RabbitMQ at %s", config.RabbitMQ.URI)

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

	if config.RabbitMQ.ExchangeName != "" {
		err = ch.ExchangeDeclare(
			config.RabbitMQ.ExchangeName,
			"topic",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			logger.Fatalf("Failed to declare exchange: %s", err)
		}

		err = ch.QueueBind(
			q.Name,
			config.RabbitMQ.RoutingKey,
			config.RabbitMQ.ExchangeName,
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

	select {}
}

// loadConfig reads configuration settings from a file and environment variables and initializes default values if necessary.
// It returns an error if the configuration file cannot be read or parsed.
func loadConfig() error {
	config.Rsyslog.Protocol = DEFAULT_PROTOCOL
	config.Rsyslog.Port = DEFAULT_PORT
	config.RabbitMQ.URI = DEFAULT_RABBITMQ_URI
	config.RabbitMQ.QueueName = DEFAULT_QUEUE_NAME

	configPath := os.Getenv("PROXMOX_LOGGER_CONFIG")
	if configPath == "" {
		configPath = DEFAULT_CONFIG_PATH
	}

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

	return nil
}

// startRsyslogListener starts an rsyslog listener using the specified protocol and port, handling incoming connections.
// It processes rsyslog messages, filtering for Proxmox-specific logs and forwards them to a RabbitMQ queue provided.
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

// handleRsyslogConnection processes incoming rsyslog TCP connections and filters Proxmox logs for forwarding to RabbitMQ.
// It reads messages from the connection, splits them into logs, and identifies Proxmox-related messages using regex.
// Relevant log messages are sent to a RabbitMQ queue or exchange for further processing.
// The function ensures the connection is closed when processing is complete.
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

// sendToRabbitMQ sends a log message to a specified RabbitMQ queue or exchange using the provided AMQP channel.
// logMessage is the message to send, ch is the AMQP channel, and queueName is the target queue if no exchange is specified.
// The message body is prefixed with "[proxmox]" and published with a content type of "text/plain".
// If an exchange is defined in the config, the message is routed using the configured routing key.
// Logs an error if the publishing to RabbitMQ fails.
func sendToRabbitMQ(logMessage string, ch *amqp.Channel, queueName string) {
	body := fmt.Sprintf("[proxmox] %s", logMessage)

	publishing := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(body),
		Timestamp:   time.Now(),
	}

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
