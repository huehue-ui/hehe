#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status.

echo "Starting entrypoint script..."

# Validate that necessary environment variables are set
: "${DB_HOST?DB_HOST not set or empty}"
: "${DB_PORT?DB_PORT not set or empty}"
: "${DB_USER?DB_USER not set or empty}"
: "${DB_PASSWORD?DB_PASSWORD not set or empty}"
: "${DB_NAME?DB_NAME not set or empty}"

echo "DB_HOST: ${DB_HOST}"
echo "DB_PORT: ${DB_PORT}"
echo "DB_USER: ${DB_USER}"
# Do not echo DB_PASSWORD for security reasons
echo "DB_NAME: ${DB_NAME}"

# Wait for the PostgreSQL database to be ready
echo "Checking for DB readiness..."
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -q; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 2
done
echo "PostgreSQL is up and running."

# Define the path to the migration files within the container
MIGRATIONS_PATH="/app/migrations" # Corrected path as per Dockerfile COPY
DATABASE_URL="postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable"

echo "Running database migrations from ${MIGRATIONS_PATH}..."
# Apply database migrations using the migrate tool
# The migrate binary should be in PATH (e.g., /usr/local/bin/migrate)
if migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" up; then
  echo "Database migrations applied successfully."
else
  echo "Database migrations failed. Check logs for details."
  # Optionally, exit here if migrations are critical for startup
  # exit 1
fi

echo "Starting the main application..."
# Execute the main application (passed as CMD in Dockerfile)
exec "$@"