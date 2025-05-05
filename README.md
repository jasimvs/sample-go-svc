# sample-go-svc
A sample http svc in Golang


## Requirements
Lets implement a simple suspicious transactions detecting service. I'll create 2 endpoints, create transaction, and get suspected transactions. The detection algorithms in the background after a transaction is created. 

Lets create some sample business rules to flag a transaction:
- Flag any transaction over a certain amount
- Flag transactions when more than X transactions by a user below $D within an hour
- Flag transactions when 3 or more consecutive transfer transactions by a user within 5 minutes


## Design, tradeoffs 
We could run the detection algorithms on create, but this will slow down the response. It's good practice to seperate the concerns and to do background processing after creating a resource - keep the create flow simple and respond quickly. Trigger a check in a goroutine (or in a background thread/callback in other languauges). However, this will be lost in the event of a server restart. 
To be durable, you need to either save a state in DB(INIT, ANALYZED - perhaps ANALYZING if its not idempotent or expensive and want to avoid reprocessing in parallel) to rerun in the event of restarts. For high scale, generate events and process them, and you can reduce DB writes to only update flagged transactions instead of persisting state change to all transactions. 
When writing to external systems twice (in this case create-txn and process-txn event or store in DB), to ensure every is processed in all failure scenarios - use CDC.  
For this demo app, we will keep things simple, no CDC, and use the in memory go channels to mimic an external persisted queue. 
When processing fails, say DB is not accessible, should retry later. Can do sync retries, but for better reliability would need async retries with external queues
We will use SQLite to easily run a DB integration tests without spinning up a database - just delete the data/ folder to reset DB.

When building an API you would typically need the following. Not implementing these in this sample app
- Authentication - ensure user is logged in for e.g. presenting a JWT Auth token, which can be verified to not to be tampered without a lookup, and have the user id in it, and actions are permitted for that user's resources.
- ALB to distribute load and uptime -  assuming you have multiple instances of service running
- Rate limiting to prevent DDOS (by IP, user ID, or more advanced heuristics to detect coordinated attacks)
- Traceability - adding open tracing info to requests makes debugging in distributed systems easy and quick. We wont be adding tracing
- OpenAPI doc - good for public APIs or exposing APIs to other teams or 3rd party. We wont be adding this 
- Logging - In real, typically a log aggregator will push the logs to an logging tool like ELK, Datadog or New Relic.
- Health endpoints for checking uptime

Creating your API using database model is quick, but its better to have separate models for communication and data storage. For e.g. You often dont want a client API to pass the transaction ID or timestamp from client when creating. And you may want to add modified_at field or metadata in storage, and mostly not needed to expose it via an API.

I will describe my journey for creating this service:
- Created a REST endpoint - Post transaction which simply responds with a 201 (no functionality implemented)
- Added lint, build, run scripts (used Makefile)
- Added a DB with repository pattern with integration tests, wired up the create transaction flow 
- Created detection module, moved Transaction model to a shared model package, used channel to separate concerns,  create and suspected transaction detection processing. Initially planned to create a detection table with transaction id (foreign key) and the relevant fields. However, looking at the Get suspicious transactions requirement, realized would have to join the tables for all GET requests. Not good for scaling, so the other choice is to duplicate all required data in a the suspected transactions table. If data storage cost isn't is a concern, that's a good idea. However, to begin with I just added the new fields to the existing transactions table, will refactor it later.
- Added the business rules for marking transaction successful
- Added the Get suspicious transactions REST endpoint with functionality 


## Run
Prerequisites:
Install Makefile: `brew install make`
Install Golang: `brew install go`

To run: `make run`

To lint:
Install golangci-lint: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
Install goimports: `go install golang.org/x/tools/cmd/goimports@latest`

To view pretty JSON output, install jq `brew install jq
`

```
sample-go-svc git:(main) ✗ make      
Usage: make [target]

Available targets:
  build      Build the Go application binary
  clean      Remove the built binary
  fix        Auto-fix lint issues and format code (requires golangci-lint, gofmt, goimports in PATH)
  help       Display this help screen
  lint       Run linters (requires golangci-lint in PATH)
  run        Build and run the application locally
  tidy       Tidy Go module files
  ```

Modify port in config.yaml, it's set to 9090 by default

## Use

```
curl -v -X POST http://localhost:9090/api/v1/transaction \
     -H "Content-Type: application/json" \
     -d '{ "userId":"user_1", "amount": 41005.00, "type": "withdrawal"}'

Response:
{"id":"tx_a81bd484-8c2c-43dd-a8d2-63994c22fb40","userId":"user_1","amount":41005,"type":"withdrawal","timestamp":"2025-05-05T19:45:30.090705Z"}

```


```
curl -X GET "http://localhost:9090/api/v1/transactions?user_id=user_1&suspicious=true" | jq .

Response:
[
  {
    "id": "tx_cd7bb804-afc0-46fc-b0f2-69eb64951205",
    "user_id": "user_1",
    "amount": 41005,
    "type": "withdrawal",
    "timestamp": "2025-05-05T19:47:30.500329Z",
    "is_suspicious": true,
    "flagged_rules": [
      "HighVolumeTransaction"
    ]
  }
]

```
