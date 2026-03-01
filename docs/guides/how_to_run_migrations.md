# How to run migrations

## Local

1. Open a terminal
2. Go to the project directory
   ```bash
    cd /path/to/gitlab-package-finder
    ```
3. Run new migrations
   ```bash
   go run ./cmd migrate up
   ```
4. Rollback migrations
   ```bash
   go run ./cmd migrate down
   ```
5. Execute migrations with custom path to migrations folder
   ```bash
   DB_MIGRATIONS_PATH=./migrations go run ./cmd migrate up
   ```
