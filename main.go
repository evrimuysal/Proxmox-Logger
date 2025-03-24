package main

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/streadway/amqp"
)

const (
	RSYSLOG_PROTOCOL = "tcp"
	RSYSLOG_PORT     = "18006"
	RABBITMQ_URI     = "amqp://guest:guest@192.168.1.5:5672/"
)

var proxmoxLogPattern = regexp.MustCompile(`<\d+>.*pvedaemon\[\d+\].*UPID:.*`)

func main() {

	conn, err := amqp.Dial(RABBITMQ_URI)
	failOnError(err, "RabbitMQ bağlantısı başarısız")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Kanal oluşturulamadı")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"proxmox_logs",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Kuyruk oluşturulamadı")

	go startRsyslogListener(RSYSLOG_PROTOCOL, RSYSLOG_PORT, ch, q.Name)

	select {}
}

func startRsyslogListener(protocol, port string, ch *amqp.Channel, queueName string) {
	addr, err := net.ResolveTCPAddr(protocol, ":"+port)
	if err != nil {
		log.Fatalf("Adres çözülemedi: %v", err)
	}

	listener, err := net.ListenTCP(protocol, addr)
	if err != nil {
		log.Fatalf("Dinleyici başlatılamadı: %v", err)
	}
	defer listener.Close()

	log.Printf("Rsyslog dinleyicisi %s:%s üzerinde başlatıldı (Proxmox pvedaemon mesajları için)", protocol, port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Bağlantı kabul edilemedi: %v", err)
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
			log.Printf("Okuma hatası: %v", err)
			return
		}

		if n > 0 {
			message := string(buffer[:n])

			logs := strings.Split(message, "\n")
			for _, logMsg := range logs {
				if logMsg = strings.TrimSpace(logMsg); logMsg != "" {

					if strings.Contains(logMsg, "pvedaemon") && proxmoxLogPattern.MatchString(logMsg) {
						log.Printf("Proxmox log bulundu: %s", logMsg)
						sendToRabbitMQ(logMsg, ch, queueName)
					}
				}
			}
		}
	}
}

func sendToRabbitMQ(logMessage string, ch *amqp.Channel, queueName string) {
	body := fmt.Sprintf("[proxmox] %s", logMessage)
	err := ch.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
			Timestamp:   time.Now(),
		},
	)
	if err != nil {
		log.Printf("RabbitMQ gönderim hatası: %v", err)
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
