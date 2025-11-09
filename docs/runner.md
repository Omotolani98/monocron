# Runner

The API Server that listens for incoming requests to the Daemon outside the machine. It is driven by REST APIs.

## Setup

### Clone the repo

### Build Docker Image & Push to Desired Container Image Registry

### Pull Image to Target VM and Run

```shell
docker run -d --name runner -p 5050:5050 -v /run/monocron:/run/monocron:rw image-name:tag
```

## Testing

Here are the lists of APIs

|*Name*|*Path*|*Description*|
|---|---|---|
|Schedule Cron|`POST /api/v1/monocron/schedules`|handles creation of cron on the server|
|Get All Jobs|`GET /api/v1/monocron/schedules`|Fetches all jobs|
|   |   |   |

Sample body for `POST /api/v1/monocron/schedules`

```json
{
    "name": "backup",
    "schedule": "*/5 * * * * *",
    "timezone": "Africa/Lagos",
    "timeout": 10,
    "argv": [
        "mkdir",
        "HelloFolder"
    ]
}
```
