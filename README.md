# Exchange Rates

This project is a Go-based API server that retrieves and caches exchange rate data and currency lists from external APIs. It is designed to minimize API calls to external services by using an in-memory caching system that refreshes data every hour.

## Prerequisite

- Go version 1.23.3

## Steps to Run the Application
1. Set the APP_ID Environment Variable:

    - On Linux/Mac:

        ```bash
        export APP_ID="your_api_key"
        ```

    - On Windows (PowerShell):

        ```powershell
        $env:APP_ID="your_api_key"
        ```

2. Run the Application:

    ```bash
    go mod tidy
    go run main.go
    ```

The server will now use the APP_ID from the environment variable, making it more secure and configurable.

## Test

- http://localhost:8080/api/latest
- http://localhost:8080/api/currencies
