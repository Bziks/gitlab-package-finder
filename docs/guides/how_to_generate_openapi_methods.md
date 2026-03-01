# How to generate OpenAPI methods based on the OpenAPI schema

1. Open terminal
2. Go to the project directory
   ```bash
    cd /path/to/gitlab-package-finder
    ```
3. Execute the following commands to generate the OpenAPI methods:
    ```bash
    # Generate API methods for server implementation
    oapi-codegen -config ./docs/api/http/oapi-codegen-server.yaml ./docs/api/http/openapi.yaml

    # Generate models for server implementation
    oapi-codegen -config ./docs/api/http/oapi-codegen-models.yaml ./docs/api/http/openapi.yaml
    ```
