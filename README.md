# notiffly: A Real-Time Notification Platform with Kafka and Go

## Overview

notiffly is a highly scalable, real-time notification platform built using Kafka and Go. It enables you to efficiently deliver notifications to a large number of users or systems in a robust and reliable manner.

## Key Features

- Real-time Delivery
- Scalability
- Flexibility
- Reliability
- Fault Tolerance
- Easy Integration

## Routemap

- [ ] Refactor code to apply better modularization
- [ ] Implement docket compose file to start kafka, producer and consumer at once each service in its own docker
- [ ] Integrate data persistency

## Getting Started

In order to start the system you need to run kafka first, then open the consumer and finally the producer.

### Set Up Kafka

```Bash
docker-compose up -d
```

## Build and Run

Open a bash console

```Bash
cd notiffly

go run .\cmd\consumer\consumer.go
```

Open another bach console

```Bash
cd notiffly

go run .\cmd\producer\producer.go
```

## API Reference

Notiffly provides the following REST API endpoints:

### Consumer

```Text
GET /notifications/:id:
Fetches a specific notification by its ID.
```

Example:

```Bash
curl -X GET http://localhost:8081/notifications/2
```

### Producer

```Text
POST /send:
Sends a new notification.
Body (x-www-form-urlencoded):

fromId: Integer representing the sender's ID.
toId: Integer representing the recipient's ID.
message: String containing the notification message.
```

```Bash
curl -X POST http://localhost:8080/send \
    -d 'fromId=123&toId=456&message=Hello%20world!'
```
