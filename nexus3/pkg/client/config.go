package client

// Config is the configuration structure used to instantiate the Nexus client
type Config struct {
	URL                   string  `json:"url"`
	Username              string  `json:"username"`
	Password              string  `json:"password"`
	Insecure              bool    `json:"insecure"`
	Timeout               *int    `json:"timeout,omitempty"`
	ClientCertificatePath *string `json:"client_cert_path,omitempty"`
	ClientKeyPath         *string `json:"client_key_path,omitempty"`
	RootCAPath            *string `json:"root_ca_path,omitempty"`
	// RetryMaxAttempts is the maximum number of retry attempts for read/update operations
	// after create/update to handle eventual consistency. Default: 5
	RetryMaxAttempts *int `json:"retry_max_attempts,omitempty"`
	// RetryWaitMs is the initial wait time in milliseconds between retries.
	// The wait time doubles with each retry (exponential backoff). Default: 500
	RetryWaitMs *int `json:"retry_wait_ms,omitempty"`
}
