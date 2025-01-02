package slack

import (
	"context"
	"fmt"
	"time"
	
	"github.com/slack-go/slack"
	
	"github.com/michaelgalloway/sophia/internal/datasources"
)

type SlackSource struct {
	client *slack.Client
	config struct {
		token   string
		channels []string
	}
}

func New(config map[string]interface{}) (datasources.DataSource, error) {
	token, ok := config["token"].(string)
	if !ok {
		return nil, fmt.Errorf("token not provided in config")
	}
	
	channels, ok := config["channels"].([]string)
	if !ok {
		return nil, fmt.Errorf("channels not provided in config")
	}
	
	s := &SlackSource{
		client: slack.New(token),
	}
	s.config.token = token
	s.config.channels = channels
	
	return s, nil
}

func (s *SlackSource) Name() string {
	return "slack"
}

func (s *SlackSource) Initialize(ctx context.Context) error {
	_, err := s.client.AuthTest()
	return err
}

func (s *SlackSource) FetchData(ctx context.Context, since time.Time) ([]datasources.Document, error) {
	var docs []datasources.Document
	
	for _, channelName := range s.config.channels {
		channel, err := s.findChannel(channelName)
		if err != nil {
			continue
		}
		
		params := slack.GetConversationHistoryParameters{
			ChannelID: channel.ID,
			Oldest:    fmt.Sprintf("%d", since.Unix()),
			Inclusive: true,
		}
		
		for {
			history, err := s.client.GetConversationHistory(&params)
			if err != nil {
				break
			}
			
			for _, msg := range history.Messages {
				timestamp, err := parseSlackTimestamp(msg.Timestamp)
				if err != nil {
					continue
				}
				
				content := fmt.Sprintf("Channel: %s\nUser: %s\nMessage: %s",
					channelName,
					msg.User,
					msg.Text,
				)
				
				// Get thread replies if they exist
				if msg.ThreadTimestamp != "" {
					var allReplies []slack.Message
					params := &slack.GetConversationRepliesParameters{
						ChannelID: channel.ID,
						Timestamp: msg.ThreadTimestamp,
					}

					for {
						replies, hasMore, nextCursor, err := s.client.GetConversationReplies(params)
						if err != nil {
							break
						}

						allReplies = append(allReplies, replies...)

						if !hasMore {
							break
						}
						params.Cursor = nextCursor
					}

					if len(allReplies) > 0 {
						content += "\n\nThread Replies:\n"
						for _, reply := range allReplies {
							if reply.Timestamp != msg.Timestamp {
								content += fmt.Sprintf("- %s: %s\n", reply.User, reply.Text)
							}
						}
					}
				}
				
				doc := datasources.Document{
					ID:      msg.Timestamp,
					Content: content,
					Metadata: map[string]interface{}{
						"channel":     channelName,
						"user":        msg.User,
						"has_thread":  msg.ThreadTimestamp != "",
						"reactions":   msg.Reactions,
						"message_url": s.createMessageLink(channel.ID, msg.Timestamp),
					},
					Source:    s.Name(),
					Timestamp: timestamp,
				}
				
				docs = append(docs, doc)
			}
			
			if !history.HasMore {
				break
			}
			params.Cursor = history.ResponseMetaData.NextCursor
		}
	}
	
	return docs, nil
}

func (s *SlackSource) findChannel(channelName string) (*slack.Channel, error) {
	channels, _, err := s.client.GetConversations(&slack.GetConversationsParameters{
		ExcludeArchived: true,
		Types:           []string{"public_channel", "private_channel"},
	})
	if err != nil {
		return nil, err
	}
	
	for _, channel := range channels {
		if channel.Name == channelName {
			return &channel, nil
		}
	}
	
	return nil, fmt.Errorf("channel %s not found", channelName)
}

func parseSlackTimestamp(ts string) (time.Time, error) {
	sec := int64(0)
	nsec := int64(0)
	
	_, err := fmt.Sscanf(ts, "%d.%d", &sec, &nsec)
	if err != nil {
		return time.Time{}, err
	}
	
	return time.Unix(sec, nsec), nil
}

func (s *SlackSource) createMessageLink(channelID, timestamp string) string {
	return fmt.Sprintf("https://slack.com/archives/%s/p%s", channelID, timestamp)
}
