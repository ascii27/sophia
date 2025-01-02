package gmail

import (
	"context"
	"fmt"
	"time"
	
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	
	"github.com/michaelgalloway/sophia/internal/auth"
	"github.com/michaelgalloway/sophia/internal/datasources"
)

type GmailSource struct {
	service   *gmail.Service
	creds     []byte
	tokenDir  string
	tokenMgr  *auth.TokenManager
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
	
	return &GmailSource{
		creds:    []byte(credentials),
		tokenDir: tokenDir,
		tokenMgr: auth.NewTokenManager(tokenDir),
	}, nil
}

func (g *GmailSource) Name() string {
	return "gmail"
}

func (g *GmailSource) Initialize(ctx context.Context) error {
	// Create OAuth2 config from credentials
	config, err := google.ConfigFromJSON(g.creds,
		gmail.GmailReadonlyScope,
	)
	if err != nil {
		return fmt.Errorf("failed to parse client secret file to config: %w", err)
	}

	// Get OAuth2 token
	token, err := g.tokenMgr.GetToken(ctx, config, "gmail")
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Create HTTP client with token
	client := config.Client(ctx, token)

	// Create the Gmail service
	g.service, err = gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create Gmail service: %w", err)
	}
	return nil
}

func (g *GmailSource) FetchData(ctx context.Context, since time.Time) ([]datasources.Document, error) {
	query := fmt.Sprintf("after:%s", since.Format("2006/01/02"))
	
	var docs []datasources.Document
	pageToken := ""
	
	for {
		req := g.service.Users.Messages.List("me").Q(query)
		if pageToken != "" {
			req.PageToken(pageToken)
		}
		
		r, err := req.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch messages: %w", err)
		}
		
		for _, msg := range r.Messages {
			message, err := g.service.Users.Messages.Get("me", msg.Id).Do()
			if err != nil {
				continue
			}
			
			headers := make(map[string]string)
			for _, header := range message.Payload.Headers {
				headers[header.Name] = header.Value
			}
			
			content := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n\n%s",
				headers["From"],
				headers["To"],
				headers["Subject"],
				getMessage(message),
			)
			
			timestamp := time.Unix(message.InternalDate/1000, 0)
			
			doc := datasources.Document{
				ID:      message.Id,
				Content: content,
				Metadata: map[string]interface{}{
					"from":    headers["From"],
					"to":      headers["To"],
					"subject": headers["Subject"],
					"labels":  message.LabelIds,
				},
				Source:    g.Name(),
				Timestamp: timestamp,
			}
			
			docs = append(docs, doc)
		}
		
		pageToken = r.NextPageToken
		if pageToken == "" {
			break
		}
	}
	
	return docs, nil
}

func getMessage(msg *gmail.Message) string {
	if msg.Payload == nil {
		return ""
	}
	
	var text string
	var walk func(*gmail.MessagePart)
	walk = func(part *gmail.MessagePart) {
		if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
			text += part.Body.Data
		}
		
		for _, p := range part.Parts {
			walk(p)
		}
	}
	
	walk(msg.Payload)
	return text
}
