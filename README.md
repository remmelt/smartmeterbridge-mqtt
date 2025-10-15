# Smart Meter Bridge MQTT Companion

Smart Meter Bridge MQTT companion provides bidirectional integration between the [Smart Meter Bridge](https://github.com/legolasbo/smartmeterBridge) service and MQTT brokers.

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
# Role: "publisher" or "subscriber"
role: publisher

# Smart Meter Bridge TCP connection (required for publisher role)
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

# Backup configuration (required for subscriber role)
backup:
  path: /mnt/backup/datagrams

# Logging
verbose: false
```

### Roles

The application supports two roles:

#### Publisher Mode (default)
Reads P1 telegrams from Smart Meter Bridge TCP server and publishes them to MQTT.

**Required configuration:**
- `bridge.host` and `bridge.port` - Smart Meter Bridge connection details
- `mqtt.*` - MQTT broker connection details

#### Subscriber Mode
Subscribes to MQTT topic and saves received P1 telegrams to dated backup files.

**Required configuration:**
- `mqtt.*` - MQTT broker connection details
- `backup.path` - Base directory for backups

Telegrams are saved to: `{backup.path}/YYYY/MM/DD.log` with timestamp prefixes.

**Features:**
- Automatic directory structure creation
- Timestamped telegram entries
- Automatic MQTT reconnection handling
- Graceful error handling

## Usage

Run the binary:

```bash
./smartmeterbridge-mqtt
```

The application behavior depends on the configured role:

### Publisher Mode
1. Connect to the Smart Meter Bridge TCP server
2. Connect to the configured MQTT broker
3. Read P1 telegrams from the bridge
4. Publish complete telegrams to the configured MQTT topic

### Subscriber Mode
1. Connect to the configured MQTT broker
2. Subscribe to the configured topic
3. Save received telegrams to dated log files
4. Automatically handle reconnections and errors

Example log file structure:
```
/mnt/backup/datagrams/
├── 2025/
│   └── 10/
│       ├── 15.log
│       ├── 16.log
│       └── 17.log
```

Each log entry format:
```
[2025-10-15 14:30:45]
/ISK5\2M550T-1012

1-3:0.2.8(50)
0-0:1.0.0(231015143045S)
...
!
```

## Development

### Dependencies

- [Eclipse Paho MQTT Go client](https://github.com/eclipse/paho.mqtt.golang)
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3)

### Running in development

```bash
go run bridge.go
```
