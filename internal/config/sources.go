package config

// DataSourceConfig defines which data sources are enabled
type DataSourceConfig struct {
	GoogleCalendar bool
	Gmail          bool
	GoogleDocs     bool
	Slack          bool
}

// DefaultConfig returns a configuration with all sources enabled
func DefaultConfig() DataSourceConfig {
	return DataSourceConfig{
		GoogleCalendar: true,
		Gmail:          false,
		GoogleDocs:     true,
		Slack:          false,
	}
}

// DisableAll returns a configuration with all sources disabled
func DisableAll() DataSourceConfig {
	return DataSourceConfig{}
}
