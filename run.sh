#!/bin/bash

set -e

# --- Configuration ---
PROJECT_ROOT=$(dirname "$(realpath "$0")") # Get the directory where the script is located
GO_SERVICES=("daemon-service-go" "writer-service-go" "client-service-go") # List of Go services
PYTHON_SERVICES=("reader-service-py") # List of Python services
ALL_SERVICES=("${GO_SERVICES[@]}" "${PYTHON_SERVICES[@]}")

# --- Functions ---

# Function to initialize Go modules and tidy dependencies
init_go_modules() {
    echo "--- Initializing Go modules and tidying dependencies ---"
    for service_dir in "${GO_SERVICES[@]}"; do
        if [ -d "$PROJECT_ROOT/$service_dir" ]; then
            echo "Initializing Go module for: $service_dir"
            (cd "$PROJECT_ROOT/$service_dir" && go mod init "$service_dir" && go mod tidy) || { echo "Failed to init/tidy Go module in $service_dir"; exit 1; }
        else
            echo "Warning: Go service directory $service_dir not found."
        fi
    done
    echo "--- Go module initialization complete ---"
}

# Function to install Python dependencies
install_python_deps() {
    echo "--- Installing Python dependencies ---"
    for service_dir in "${PYTHON_SERVICES[@]}"; do
        if [ -d "$PROJECT_ROOT/$service_dir" ]; then
            if [ -f "$PROJECT_ROOT/$service_dir/requirements.txt" ]; then
                echo "Installing Python dependencies for: $service_dir"
                # Build a temporary container to install deps. This is more robust for CI/CD like scenarios.
                # Or simply run pip install locally if you have python environment.
                # For Docker Compose, dependencies are installed during image build.
                echo "Python dependencies for $service_dir will be installed during Docker image build."
            else
                echo "Warning: requirements.txt not found in $service_dir."
            fi
        else
            echo "Warning: Python service directory $service_dir not found."
        fi
    done
    echo "--- Python dependency installation check complete ---"
}

# Function to build Docker images
build_images() {
    echo "--- Building Docker images ---"
    docker-compose -f "$PROJECT_ROOT/docker-compose.yml" build --no-cache || { echo "Failed to build Docker images"; exit 1; }
    echo "--- Docker image build complete ---"
}

# Function to bring up Docker containers
start_containers() {
    echo "--- Starting Docker containers ---"
    docker-compose -f "$PROJECT_ROOT/docker-compose.yml" up -d || { echo "Failed to start Docker containers"; exit 1; }
    echo "--- All services are up and running ---"
}

# Function to stop and remove Docker containers and networks
stop_and_clean() {
    echo "--- Stopping and removing Docker containers and networks ---"
    docker-compose -f "$PROJECT_ROOT/docker-compose.yml" down -v --remove-orphans || { echo "Failed to stop/clean Docker containers"; exit 1; }
    echo "--- Cleanup complete ---"
}

# Function to view logs of a specific service or all services
view_logs() {
    if [ -z "$1" ]; then
        echo "--- Showing logs for all services ---"
        docker-compose -f "$PROJECT_ROOT/docker-compose.yml" logs -f
    else
        echo "--- Showing logs for service: $1 ---"
        docker-compose -f "$PROJECT_ROOT/docker-compose.yml" logs -f "$1"
    fi
}

# Function to find and replace text in files
find_and_replace() {
    if [ -z "$1" ] || [ -z "$2" ]; then
        echo "Usage: $0 replace <search_string> <replace_string> [file_extension]"
        echo "  Searches for <search_string> in files (optionally filtered by <file_extension>)"
        echo "  and replaces it with <replace_string> across the project."
        echo "  Example: $0 replace \"old_token\" \"new_secret\" .env"
        echo "  Example: $0 replace \"http://localhost:8086\" \"http://influxdb:8086\""
        exit 1
    fi

    SEARCH_STRING="$1"
    REPLACE_STRING="$2"
    FILE_EXTENSION="$3" # Optional argument

    echo "--- Searching for '$SEARCH_STRING' and replacing with '$REPLACE_STRING' ---"

    # Use a temporary file for sed's output to avoid issues with in-place editing
    # and ensure compatibility across different sed versions (macOS/Linux)
    if [ -n "$FILE_EXTENSION" ]; then
        # If file extension is provided, limit search to those files
        find "$PROJECT_ROOT" -type f -name "*$FILE_EXTENSION" -print0 | xargs -0 sed -i '' -e "s|$SEARCH_STRING|$REPLACE_STRING|g"
    else
        # Search in common text-based source files if no extension specified
        # Excludes directories like .git and binary files
        find "$PROJECT_ROOT" -type f \
             -not -path "*/.git/*" \
             -not -path "*/node_modules/*" \
             -not -path "*/venv/*" \
             -not -path "*/__pycache__/*" \
             -not -name "*.log" \
             -exec grep -l "$SEARCH_STRING" {} + | xargs -r sed -i '' -e "s|$SEARCH_STRING|$REPLACE_STRING|g"
    fi

    echo "--- Replacement complete ---"
}

# --- Main Script Logic ---

case "$1" in
    init)
        init_go_modules
        install_python_deps # This just prints a message, actual installation is in Dockerfile
        ;;
    build)
        build_images
        ;;
    start)
        start_containers
        ;;
    stop)
        stop_and_clean
        ;;
    restart)
        stop_and_clean
        init_go_modules
        install_python_deps
        build_images
        start_containers
        ;;
    logs)
        view_logs "$2"
        ;;
    all)
        stop_and_clean
        init_go_modules
        install_python_deps
        build_images
        start_containers
        view_logs # Show logs after starting
        ;;
    replace) # <-- НОВАЯ КОМАНДА
        find_and_replace "$2" "$3" "$4"
        ;;
    *)
        echo "Usage: $0 {init|build|start|stop|restart|logs [service_name]|all}"
        echo "  init: Initializes Go modules and checks Python dependencies."
        echo "  build: Builds Docker images."
        echo "  start: Starts Docker containers in detached mode."
        echo "  stop: Stops and removes Docker containers and networks."
        echo "  restart: Stops, cleans, initializes, builds, and starts all services."
        echo "  logs: Shows logs for all services or a specific service."
        echo "  all: Runs init, build, start, and then shows logs."
        exit 1
        ;;
esac