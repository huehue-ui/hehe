# Campaign Service

This service manages advertising campaigns and provides an API endpoint for campaign delivery based on targeting rules.

It's built with Go, uses PostgreSQL for data storage, Redis for caching, and Prometheus for monitoring.

## Features
- Campaign management (though CRUD APIs for campaigns are not yet exposed in this version)
- Targeted campaign delivery via `/v1/delivery` endpoint
- Caching of delivery responses using Redis
- Metrics exposed for Prometheus

## Prerequisites
- Go (version 1.20+ recommended, tested with 1.21)
- PostgreSQL (version 13+ recommended)
- Redis (version 5+)
- Docker & Docker Compose (for running DB, Redis, and Prometheus easily)
- `migrate` CLI (for database migrations): [golang-migrate/migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)

## Setup and Running

### 1. Clone Repository
```bash
git clone <your-repo-url> # Replace <your-repo-url> with the actual URL
cd campaignservice
```

### 2. Configuration (.env file)
Create a `.env` file in the root of the project. You can copy `.env.example` if it exists, or create a new one.
Fill in your local configuration details. Example:
```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=youruser
DB_PASSWORD=yourpassword
DB_NAME=campaigndb

# Application Port
APP_PORT=8080

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Metrics Server Port (for Prometheus)
# This is configured in main.go (default: 9091) and not directly read from .env for the port itself.
```
**Note:** The application loads configuration from environment variables. The `.env` file is primarily for convenience in local development (loaded by `godotenv`). In production environments, ensure these environment variables are set directly.

### 3. Using Docker Compose (Recommended for Local Development)
The `deployments/docker/docker-compose.yml` file can be used to spin up PostgreSQL, Redis, and Prometheus.
```bash
docker-compose -f deployments/docker/docker-compose.yml up -d postgres redis prometheus
```
This will start:
- PostgreSQL on port `5432` (accessible at `localhost:5432`)
- Redis on port `6379` (accessible at `localhost:6379`)
- Prometheus on port `9090` (UI accessible at `http://localhost:9090`)

Wait for these services to initialize and be ready.

### 4. Database Migrations
Ensure your PostgreSQL instance (local or Docker) is running and accessible. The migration command uses the database connection details.
Update the connection string in the command if your settings (from `.env` or environment) are different.
```bash
# Example using values from a typical .env file (ensure these are set in your environment or .env):
# export DB_USER=youruser
# export DB_PASSWORD=yourpassword
# export DB_HOST=localhost
# export DB_PORT=5432
# export DB_NAME=campaigndb
# migrate -path migrations -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" up

# Or provide the connection string directly:
migrate -path migrations -database "postgres://youruser:yourpassword@localhost:5432/campaigndb?sslmode=disable" up
```
Replace `youruser`, `yourpassword`, `localhost`, `5432`, `campaigndb` with your actual DB connection details if they differ.

### 5. Seed Data (Optional)
The project includes a seeder to populate the database with sample campaigns and targeting rules. This uses the same database configuration as the main application.
```bash
go run cmd/seed/main.go -records=2000 -workers=20
```
Adjust `-records` and `-workers` flags as needed.

### 6. Build and Run the Application
There are two main ways to run the application:

**a) Using `air` (for live reloading during development, if `.air.toml` is configured):**
```bash
air
```
This will typically run the application on the port specified by `APP_PORT` (default `8080`).

**b) Standard Go build and run:**
```bash
go build -o campaignservice cmd/api/main.go
./campaignservice
```
The application server will start (default: `http://localhost:8080`).
The metrics server will start on its configured port (default `9091`, serving at `http://localhost:9091/metrics`).

## API Endpoints

### Campaign Delivery
- **GET `/v1/delivery`**
  - Fetches relevant campaigns based on targeting parameters.
  - **Query Parameters:**
    - `app` (string, required): Application ID.
    - `os` (string, required): Operating system (e.g., "ios", "android").
    - `country` (string, required): Country code (e.g., "US", "GB").
    - `page` (int, optional, default: 1): For pagination.
    - `limit` (int, optional, default: 10, max: 100): Number of items per page.
  - **Example:**
    ```bash
    curl "http://localhost:8080/v1/delivery?app=mygame&os=ios&country=US&page=1&limit=5"
    ```
  - **Response:** JSON array of `DeliveryResponse` objects or an empty array `[]`.
    ```json
    [
      {
        "cid": "campaign_id_123",
        "img": "http://example.com/image.png",
        "cta": "Click Here!"
      }
    ]
    ```
  - **Caching:** Responses are cached in Redis for 5 minutes. Successful responses include an `X-Cache` header (e.g., `X-Cache: HIT` or `X-Cache: MISS`).

### Metrics
- **GET `/metrics`** (served on the metrics port, default `:9091`)
  - Exposes application metrics in Prometheus format.
  - Example: `http://localhost:9091/metrics` (if running locally with default port)

## Running Tests
To run all unit tests in the project:
```bash
go test ./...
```
This command will discover and execute all `*_test.go` files.

## Monitoring
- **Prometheus:** The service exposes metrics for Prometheus on a dedicated port (default `:9091`, path `/metrics`). The Prometheus server should be configured to scrape this target. If using the provided Docker Compose setup for Prometheus, it's pre-configured to scrape `campaignservice:9091`. The Prometheus UI can be accessed (default `http://localhost:9090`).
- **Grafana:** While not included in the current `docker-compose.yml`, Prometheus can serve as a data source for Grafana, allowing for the creation of dashboards to visualize the application metrics.

## Project Structure
A brief overview of the project layout:
```
.
├── cmd/
│   ├── api/main.go        # Main application entry point, server setup
│   └── seed/main.go       # Data seeder utility
├── deployments/
│   ├── docker/            # Dockerfile and Docker Compose configurations
│   └── monitoring/        # Prometheus configuration (prometheus.yml)
├── internal/
│   ├── api/handler/       # HTTP request handlers (e.g., delivery.go)
│   ├── domain/models/     # Core domain models and configuration structures
│   ├── infrastructure/
│   │   ├── db/db.go       # Database interaction logic (queries, connection)
│   │   └── cache/         # (Placeholder for more complex Redis logic if needed)
├── migrations/            # Database schema migration files (.sql)
├── pkg/
│   ├── utils/             # Shared utility functions (metrics, HTTP wrappers, error constants)
├── .air.toml              # Configuration for Air live reload tool (optional)
├── .env                   # Local environment variables (should be gitignored)
├── go.mod                 # Go module definition file
├── go.sum                 # Go module checksums
└── README.md              # This file
```

## Contributing
Please ensure that tests pass (`go test ./...`) and consider updating documentation for any significant changes or new features.
This repo was initially tested with Go 1.21 and Postgres 15.4.