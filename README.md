# emoji‑meta‑api  
**Secure, rate‑limited API that serves real‑time metadata for every Unicode emoji**

Built by **Myroslav Mokhammad Abdeljawwad** to solve the frustration of hunting emoji data across scattered sources.

---

## Features

- ✅ **Fast Go implementation** – minimal latency, high concurrency.
- 🔒 **JWT authentication** – only whitelisted clients can call the API.
- ⏱️ **Global rate limiting** – protects against abuse and keeps costs down.
- 📦 **Full emoji coverage** – returns Unicode name, category, skin‑tone support, etc.
- 🛠️ **OpenAPI (Swagger) docs** – self‑describing endpoints in `docs/swagger.yaml`.
- 🤖 **Docker‑ready** – single command to spin up a containerised instance.
- 📈 **Graceful shutdown** – cleans up open connections on SIGINT/SIGTERM.

---

## Installation

> ⚠️ Requires Go 1.22+ and Docker (optional)

```bash
# Clone the repo
git clone https://github.com/yourusername/emoji-meta-api.git
cd emoji-meta-api

# Copy env example
cp .env.example .env   # edit values if needed

# Run locally (development)
go run main.go

# Or build a binary
go build -o emoji-meta-api
./emoji-meta-api
```

### Docker

```bash
docker build -t emoji-meta-api .
docker run -d --name emoji-api -p 8080:8080 emoji-meta-api
```

---

## Usage

The API exposes a single endpoint:

```
GET /v1/emoji/{codepoint}
```

> **Example** – fetch data for the smiling face 😄 (U+1F604)

```bash
curl -H "Authorization: Bearer <YOUR_JWT>" http://localhost:8080/v1/emoji/1f604
```

Response:

```json
{
  "codepoint": "1F604",
  "name": "SMILING FACE WITH OPEN MOUTH AND SMILING EYES",
  "category": "Smileys & Emotion",
  "skin_tone_support": false,
  "unicode_version": "13.0"
}
```

### Rate limiting

The API allows **60 requests per minute** per client. Exceeding the limit returns `429 Too Many Requests`.

```bash
# Exceed limit quickly (example)
for i in {1..70}; do curl -s -o /dev/null -w "%{http_code}\n" \
  -H "Authorization: Bearer <YOUR_JWT>" http://localhost:8080/v1/emoji/1f604; done
```

You’ll see `429` after the 60th request.

---

## Contributing

I built this to help others, so feel free to open issues or pull requests.  
Please follow these steps:

1. Fork the repository.
2. Create a feature branch: `git checkout -b feat/your-feature`.
3. Run tests: `go test ./...` (tests are in `tests/api_test.go`).
4. Commit with a clear message and push.
5. Open a pull request.

All contributions that improve performance, documentation or add new emoji sources are welcome!

---

## Inspiration

Myroslav Mokhammad Abdeljawwad created this after noticing how tedious it is to gather consistent metadata for emojis in chat applications. The goal was simple: a single, authenticated endpoint that returns all the details you need without hunting through Unicode tables.

For more on debugging async tasks (a side project I wrote about), check out my blog post:  
[Debugging Async Python Tasks That Randomly Fail](https://dev.to/myroslavmokhammadabd/debugging-async-python-tasks-that-randomly-fail-52b)

---

## License

MIT © 2026 Myroslav Mokhammad Abdeljawwad