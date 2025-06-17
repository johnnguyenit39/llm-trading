package models

type ThreadRunResponse struct {
	ID                string      `json:"id"`
	Object            string      `json:"object"`
	CreatedAt         int         `json:"created_at"`
	AssistantID       string      `json:"assistant_id"`
	ThreadID          string      `json:"thread_id"`
	RunID             string      `json:"run_id"`
	Status            string      `json:"status"`
	IncompleteDetails interface{} `json:"incomplete_details"`
	IncompleteAt      interface{} `json:"incomplete_at"`
	CompletedAt       int         `json:"completed_at"`
	Role              string      `json:"role"`
	Content           []struct {
		Type string `json:"type"`
		Text struct {
			Value       string        `json:"value"`
			Annotations []interface{} `json:"annotations"`
		} `json:"text"`
	} `json:"content"`
	Attachments []interface{} `json:"attachments"`
	Metadata    struct {
	} `json:"metadata"`
}
