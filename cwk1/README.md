# RetailDB Go Service

Lightweight Go web service for the `retailDB.sqlite` dataset (COMP3011 coursework).

## Features

- SQLite-backed service that aggregates across `info`, `brands`, `finance`, `reviews`, and `traffic` tables
- Admin auth endpoint to generate JWTs
- Endpoints:
  - `POST /auth/login` -> {username, password} -> returns `token`
  - `GET /products` -> list (query `limit` and `offset`)
  - `GET /products/:id` -> product aggregate
  - `PUT /admin/products/:id` -> update modified\_\* fields (requires Bearer token)
  - `GET /healthz`

## Requirements

- Go 1.20+
- `retailDB.sqlite` placed in the project root (or set `RETAILDB_PATH`)

## Quick start

1. Copy or set environment variables (see `.env.example`):

```
ADMIN_USER=admin
ADMIN_PASSWORD=password
JWT_SECRET=your-secret
RETAILDB_PATH=./retailDB.sqlite
PORT=8080
```

2. Run the service:

```
go run .
```

3. Login and get a token:

```
POST /auth/login
{ "username": "admin", "password": "password" }
```

4. Use token with `Authorization: Bearer <token>` for `PUT /admin/products/:id`.

## API docs

- OpenAPI spec available at: `/openapi.yaml`
- Interactive docs (Swagger UI) at: `/docs` (loads `/openapi.yaml`)

## Notes & next steps

- No Docker included as requested.
- Consider storing admin credentials securely and using hashed passwords for production.
- I can add tests, more endpoints and OpenAPI docs next.
