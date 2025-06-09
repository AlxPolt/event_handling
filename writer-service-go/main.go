package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/nats-io/nats.go"
)

// Constants for default configuration and subject names
const (
	defaultNatsURL      = "nats://nats:4222"
	natsSubjectWildcard = "events.*"           // Wildcard to subscribe to all event types (events.security, events.metrics)
	natsQueueGroup      = "writer_queue_group" // NATS queue group for distributed consumption
	defaultInfluxDBHost = "http://influxdb:8086"
	eventsMeasurement   = "events"         // InfluxDB measurement for all generic events (e.g., DriveFailure, UnauthorizedAccess)
	metricsMeasurement  = "device_metrics" // InfluxDB measurement for device metrics (e.g., DiskTemp, IOPs)
)

// Event represents a generic event, including security events (matches daemon-go's structure more closely)
type Event struct {
	ID           string `json:"id"`
	Criticality  int    `json:"criticality"`
	Timestamp    string `json:"timestamp"`
	SourceDevice string `json:"sourceDevice"`
	EventType    string `json:"eventType"`
	EventMessage string `json:"eventMessage"` // Added field for the event message
}

// DeviceMetric represents a device metric (compact structure)
type DeviceMetric struct {
	Timestamp    string  `json:"timestamp"`
	SourceDevice string  `json:"sourceDevice"`
	MetricType   string  `json:"metricType"`
	Value        float64 `json:"value"`
}

func init() {
	// Configure logger to show file and line number for easier debugging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetPrefix("Writer Service (Go): ")
}

func main() {
	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal. Initiating graceful shutdown...")
		cancel() // Cancel the context to signal goroutines to stop
	}()

	// 1. Get configuration from environment variables
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = defaultNatsURL
	}

	influxDBHost := os.Getenv("INFLUXDB_HOST")
	if influxDBHost == "" {
		influxDBHost = defaultInfluxDBHost
	}
	influxDBToken := os.Getenv("INFLUXDB_TOKEN")
	influxDBOrg := os.Getenv("INFLUXDB_ORG")
	influxDBBucket := os.Getenv("INFLUXDB_BUCKET")

	if influxDBToken == "" || influxDBOrg == "" || influxDBBucket == "" {
		log.Fatalf("InfluxDB token, organization, or bucket environment variables are not set. Please check your .env file.")
	}

	// 2. Connect to NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer func() {
		log.Println("Closing NATS connection...")
		nc.Close()
	}()
	log.Printf("Connected to NATS at %s", natsURL)

	// 3. Connect to InfluxDB
	client := influxdb2.NewClient(influxDBHost, influxDBToken)
	defer func() {
		log.Println("Closing InfluxDB client...")
		client.Close()
	}()

	writeAPI := client.WriteAPIBlocking(influxDBOrg, influxDBBucket)
	log.Printf("Connected to InfluxDB at %s, Org: %s, Bucket: %s", influxDBHost, influxDBOrg, influxDBBucket)

	// Ping InfluxDB to check connection
	_, err = client.Health(ctx)
	if err != nil {
		log.Fatalf("InfluxDB health check failed: %v. Please ensure InfluxDB is running and accessible.", err)
	}
	log.Println("InfluxDB is healthy.")

	// 4. Subscribe to NATS subject(s) using a wildcard and a queue group
	_, err = nc.QueueSubscribe(natsSubjectWildcard, natsQueueGroup, func(m *nats.Msg) {
		go func(m *nats.Msg) {
			switch m.Subject {
			case "events.event":
				handleEvent(ctx, m.Data, writeAPI)
			case "events.metrics":
				handleDeviceMetric(ctx, m.Data, writeAPI)
			default:
				log.Printf("Received unknown message type on subject: %s", m.Subject)
			}
		}(m) // передаём m внутрь горутины
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to NATS subject wildcard '%s' with queue group '%s': %v", natsSubjectWildcard, natsQueueGroup, err)
	}

	log.Printf("Subscribed to NATS subject wildcard '%s' in queue group '%s'. Waiting for messages...", natsSubjectWildcard, natsQueueGroup)

	// Keep the service running until context is cancelled (e.g., by OS signal)
	<-ctx.Done()
	log.Println("Writer Service (Go): Shutting down.")
}

// handleEvent processes and writes a generic event to InfluxDB
func handleEvent(ctx context.Context, data []byte, writeAPI api.WriteAPIBlocking) {
	var event Event // Use the updated Event struct
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("ERROR: Failed to unmarshal event: %v. Data: %s", err, string(data))
		return
	}

	parsedTime, err := time.Parse(time.RFC3339Nano, event.Timestamp)
	if err != nil {
		log.Printf("ERROR: Failed to parse event timestamp '%s': %v", event.Timestamp, err)
		return
	}

	p := influxdb2.NewPointWithMeasurement(eventsMeasurement). // Using 'events' as measurement name
									AddTag("event_id", event.ID).
									AddTag("criticality_level", fmt.Sprintf("%d", event.Criticality)).
									AddTag("source_device", event.SourceDevice).
									AddTag("event_type", event.EventType).
									AddField("event_message", event.EventMessage). // Add EventMessage as a field
									SetTime(parsedTime)

	if err := writeAPI.WritePoint(ctx, p); err != nil {
		log.Printf("ERROR: Failed to write event ID %s to InfluxDB: %v", event.ID, err)
	} else {
		log.Printf("Successfully wrote event ID %s (Type: %s, Device: %s, Message: '%s') to InfluxDB.", event.ID, event.EventType, event.SourceDevice, event.EventMessage)
	}
}

// handleDeviceMetric processes and writes a device metric to InfluxDB
func handleDeviceMetric(ctx context.Context, data []byte, writeAPI api.WriteAPIBlocking) {
	var metric DeviceMetric
	if err := json.Unmarshal(data, &metric); err != nil {
		log.Printf("ERROR: Failed to unmarshal device metric: %v. Data: %s", err, string(data))
		return
	}

	parsedTime, err := time.Parse(time.RFC3339Nano, metric.Timestamp)
	if err != nil {
		log.Printf("ERROR: Failed to parse device metric timestamp '%s': %v", metric.Timestamp, err)
		return
	}

	p := influxdb2.NewPointWithMeasurement(metricsMeasurement).
		AddTag("source_device", metric.SourceDevice).
		AddTag("metric_type", metric.MetricType).
		AddField("value", metric.Value). // Numerical values are typically fields
		SetTime(parsedTime)

	if err := writeAPI.WritePoint(ctx, p); err != nil {
		log.Printf("ERROR: Failed to write device metric for %s/%s to InfluxDB: %v", metric.SourceDevice, metric.MetricType, err)
	} else {
		log.Printf("Successfully wrote device metric for %s/%s (Value: %.2f) to InfluxDB.", metric.SourceDevice, metric.MetricType, metric.Value)
	}
}
