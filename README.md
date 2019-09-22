# api-server
This is the API service for lsof.top - its written in Go.

## Endpoints
- `/cache`
  - `/cache/record`
    - Per-record management in the cache
    - `/cache/record/purge`
      - `DELETE` method
  - `/cache/purge` (Admin/Staff only)
    - Management of all cache records
- `/domain` (Admin/Staff only)
  - `/domain/create`
    - Create a domain
  - `/domain/delete`
    - `DELETE` method
    - Delete a domain
- `/record`
  - `/record/create`
    - Create a record
  - `/record/update`
    - Update a record
  - `/record/delete`
    - `DELETE` method
    - Delete a record


# Quickstart
```
1. docker build -t api-server .
2. docker run --net=host -it api-server
```

# To-do
- Move X-Domain header to POST data
- Add additional record fields to POST data
- Change cache purge to GET and POST request 