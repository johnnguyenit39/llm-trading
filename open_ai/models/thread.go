package models

type Thread struct {
	ID            string                 `json:"id"`
	Object        string                 `json:"object"`
	CreatedAt     int64                  `json:"created_at"`
	Metadata      map[string]interface{} `json:"metadata"`
	ToolResources map[string]interface{} `json:"tool_resources"`
}
