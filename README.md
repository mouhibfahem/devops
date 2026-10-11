# DevOps Project — Flashcards API

A small REST API to revise course topics with flashcards, used to practice DevOps
end to end: version control, CI/CD, containerization, Kubernetes, observability
and security.

The API starts with a few DevOps flashcards and has a **quiz mode**
(`GET /cards/random`) that returns a random question without its answer.

## Run locally

Requirements: [Go](https://go.dev/dl/) 1.24 or newer.

```bash
go run .                 # starts the API on http://localhost:8080
PORT=9000 go run .       # use another port
```

With `make` (Linux / macOS / Git Bash with make installed):

| Command         | What it does                         |
| --------------- | ------------------------------------ |
| `make run`      | Start the API                        |
| `make test`     | Run the unit tests                   |
| `make coverage` | Run the tests with a coverage report |
| `make lint`     | `go vet` + formatting check          |
| `make build`    | Build the binary into `bin/`         |

## API

| Method   | Path                    | Description                                   | Success |
| -------- | ----------------------- | --------------------------------------------- | ------- |
| `GET`    | `/health`               | Health check                                  | 200     |
| `GET`    | `/cards[?topic=docker]` | List cards, optionally filtered by topic      | 200     |
| `POST`   | `/cards`                | Create a card (`question` and `answer` required) | 201  |
| `GET`    | `/cards/{id}`           | Get one card                                  | 200     |
| `DELETE` | `/cards/{id}`           | Delete one card                               | 204     |
| `GET`    | `/cards/random[?topic=]`| Quiz: a random question, without the answer   | 200     |

Errors are returned as JSON, for example `{"error": "card not found"}`, with
status `400` (invalid input), `404` (not found) or `405` (method not allowed).

### Examples

```bash
curl http://localhost:8080/cards
curl "http://localhost:8080/cards?topic=docker"

curl -X POST http://localhost:8080/cards \
  -H "Content-Type: application/json" \
  -d '{"topic":"k8s","question":"What is a Pod?","answer":"The smallest deployable unit in Kubernetes"}'
# {"id":4,"topic":"k8s","question":"What is a Pod?","answer":"The smallest deployable unit in Kubernetes"}

curl http://localhost:8080/cards/random
# {"id":2,"question":"What is an SLO?","topic":"sre"}

curl -X DELETE http://localhost:8080/cards/4
```

> Cards are kept in memory: restarting the API resets them to the seeded cards.

## Roadmap

Progress is tracked in [GitHub Issues](https://github.com/mouhibfahem/devops/issues):
CI/CD, Docker, structured logs, metrics, tracing, security scans and Kubernetes
deployment.
