# api-server
This is the API service for lsof.top - its written in Go.

## Endpoints
- `/cache/purge`
  - This purges the cache record in question
- `/record`
  - `/record/create`
    - Create a record
  - `/record/update`
    - Update a record

# Quickstart
```
1. docker build -t api-server .
2. docker run --net=host -it api-server
```

# To-do
- Move X-Domain header to POST data
- Add additional record fields to POST data
- Change cache purge to GET and POST request 