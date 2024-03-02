# vote-frame-shortener-link

Frame-Shortener is a URL shortening service built in Go, utilizing MongoDB for storage. It allows the creation of short links that redirect to longer URLs within specified allowed domains.

## Configuration

Before running the service, ensure you set the following environment variables:

- `ALLOWED_DOMAINS`: Comma-separated list of domains for which short URLs can be created. Example: `example.com,anotherdomain.com`
- `MONGO_URI`: URI for connecting to the MongoDB instance. Example: `mongodb://localhost:27017`
- `MONGO_DB`: The MongoDB database name where the URL mappings will be stored.

## Running the Service

To run the service, execute the binary or run the Go file directly:

```bash
go run main.go
```

Ensure the environment variables are set before starting the service.

## API Usage

### Adding a New URL

To add a new URL, send a request to `/add/<long-link>`. The service will respond with a JSON containing the short link.

**Example Request:**

```bash
curl http://localhost:8080/add/example.com/my/long/link
```

**Example Response:**

```json
{
  "link": "1a2b3c4d"
}
```

You can then access the short link via `http://localhost:8080/1a2b3c4d`, which will redirect you to the long link you provided.

## Implementation Details

- The service listens on port 8080.
- Short links are generated using UUIDs, truncated to the first 8 characters.
- MongoDB is used to store the mapping between short links and long URLs.
- An index is created on the `shortLink` field in the MongoDB collection for efficient lookup.

Ensure that the MongoDB instance is running and accessible at the URI provided in the `MONGO_URI` environment variable before starting the Frame-Shortener service.