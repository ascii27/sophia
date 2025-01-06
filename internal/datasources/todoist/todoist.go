package todoist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/michaelgalloway/sophia/internal/datasources"
)

type TodoistSource struct {
	client *http.Client
	token  string
	filter string
}

type TodoistTask struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	Description string    `json:"description"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Priority    int32     `json:"priority"`
	Due         *DueDate  `json:"due"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
}

type DueDate struct {
	Date     string `json:"date"`
	Datetime string `json:"datetime"`
}

func New(config map[string]interface{}) (datasources.DataSource, error) {
	token, ok := config["token"].(string)
	if !ok {
		return nil, fmt.Errorf("token not provided in config")
	}

	filter, ok := config["filter"].(string)
	if !ok {
		return nil, fmt.Errorf("filter not provided in config")
	}

	return &TodoistSource{
		client: &http.Client{},
		token:  token,
		filter: filter,
	}, nil
}

func (t *TodoistSource) Name() string {
	return "todoist"
}

func (t *TodoistSource) Initialize(ctx context.Context) error {
	return nil
}

func (t *TodoistSource) FetchData(ctx context.Context, since time.Time) ([]datasources.Document, error) {

	taskUrl := fmt.Sprintf("https://api.todoist.com/rest/v2/tasks?filter=%s", url.QueryEscape(t.filter))
	req, err := http.NewRequestWithContext(ctx, "GET", taskUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+t.token)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch tasks: %s: %s", resp.Status, string(body))
	}

	var tasks []TodoistTask
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var docs []datasources.Document
	for _, task := range tasks {
		content := fmt.Sprintf("Task: %s\n", task.Content)
		if task.Description != "" {
			content += fmt.Sprintf("Description: %s\n", task.Description)
		}
		if task.Priority != 0 {
			content += fmt.Sprintf("Priority (higher is more important): %d\n", task.Priority)
		}
		if task.Due != nil {
			if task.Due.Datetime != "" {
				content += fmt.Sprintf("Due: %s\n", task.Due.Datetime)
			} else {
				content += fmt.Sprintf("Due: %s\n", task.Due.Date)
			}
		}
		if task.Due != nil {
			if task.Due.Datetime != "" {
				content += fmt.Sprintf("Due: %s\n", task.Due.Datetime)
			} else {
				content += fmt.Sprintf("Due: %s\n", task.Due.Date)
			}
		}

		doc := datasources.Document{
			ID:      task.ID,
			Content: datasources.TruncateContent(content),
			Title:   task.Content,
			URL:     task.URL,
			Source:  t.Name(),
			Metadata: map[string]interface{}{
				"project_id":   task.ProjectID,
				"project_name": task.ProjectName,
				"due":          task.Due,
			},
			Timestamp: task.CreatedAt,
		}

		docs = append(docs, doc)
	}

	return docs, nil
}
