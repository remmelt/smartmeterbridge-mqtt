package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/yaml.v3"
)

type Config struct {
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

	return &cfg, nil
}

func main() {
	cfg, err := loadConfig("config.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to smartmeterBridge TCP
	bridgeAddr := fmt.Sprintf("%s:%d", cfg.Bridge.Host, cfg.Bridge.Port)
	conn, err := net.Dial("tcp", bridgeAddr)
	if err != nil {
		log.Fatalf("Failed to connect to bridge: %v", err)
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
		log.Fatalf("Failed to connect to MQTT: %v", token.Error())
	}
	defer client.Disconnect(250)

	if cfg.Verbose {
		log.Printf("Connected to MQTT broker at %s", cfg.MQTT.Broker)
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
		log.Fatalf("Error reading from bridge: %v", err)
	}
}
