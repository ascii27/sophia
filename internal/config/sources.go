package config

// DataSourceConfig controls which data sources are enabled
type DataSourceConfig struct {
	GoogleCalendar bool
	Gmail          bool
	GoogleDocs     bool
	Slack          bool
	Todoist        bool
}

// DefaultConfig returns the default configuration
func DefaultConfig() DataSourceConfig {
	return DataSourceConfig{
		GoogleCalendar: true,
		Gmail:          true,
		GoogleDocs:     true,
		Slack:          true,
		Todoist:        true,
	}
}

// DisableAll returns a config with all sources disabled
func DisableAll() DataSourceConfig {
	return DataSourceConfig{
		GoogleCalendar: false,
		Gmail:         false,
		GoogleDocs:    false,
		Slack:         false,
		Todoist:       false,
	}
}
