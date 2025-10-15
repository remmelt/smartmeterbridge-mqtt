package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Role string `yaml:"role"` // "publisher" or "subscriber"
	Bridge struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"bridge"`
	MQTT struct {
		Broker   string `yaml:"broker"`
		ClientID string `yaml:"client_id"`
		Topic    string `yaml:"topic"`
		QoS      byte   `yaml:"qos"`
		Retain   bool   `yaml:"retain"`
	} `yaml:"mqtt"`
	Backup struct {
		Path string `yaml:"path"` // Base path for subscriber backups
	} `yaml:"backup"`
	Verbose bool `yaml:"verbose"`
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Set default role
	if cfg.Role == "" {
		cfg.Role = "publisher"
	}

	// Validate role
	if cfg.Role != "publisher" && cfg.Role != "subscriber" {
		return nil, fmt.Errorf("invalid role: %s (must be 'publisher' or 'subscriber')", cfg.Role)
	}

	// Validate subscriber config
	if cfg.Role == "subscriber" && cfg.Backup.Path == "" {
		return nil, fmt.Errorf("backup.path is required when role is 'subscriber'")
	}

	return &cfg, nil
}

func runSubscriber(cfg *Config) error {
	// Connect to MQTT
	opts := mqtt.NewClientOptions().AddBroker(cfg.MQTT.Broker)
	opts.SetClientID(cfg.MQTT.ClientID)
	opts.SetAutoReconnect(true)
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		if cfg.Verbose {
			log.Printf("Connected to MQTT broker at %s", cfg.MQTT.Broker)
		}
		// Subscribe on connect/reconnect
		token := client.Subscribe(cfg.MQTT.Topic, cfg.MQTT.QoS, func(client mqtt.Client, msg mqtt.Message) {
			if err := saveTelegram(cfg, msg.Payload()); err != nil {
				log.Printf("Error saving telegram: %v", err)
			}
		})
		token.Wait()
		if token.Error() != nil {
			log.Printf("Error subscribing to topic: %v", token.Error())
		} else if cfg.Verbose {
			log.Printf("Subscribed to %s", cfg.MQTT.Topic)
		}
	})
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("Connection lost: %v (will auto-reconnect)", err)
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT: %w", token.Error())
	}

	if cfg.Verbose {
		log.Printf("Running as subscriber: MQTT -> File backup")
	}

	// Block forever
	select {}
}

func saveTelegram(cfg *Config, telegram []byte) error {
	now := time.Now()

	// Build directory path: /mnt/backup/datagrams/YYYY/MM
	dirPath := filepath.Join(cfg.Backup.Path, fmt.Sprintf("%04d", now.Year()), fmt.Sprintf("%02d", now.Month()))

	// Create directory structure if it doesn't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dirPath, err)
	}

	// Build file path: DD.log
	filePath := filepath.Join(dirPath, fmt.Sprintf("%02d.log", now.Day()))

	// Open file in append mode, create if doesn't exist
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", filePath, err)
	}
	defer file.Close()

	// Write timestamp + telegram
	timestamp := now.Format("2006-01-02 15:04:05")
	if _, err := fmt.Fprintf(file, "[%s]\n%s\n", timestamp, string(telegram)); err != nil {
		return fmt.Errorf("writing to file: %w", err)
	}

	if cfg.Verbose {
		log.Printf("Saved telegram to %s", filePath)
	}

	return nil
}

func runPublisher(cfg *Config) error {
	// Connect to smartmeterBridge TCP
	bridgeAddr := fmt.Sprintf("%s:%d", cfg.Bridge.Host, cfg.Bridge.Port)
	conn, err := net.Dial("tcp", bridgeAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to bridge: %w", err)
	}
	defer conn.Close()

	if cfg.Verbose {
		log.Printf("Connected to bridge at %s", bridgeAddr)
	}

	// Connect to MQTT
	opts := mqtt.NewClientOptions().AddBroker(cfg.MQTT.Broker)
	opts.SetClientID(cfg.MQTT.ClientID)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT: %w", token.Error())
	}
	defer client.Disconnect(250)

	if cfg.Verbose {
		log.Printf("Connected to MQTT broker at %s", cfg.MQTT.Broker)
		log.Printf("Running as publisher: Bridge -> MQTT")
	}

	// Read telegrams and publish
	scanner := bufio.NewScanner(conn)
	var telegram strings.Builder
	inTelegram := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "/") {
			// Start of telegram
			inTelegram = true
			telegram.Reset()
		}

		if inTelegram {
			telegram.WriteString(line + "\n")
		}

		if strings.HasPrefix(line, "!") {
			// End of telegram - publish
			token := client.Publish(cfg.MQTT.Topic, cfg.MQTT.QoS, cfg.MQTT.Retain, telegram.String())
			token.Wait()
			if cfg.Verbose {
				log.Printf("Published telegram to %s", cfg.MQTT.Topic)
			}
			inTelegram = false
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from bridge: %w", err)
	}

	return nil
}

func main() {
	cfg, err := loadConfig("config.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	switch cfg.Role {
	case "publisher":
		if err := runPublisher(cfg); err != nil {
			log.Fatalf("Publisher error: %v", err)
		}
	case "subscriber":
		if err := runSubscriber(cfg); err != nil {
			log.Fatalf("Subscriber error: %v", err)
		}
	default:
		log.Fatalf("Unknown role: %s", cfg.Role)
	}
}
