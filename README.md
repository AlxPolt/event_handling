
# Event Handling 

**Microservices architecture built with Go, Python, NATS, InfluxDB, Docker, and Grafana**

This project implements a distributed event handling system built with a microservices architecture. It's designed to simulate, process, and visualize real-time data, showcasing robust inter-service communication and data persistence.

This system actively simulates device metrics and critical events from various data storage components like DiskUnit and StorageArray. These simulated data points, such as DiskTemp, IOPs, DriveFailure, and DataCorruption, are continuously published to NATS. From there, they flow into a time-series database (InfluxDB) for real-time storage and analysis in Grafana, enabling immediate insights into system health and potential issues.

## Architecture
```txt
Daemon (Go) → NATS → Writer (Go) → InfluxDB
Reader (Python) → NATS → Client (Go)
Grafana ↔ InfluxDB
```

## Tech Stack
 - Go: Used for high-performance event generation and data writing services.
 - Python: Utilized for a flexible event processing and reading service.
 - NATS: Serves as the lightweight, high-performance message broker for asynchronous communication between all microservices.
 - InfluxDB: A time-series database optimized for storing large volumes of time-stamped event and metric data.
 - Grafana: Provides powerful real-time visualization and analytics dashboards, connecting to InfluxDB.
 - Docker & Docker Compose: For containerizing all services and defining their relationships, ensuring easy setup, deployment, and management.
 - Bash Scripts: Provide a convenient command-line interface for managing the entire system, including building, running, and logging services.

## Microservices Overview
- **Daemon** *(Go)*: generates JSON events and publishes to NATS.
- **Writer** *(Go)*: listens to NATS events and stores them in InfluxDB, leverages Go's concurrency model to handle incoming NATS messages.
- **Reader** *(Python)*: fetches relevant time-series data from InfluxDB, performs computations (e.g. filtering critical alerts, detecting anomalies, evaluating device health), and returns structured JSON responses. 
- **Client** *(Go)*: sends queries to the Reader service via NATS and writes critical event summaries to a local log file 
- **Grafana** *(Visual Tool)*: Connects to InfluxDB and visualizes the event stream.

####  Example Event
```json
{
  "id": "e4d5f6a7-8b9c-0d1e-2f3a-4b5c6d7e8f90",
  "criticality": 8,
  "timestamp": "2025-06-06T12:08:40.987654321Z",
  "sourceDevice": "StorageArray",
  "eventType": "UnauthorizedAccess"
}
```
####  Example DeviceMetric
```json
{
  "timestamp": "2025-06-06T12:08:40.123456789Z",
  "sourceDevice": "DiskUnit",
  "metricType": "DiskTemp",
  "value": 45.75
}
```


## 🚀 How to Run

### 1. Clone the Repository
```bash
git clone https://github.com/AlxPolt/event_handling.git
cd event_handling
```

### 2. Start the Platform

The project includes a convenient run.sh Bash script to manage the entire system lifecycle. To build all Docker images and start all services, execute:

```bash
./run.sh
```

#### Script Usage

You can manage the entire project lifecycle using the included `run.sh` script. This utility simplifies initialization, building, running, and debugging services.

```bash
./run.sh <command> [options]
```

#####  Available Commands:

- `./run.sh init`  
  Initializes Go modules and installs dependencies for all Go-based services.

- `./run.sh build`  
  Builds Docker images for all services defined in `docker-compose.yml`.

- `./run.sh start`  
  Starts all containers using Docker Compose.

- `./run.sh stop`  
  Stops and removes all running containers.

- `./run.sh restart`  
  Performs a full restart: stops containers, cleans up, re-initializes dependencies, rebuilds images, and starts services again.

- `./run.sh logs`  
  Displays logs from all running containers in real-time.

- `./run.sh logs <service-name>`  
  Shows logs only from a specific service (e.g., `daemon-service-go`, `reader-service-py`).

- `./run.sh all`  
  Executes the full sequence: initialization → build → start → logs.

- `./run.sh replace "<SEARCH_STRING>" "<REPLACE_STRING>" [FILE_EXTENSION]`  
  Replaces text recursively in files (e.g., change URLs or env values). Optional third argument limits the replacement to files with a specific extension (e.g., `.go` or `.py`).



### 3. Access Interfaces
- **InfluxDB**: [http://localhost:8086](http://localhost:8086)
  - Username: `admin`, Password: `MyStrongPassword888` (or as configured via environment variables)
- **Grafana**: [http://localhost:3000](http://localhost:3000)
  - Username: `admin`, Password: `admin` (or as configured via environment variables)


This project simulates a real-time pipeline with event generation, message queueing, time-series storage, and monitoring. It demonstrates microservices architecture, streaming data ingestion, schema-less storage, and analytics dashboards 
