package llm

import (
	"net/http"
	"time"
)

// apiClient is used for non-streaming API calls. Times out after 60s.
var apiClient = &http.Client{Timeout: 60 * time.Second}

// streamClient is used for streaming calls where data arrives incrementally.
// No full-response timeout — callers must pass a context with deadline.
var streamClient = &http.Client{Timeout: 0}
