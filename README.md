# Campaign Service

This document outlines how to set up and run the campaign service locally.

## Local Setup Instructions:

1. **Clone the repository:**
   ```bash
   git clone git@github.com:TechLead-War/campaignservice.git
   cd campaignservice
   ```

2. **Run Migrations:**
   Ensure you have the [golang-migrate/migrate](https://github.com/golang-migrate/migrate) tool installed.
   ```bash
   # Example:
   # migrate -path migrations -database "postgres://postgres:password@localhost:5432/campaign_service?sslmode=disable" up
   # Adjust DB connection string as per your local PostgreSQL setup.
   # The default DB name used in docker-compose is 'campaigns' with user/pass 'postgres'.
   migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/campaigns?sslmode=disable" up
   ```

3. **Seed Data (Optional):**
   To populate the database with sample data, run the seed script from the `cmd/seed` directory:
   ```bash
   # Example: From the root of the project
   go run ./cmd/seed/main.go -records=2000 -workers=20
   ```
   This command will insert 2000 records using 20 concurrent workers. Adjust parameters as needed.

4. **Run the Service:**
   The project uses [Air](https://github.com/cosmtrek/air) for live reloading during development.
   Ensure `air` is installed (`go install github.com/cosmtrek/air@latest`).
   ```bash
   # From the root of the project
   air
   ```
   The service will typically be available at `http://localhost:8080`.
   The `.air.toml` file configures the live reload behavior.

## Docker-based Setup:

For a containerized setup, refer to the Docker files in the `deployments/docker/` directory.
You can typically start the entire stack (service, database, Prometheus, Grafana) using Docker Compose:
```bash
# From the deployments/docker/ directory
docker-compose up --build
```
This method handles database migrations automatically via the entrypoint script.

### Note:
- This repository has been tested with Go `1.23.x` (see `go.mod` and `Dockerfile`) and PostgreSQL `14` (see `docker-compose.yml`).
- The system has been tested with a significant volume of data (e.g., 3 million+ records).