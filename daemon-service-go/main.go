// Package implements a daemon service that periodically generates
// random device metrics and events, and publishes them to NATS subjects.
package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// Constants for default configuration and subject names.
const (
	defaultNatsURL            = "nats://nats:4222"
	EventsSubject             = "events.event"   // NATS subject for  events
	DeviceMetricsSubject      = "events.metrics" // NATS subject for device metrics
	defaultGenerationInterval = 1                // Default time in seconds between each event/metric generation cycle
)

// Represents a simulated event.
type Event struct {
	ID           string `json:"id"`
	Criticality  int    `json:"criticality"` // Criticality level (e.g., 1-10).
	Timestamp    string `json:"timestamp"`   // UTC timestamp (RFC3339Nano format).
	SourceDevice string `json:"sourceDevice"`
	EventType    string `json:"eventType"` // The type of  event
}

// Represents a simulated device metric
type DeviceMetric struct {
	Timestamp    string  `json:"timestamp"`
	SourceDevice string  `json:"sourceDevice"`
	MetricType   string  `json:"metricType"` //The type of metric
	Value        float64 `json:"value"`
}

// List of available simulated devices and event/metric types.
var (
	sourceDevices = []string{
		"StorageArray", // General storage system
		"DiskUnit",     // Individual disk drive
		"CloudStorage", // Cloud integration point
	}

	// eventTypes are critical storage-related incidents.
	eventTypes = []string{
		"DriveFailure",       // Disk drive hardware failure
		"DataCorruption",     // Data integrity issue
		"UnauthorizedAccess", // Security breach attempt
	}

	// metricTypes are core storage performance and health indicators.
	metricTypes = []string{
		"DiskTemp",     // Drive temperature
		"IOPs",         // Input/Output Operations Per Second
		"Latency",      // Data access latency
		"CapacityUsed", // Storage capacity utilization
	}
)

func main() {

	// Read NATS URL from environment variable or use default
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = defaultNatsURL
	}

	// Connect to NATS server
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Daemon Service (Go): Failed to connect to NATS: %v", err)
	}
	defer nc.Close()
	log.Printf("Daemon Service (Go): Connected to NATS at %s", natsURL)

	// Read generation interval from environment variable
	generationIntervalStr := os.Getenv("GENERATION_INTERVAL_SECONDS")
	generationInterval, err := strconv.Atoi(generationIntervalStr)
	if err != nil || generationInterval <= 0 {
		generationInterval = defaultGenerationInterval
	}

	log.Printf("Daemon Service (Go): Publishing events to '%s' and metrics to '%s' every %d second(s).",
		EventsSubject, DeviceMetricsSubject, generationInterval)

	// Create a new ticker that sends a signal on its channel.
	ticker := time.NewTicker(time.Duration(generationInterval) * time.Second)
	defer ticker.Stop()

	// Create a new random number generator instance
	randGen := rand.New(rand.NewSource(time.Now().UnixNano()))

	for range ticker.C {

		// Generate and publish device metrics
		for _, device := range sourceDevices {
			metric := generateDeviceMetric(device, randGen)
			metricJSON, err := json.Marshal(metric)
			if err != nil {
				log.Printf("Daemon: Failed to serialize metric for device '%s': %v", device, err)
				continue
			}
			err = nc.Publish(DeviceMetricsSubject, metricJSON)
			if err != nil {
				log.Printf("Daemon: Error publishing metric from device '%s': %v", device, err)
			} else {
				log.Printf("Daemon: Published metric [%s] from device [%s]", metric.MetricType, metric.SourceDevice)
			}
		}

		// Generate and publish events with a lower probability
		if randGen.Float32() < 0.25 {
			event := generateEvent(randGen)
			eventJSON, err := json.Marshal(event)
			if err != nil {
				log.Printf("Daemon: Failed to serialize event '%s' from device '%s': %v", event.EventType, event.SourceDevice, err)
				continue
			}
			err = nc.Publish(EventsSubject, eventJSON)
			if err != nil {
				log.Printf("Daemon: Error publishing event [%s] from [%s]: %v", event.EventType, event.SourceDevice, err)
			} else {
				log.Printf("Daemon: Published event [%s] from [%s] with criticality [%d]", event.EventType, event.SourceDevice, event.Criticality)
			}
		}
	}
}

// Creates a random event
func generateEvent(randGen *rand.Rand) Event {
	device := sourceDevices[randGen.Intn(len(sourceDevices))]
	eventType := eventTypes[randGen.Intn(len(eventTypes))]
	criticality := randGen.Intn(10) + 1 // Random int from 1 to 10

	return Event{
		ID:           uuid.New().String(),
		Criticality:  criticality,
		Timestamp:    time.Now().Format(time.RFC3339Nano),
		SourceDevice: device,
		EventType:    eventType,
	}
}

// Creates a random device metric
func generateDeviceMetric(device string, randGen *rand.Rand) DeviceMetric {
	metricType := metricTypes[randGen.Intn(len(metricTypes))]
	value := 0.0

	// Generate values
	switch metricType {
	case "DiskTemp":
		value = 25.0 + randGen.Float64()*35.0 // Disk temperature: 25.0 to 60.0
	case "IOPs":
		value = 100.0 + randGen.Float64()*900.0 // I/O Operations Per Second: 100 to 1000
	case "Latency":
		value = 0.5 + randGen.Float64()*10.0 // Latency: 0.5 to 10.5
	case "CapacityUsed":
		value = 10.0 + randGen.Float64()*85.0 // Capacity utilization: 10.0 to 95.0 %
	default:
		// Fallback for any unexpected metric types
		value = randGen.Float64() * 100.0 // 0.0 to 100.0
	}

	return DeviceMetric{
		Timestamp:    time.Now().Format(time.RFC3339Nano),
		SourceDevice: device,
		MetricType:   metricType,
		Value:        value,
	}
}
