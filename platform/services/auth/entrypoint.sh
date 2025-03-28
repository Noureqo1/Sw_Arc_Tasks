#!/bin/sh
set -e

export PGPASSWORD=postgres

echo "Waiting for Postgres..."
until pg_isready -h db -p 5432 -U postgres
do
  echo "Postgres is unavailable - sleeping"
  sleep 1
done
echo "Postgres is up!"

echo "Running raw SQL migrations..."
for file in $(ls /app/migrations/*.sql | sort); do
  echo "Applying migration: $file"
  psql -h db -U postgres -d authdb -f "$file"
done

echo "All migrations applied successfully."
exec /auth

