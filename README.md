# Smart Meter Bridge MQTT Companion

Smart Meter Bridge MQTT companion reads from the [Smart Meter Bridge](https://github.com/legolasbo/smartmeterBridge) service and writes to a configurable MQTT topic.

## Installation

### Prerequisites

- Go 1.25 or later
- Access to a running Smart Meter Bridge instance
- Access to an MQTT broker

### Building

```bash
go build -o smartmeterbridge-mqtt bridge.go
```

## Configuration

Copy the example configuration file:

```bash
cp config.example.yml config.yml
```

Edit `config.yml` with your settings:

```yaml
# Smart Meter Bridge TCP connection
bridge:
  host: localhost
  port: 9988

# MQTT broker connection
mqtt:
  broker: tcp://192.168.10.11:1883
  client_id: p1-backup-forwarder
  topic: p1/raw/telegram
  qos: 1
  retain: false

# Logging
verbose: false
```

## Usage

Run the binary:

```bash
./smartmeterbridge-mqtt
```

The application will:
1. Connect to the Smart Meter Bridge TCP server
2. Connect to the configured MQTT broker
3. Read P1 telegrams from the bridge
4. Publish complete telegrams to the configured MQTT topic

## Development

### Dependencies

- [Eclipse Paho MQTT Go client](https://github.com/eclipse/paho.mqtt.golang)
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3)

### Running in development

```bash
go run bridge.go
```
