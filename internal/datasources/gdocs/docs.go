package gdocs

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/michaelgalloway/sophia/internal/auth"
	"github.com/michaelgalloway/sophia/internal/datasources"
)

type GoogleDocsSource struct {
	docsService  *docs.Service
	driveService *drive.Service
	creds        []byte
	tokenDir     string
	tokenMgr     *auth.TokenManager
}

func New(config map[string]interface{}) (datasources.DataSource, error) {
	credentials, ok := config["credentials"].(string)
	if !ok {
		return nil, fmt.Errorf("credentials not provided in config")
	}

	tokenDir, ok := config["token_dir"].(string)
	if !ok {
		return nil, fmt.Errorf("token_dir not provided in config")
	}

	return &GoogleDocsSource{
		creds:    []byte(credentials),
		tokenDir: tokenDir,
		tokenMgr: auth.NewTokenManager(tokenDir),
	}, nil
}

func (g *GoogleDocsSource) Name() string {
	return "google_docs"
}

func (g *GoogleDocsSource) Initialize(ctx context.Context) error {
	// Create OAuth2 config from credentials
	config, err := google.ConfigFromJSON(g.creds,
		docs.DriveReadonlyScope,
		drive.DriveReadonlyScope,
	)
	if err != nil {
		return fmt.Errorf("failed to parse client secret file to config: %w", err)
	}

	// Get OAuth2 token
	token, err := g.tokenMgr.GetToken(ctx, config, "docs")
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Create HTTP client with token
	client := config.Client(ctx, token)

	// Create the Docs service
	g.docsService, err = docs.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create Docs service: %w", err)
	}

	// Create the Drive service
	g.driveService, err = drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create Drive service: %w", err)
	}

	return nil
}

func (g *GoogleDocsSource) FetchData(ctx context.Context, since time.Time) ([]datasources.Document, error) {
	//query := fmt.Sprintf("mimeType='application/vnd.google-apps.document' and modifiedTime > '%s'", since.Format(time.RFC3339))

	sinceTime := time.Date(2024, time.June, 10, 23, 0, 0, 0, time.UTC).Format(time.RFC3339)
	query := fmt.Sprintf("mimeType='application/vnd.google-apps.document' and modifiedTime > '%s'", sinceTime)

	var docs []datasources.Document
	pageToken := ""

	for {
		fileList, err := g.driveService.Files.List().
			Q(query).
			Fields("files(id, name, modifiedTime, owners)").
			PageToken(pageToken).
			Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list documents: %w", err)
		}

		for _, file := range fileList.Files {
			doc, err := g.docsService.Documents.Get(file.Id).Do()
			if err != nil {
				continue
			}

			content := extractContent(doc)
			modTime, _ := time.Parse(time.RFC3339, file.ModifiedTime)

			document := datasources.Document{
				ID:        file.Id,
				Content:   datasources.TruncateContent(content),
				Title:     file.Name,
				URL:       fmt.Sprintf("https://docs.google.com/document/d/%s", file.Id),
				Source:    g.Name(),
				Timestamp: modTime,
				Metadata: map[string]interface{}{
					"owners": file.Owners,
				},
			}

			docs = append(docs, document)
		}

		pageToken = fileList.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return docs, nil
}

func extractContent(doc *docs.Document) string {
	var content string

	for _, elem := range doc.Body.Content {
		if elem.Paragraph != nil {
			for _, pe := range elem.Paragraph.Elements {
				if pe.TextRun != nil {
					content += pe.TextRun.Content
				}
			}
		}
	}

	return content
}
