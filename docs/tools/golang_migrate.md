# golang-migrate

Source: https://github.com/golang-migrate/migrate

## How to use with docker

1. Open terminal
2. Go to the project directory
    ```bash
    cd /path/to/gitlab-package-finder
    ```
3. Generate a new migration
    ```bash
    docker run -v ./migrations:/migrations migrate/migrate create -ext mysql -dir /migrations -seq MIGRATION_NAME
    ```
4. New files will be created in the folder `migrations`. Write mysql queries in these files
