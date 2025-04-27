# sample-go-svc
A sample http svc in Golang


## Requirements
Lets implement a simple suspicious transactions detecting service. I'll create 2 endpoints, create transaction, and get suspected transactions. The detection algorithms in the background after a transaction is created. 

Lets create some sample business rules to flag a transaction:
- Flag any transaction over a certain amount
- Flag a user with more than X transactions below $D within an hour
- Flag a user with 3 or more consecutive transfer transactions within 5 minutes


## Design, tradeoffs 
We could run the detection algorithms on create, but this will slow down the response. It's good practice to seperate the concerns and to do background processing after creating a resource - keep the create flow simple and respond quickly. Trigger a check in a goroutine (or in a background thread/callback in other languauges). However, this will be lost in the event of a server restart. 
To be durable, you need to either save a state in DB(INIT, ANALYZED - perhaps ANALYZING if its not idempotent or expensive and want to avoid reprocessing in parallel) to rerun in the event of restarts. For high scale, generate events and process them, and you can reduce DB writes to only update flagged transactions instead of persisting state change to all transactions. 
When writing to external systems twice (in this case create-txn and process-txn event or store in DB), to ensure every is processed in all failure scenarios - use CDC.  
For this demo app, we will keep things simple, no CDC, and use the in memory go channels to mimic an external persisted queue. 
We will use SQLite to easily run a DB without spinning up a database - just delete the data/ folder to reset DB.

When building an API you would typically need the following. Not implementing these in this sample app
- Authentication - ensure user is logged in for e.g. presenting a JWT Auth token, which can be verified to not to be tampered without a lookup, and have the user id in it, and actions are permitted for that user's resources.
- ALB to distribute load and uptime -  assuming you have multiple instances of service running
- Rate limiting to prevent DDOS (by IP, user ID, or more advanced heuristics to detect coordinated attacks)
- Traceability - adding open tracing info to requests makes debugging in distributed systems easy and quick. We wont be adding tracing
- OpenAPI doc - good for public APIs or exposing APIs to other teams or 3rd party. We wont be adding this 
- Logging - In this sample app, we will log to a file. In real, typically a log aggregator will push the logs to an logging tool like ELK, Datadog or New Relic. 

Creating your API using database model is quick, but its better to have separate models for communication and data storage. For e.g. You often dont want a client API to pass the transaction ID or timestamp from client when creating. And you may want to add modified_at field or metadata in storage, and there isnt a reason to expose it via an API.

Build and run:
Install Makefile
Install Golang
Install golangci-lint: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
Install goimports `go install golang.org/x/tools/cmd/goimports@latest`


