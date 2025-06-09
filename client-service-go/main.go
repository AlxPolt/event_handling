package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	natsSubjectRequest = "reader.query"
)

type ReaderRequest struct {
	QueryType string                 `json:"query_type"`
	Params    map[string]interface{} `json:"params"`
}

type ReaderResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	fmt.Println("Client started")
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		fmt.Printf("Failed to connect to NATS: %v\n", err)
		return
	}
	defer nc.Close()

	sendQueries(nc)
}

func sendQueries(nc *nats.Conn) {
	queries := []ReaderRequest{
		{
			QueryType: "alerts_critical",
			Params: map[string]interface{}{
				"since_minutes":   15,
				"min_criticality": 8,
			},
		},
		{
			QueryType: "device_health",
			Params: map[string]interface{}{
				"source_device": "sensor-1",
			},
		},
		{
			QueryType: "anomaly_temperature",
			Params: map[string]interface{}{
				"source_device":  "sensor-1",
				"threshold":      1.3,
				"window_minutes": 20,
			},
		},
	}

	for _, req := range queries {
		sendQuery(nc, req)
		time.Sleep(1 * time.Second)
	}
}

func sendQuery(nc *nats.Conn, request ReaderRequest) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		writeToFile("client_output.log", fmt.Sprintf("Failed to marshal request: %v", err))
		return
	}

	msg, err := nc.Request(natsSubjectRequest, requestJSON, 10*time.Second)
	if err != nil {
		writeToFile("client_output.log", fmt.Sprintf("Request failed: %v", err))
		return
	}

	var response ReaderResponse
	err = json.Unmarshal(msg.Data, &response)
	if err != nil {
		writeToFile("client_output.log", fmt.Sprintf("Failed to unmarshal response: %v", err))
		return
	}

	if response.Status == "success" {
		formatted := formatJSON(response.Data)
		writeToFile("client_output.log", fmt.Sprintf("QueryType: %s\n%s\n", request.QueryType, formatted))
	} else {
		writeToFile("client_output.log", fmt.Sprintf("QueryType: %s\nError: %s\n", request.QueryType, response.Message))
	}
}

func writeToFile(filename string, content string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s\n", content))
	return err
}

func formatJSON(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting JSON: %v", err)
	}
	return string(b)
}
