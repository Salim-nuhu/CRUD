# CRUD

A CRUD backend written in Go. This project currently exposes a health check and
an endpoint for creating users, with MongoDB as the data store.

## How the pieces fit together

Think of the application as a small restaurant:

- `cmd/api/main.go` opens the restaurant: it loads settings, connects services,
  registers HTTP routes, and starts the server.
- `internal/config/config.go` reads the settings, like a checklist the staff
  uses before opening.
- `internal/handlers/user_repository.go` is the front counter: it reads and
  validates the incoming web request.
- `internal/models/user.go` describes the shape of a user, like a blank form.
- `internal/repository/user_repository.go` stores users in MongoDB, like the
  filing room behind the counter.
- `internal/database/mongo.go` creates and checks the MongoDB connection.
- `internal/response/json.go` formats success and error replies as JSON.

## Routes

- `GET /health` returns a small JSON status response.
- `POST /users` creates a user from a JSON body containing `name` and `email`.

## Configuration

Set `PORT`, `MONGO_URI`, and `DB_NAME` as environment variables, or provide
them in a local `.env` file. `PORT` defaults to `8080`, and `DB_NAME` defaults
to `crud_api`. Keep database credentials private; do not commit a real `.env`
file.

## Run

From this project directory, run:

```powershell
go run ./cmd/api
```

MongoDB must be reachable at the address configured in `MONGO_URI`.
