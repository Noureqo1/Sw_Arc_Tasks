# Circuit Breaker Implementation

This task implements a circuit breaker pattern with retry mechanism and exponential backoff for handling transient failures in distributed systems.

## Features

- Circuit breaker with three states (Closed, Open, Half-Open)
- Configurable failure thresholds and reset timeouts
- Retry mechanism with exponential backoff
- Simple API gateway integration
- Configurable parameters

## API Endpoints

### GET /api/request - Test endpoint that simulates external service calls
- successful request
![api-request](tests/task4/Screenshot%202025-03-28%20143613.png)
- failed request
![api-request](tests/task4/Screenshot%202025-03-28%20143911.png)

### GET /api/state - Get current circuit breaker state

- open state
![api-state](tests/task4/Screenshot%202025-03-28%20143930.png)
- closed state
![api-state](tests/task4/Screenshot%202025-03-28%20143802.png)

### Testing Scenarios

1. Normal Operation:
   - Send requests to `/api/request` repeatedly
   - Observe successful responses and occasional failures

2. Circuit Breaking:
   - Send requests rapidly to trigger failures
   - After 3 failures, circuit should open
   - Requests will be rejected for 5 seconds

3. Recovery:
   - Wait 5 seconds after circuit opens
   - Send new requests to observe half-open state
   - After 2 successful requests, circuit should close

## Configuration

Current settings (can be modified in main.go):
- Failure Threshold: 3 failures
- Reset Timeout: 5 seconds
- Half-Open Success Needed: 2 requests
- Max Retries: 3
- Base Retry Delay: 100ms
- Max Retry Delay: 2s
