# Sophia - Your Personal AI Assistant

Sophia is a modular Go service that helps you manage your digital life by providing intelligent responses to questions about your calendar, emails, documents, and Slack conversations. It uses OpenAI's GPT-4 and embeddings to provide context-aware responses based on your personal data.

## Features

- Hourly synchronization with multiple data sources:
  - Google Calendar
  - Gmail
  - Google Docs
  - Slack
  - Todoist
- Vector-based semantic search using pgvector
- OpenAI GPT-4 integration for intelligent responses
- Modular architecture for easy addition of new data sources
- RESTful API endpoint for queries

## Prerequisites

- Go 1.21 or later
- PostgreSQL with pgvector extension
- OpenAI API key
- Google Cloud Platform credentials
- Slack API token
- Todoist API token

## Environment Variables

Create a `.env` file in the project root with the following variables:

```env
# OpenAI
OPENAI_API_KEY=your_openai_api_key

# Google
GOOGLE_CREDENTIALS=path_to_your_credentials.json

# Slack
SLACK_TOKEN=your_slack_bot_token
SLACK_CHANNELS=general,random,team

# Todoist
TODOIST_API_TOKEN=your_todoist_api_token

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=your_username
POSTGRES_PASSWORD=your_password
POSTGRES_DB=sophia
```

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/sophia.git
cd sophia
```

2. Install dependencies:
```bash
go mod download
```

3. Create the PostgreSQL database and enable pgvector:
```sql
CREATE DATABASE sophia;
\c sophia
CREATE EXTENSION vector;
```

4. Run the service:
```bash
go run cmd/server/main.go
```

## Usage

The service exposes an HTTP endpoint at `http://localhost:8080/ask` that accepts POST requests with a `query` parameter.

Example using curl:
```bash
curl -X POST http://localhost:8080/ask \
  -d "query=What meetings do I have tomorrow?"
```

Example response:
```
Based on your calendar, tomorrow you have:
1. Team Standup at 10:00 AM with the engineering team
2. Client Meeting at 2:00 PM with Acme Corp
3. Weekly Planning at 4:30 PM

The meeting with Acme Corp has related emails discussing the project timeline, and there are several Slack conversations in the #client-projects channel about the deliverables.
```

## Adding New Data Sources

To add a new data source:

1. Create a new package under `internal/datasources`
2. Implement the `DataSource` interface:
```go
type DataSource interface {
    Name() string
    FetchData(ctx context.Context, since time.Time) ([]Document, error)
    Initialize(ctx context.Context) error
}
```
3. Add the new source to `cmd/server/main.go`

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
