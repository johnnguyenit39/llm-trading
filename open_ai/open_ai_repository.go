package openai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"j_ai_trade/common"
	httpRequest "j_ai_trade/http_request"
	models "j_ai_trade/open_ai/models"
	"net/http"
	"os"
	"strings"
)

type OpenAiRepository interface {
	GetMessageThread(threadID string) (*string, error)
	GetStreamMessageThread(threadID string, writer io.Writer, onFullMessageGenerated func(string)) error
	CreateNewChatThread() (*string, error)
	CreateNewChatMessage(threadID, role, content string) error
	DeleteChatThread(threadID string) error
	GetMessageThreadWithContent(input string) (*string, error)
}

func NewOpenAiRepository() *openAiRepository {
	return &openAiRepository{}
}

type openAiRepository struct{}

func (r *openAiRepository) CreateNewChatMessage(threadID, role, content string) error {
	if threadID == "" {
		return common.ErrorSimpleMessage("thread id is empty.")
	}
	//Http service
	httpRequestRepository := httpRequest.NewHttpRepository()
	httpRequestService := httpRequest.NewHttpService(httpRequestRepository)
	openAiKey := os.Getenv("OPEN_AI_KEY")
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", openAiKey),
		"OpenAI-Beta":   "assistants=v2",
	}

	// Create request body
	bodyMap := map[string]interface{}{
		"role":    role,
		"content": content, // Enable streaming
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadID)
	_, err = httpRequestService.Post(url, body, headers)
	if err != nil {
		return err
	}

	return nil
}

func (r *openAiRepository) GetMessageThreadWithContent(input string) (*string, error) {
	// Http service
	httpRequestRepository := httpRequest.NewHttpRepository()
	httpRequestService := httpRequest.NewHttpService(httpRequestRepository)
	openAiKey := os.Getenv("OPEN_AI_KEY")
	assistantID := os.Getenv("FINANCIAL_ASSISTANT_ID")

	// Create request body
	bodyMap := map[string]interface{}{
		"assistant_id": assistantID,
		"thread": map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": input},
			},
		},
		"stream": true, // Enable streaming
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", openAiKey),
		"OpenAI-Beta":   "assistants=v2",
	}

	url := fmt.Sprintf("https://api.openai.com/v1/threads/runs")

	// Start request
	response, err := httpRequestService.Post(url, body, headers)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// Read streaming response
	reader := bufio.NewReader(response.Body)
	var latestMessage string
	var captureNextJSON bool // Flag to start capturing JSON after event is detected

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break // End of stream
			}
			return nil, err
		}

		line = strings.TrimSpace(line) // Remove unnecessary spaces

		// Check if the event is "thread.message.completed"
		if strings.HasPrefix(line, "event: thread.message.completed") {
			captureNextJSON = true // Next JSON message contains the data we need
			continue
		}

		// If the next line is the data after the event
		if captureNextJSON && strings.HasPrefix(line, "data:") {
			// Extract the JSON part after "data: "
			jsonData := strings.TrimPrefix(line, "data: ")

			// Parse JSON response
			var eventData struct {
				Content []struct {
					Type string `json:"type"`
					Text struct {
						Value string `json:"value"`
					} `json:"text"`
				} `json:"content"`
			}

			if err := json.Unmarshal([]byte(jsonData), &eventData); err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Extract the actual message text
			if len(eventData.Content) > 0 {
				latestMessage = eventData.Content[0].Text.Value
			}

			break // Exit loop after capturing the latest message
		}
	}

	if latestMessage == "" {
		return nil, fmt.Errorf("no message found in the stream")
	}

	return &latestMessage, nil
}

func (r *openAiRepository) GetMessageThread(threadID string) (*string, error) {
	// Http service
	httpRequestRepository := httpRequest.NewHttpRepository()
	httpRequestService := httpRequest.NewHttpService(httpRequestRepository)
	openAiKey := os.Getenv("OPEN_AI_KEY")
	assistantID := os.Getenv("FINANCIAL_ASSISTANT_ID")
	// Create request body
	bodyMap := map[string]interface{}{
		"assistant_id": assistantID,
		"stream":       true, // Enable streaming
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", openAiKey),
		"OpenAI-Beta":   "assistants=v2",
	}

	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs", threadID)

	// Start request
	response, err := httpRequestService.Post(url, body, headers)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// Read streaming response
	reader := bufio.NewReader(response.Body)
	var latestMessage string
	var captureNextJSON bool // Flag to start capturing JSON after event is detected

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break // End of stream
			}
			return nil, err
		}

		line = strings.TrimSpace(line) // Remove unnecessary spaces

		// Check if the event is "thread.message.completed"
		if strings.HasPrefix(line, "event: thread.message.completed") {
			captureNextJSON = true // Next JSON message contains the data we need
			continue
		}

		// If the next line is the data after the event
		if captureNextJSON && strings.HasPrefix(line, "data:") {
			// Extract the JSON part after "data: "
			jsonData := strings.TrimPrefix(line, "data: ")

			// Parse JSON response
			var eventData struct {
				Content []struct {
					Type string `json:"type"`
					Text struct {
						Value string `json:"value"`
					} `json:"text"`
				} `json:"content"`
			}

			if err := json.Unmarshal([]byte(jsonData), &eventData); err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Extract the actual message text
			if len(eventData.Content) > 0 {
				latestMessage = eventData.Content[0].Text.Value
			}

			break // Exit loop after capturing the latest message
		}
	}

	if latestMessage == "" {
		return nil, fmt.Errorf("no message found in the stream")
	}

	return &latestMessage, nil
}
func (r *openAiRepository) GetStreamMessageThread(threadID string, writer io.Writer, onFullMessageGenerated func(string)) error {
	// Initialize HTTP service
	httpRequestRepository := httpRequest.NewHttpRepository()
	httpRequestService := httpRequest.NewHttpService(httpRequestRepository)
	openAiKey := os.Getenv("OPEN_AI_KEY")
	assistantID := os.Getenv("FINANCIAL_ASSISTANT_ID")
	// Create request body
	bodyMap := map[string]interface{}{
		"assistant_id": assistantID,
		"stream":       true, // Enable streaming
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return err
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", openAiKey),
		"OpenAI-Beta":   "assistants=v2",
	}

	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs", threadID)

	// Start the request
	response, err := httpRequestService.Post(url, body, headers)
	if err != nil {
		return err
	}
	defer response.Body.Close() // Ensure response is closed

	// Read streaming response
	reader := bufio.NewReader(response.Body)
	var captureNextJSON bool // Flag to start capturing JSON after event is detected

	for {
		// Read every line
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break // OpenAI stopped responding
			}
			return err
		}

		line = strings.TrimSpace(line) // Clean line

		// Send **every** message without filtering
		fmt.Fprintf(writer, "%s\n", line)

		// Flush to send data immediately
		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}

		// Check if the event is "thread.message.completed"
		if strings.HasPrefix(line, "event: thread.message.completed") {
			captureNextJSON = true // Next JSON message contains the data we need
			continue
		}

		// If the next line is the data after the event
		if captureNextJSON && strings.HasPrefix(line, "data:") {
			// Extract the JSON part after "data: "
			jsonData := strings.TrimPrefix(line, "data: ")

			// Parse JSON response
			var eventData struct {
				Content []struct {
					Type string `json:"type"`
					Text struct {
						Value string `json:"value"`
					} `json:"text"`
				} `json:"content"`
			}

			if err := json.Unmarshal([]byte(jsonData), &eventData); err != nil {
				return nil
			}

			// Extract the actual message text
			if len(eventData.Content) > 0 {
				onFullMessageGenerated(eventData.Content[0].Text.Value)
				r.CreateNewChatMessage(threadID, "assistant", eventData.Content[0].Text.Value)
			}

			break // Exit loop after capturing the latest message
		}
	}

	// ✅ Ensure writer is properly closed (if closable)
	if closer, ok := writer.(io.Closer); ok {
		closer.Close()
	}

	return nil
}

// CreateNewChatThread implements OpenAiRepository.
func (r *openAiRepository) CreateNewChatThread() (*string, error) {
	//Http service
	httpRequestRepository := httpRequest.NewHttpRepository()
	httpRequestService := httpRequest.NewHttpService(httpRequestRepository)
	openAiKey := os.Getenv("OPEN_AI_KEY")
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", openAiKey),
		"OpenAI-Beta":   "assistants=v2",
	}

	response, err := httpRequestService.Post("https://api.openai.com/v1/threads", nil, headers)
	if err != nil {
		return nil, err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var thread models.Thread
	err = json.Unmarshal([]byte(responseBody), &thread)
	if err != nil {
		return nil, err
	}

	return &thread.ID, nil
}

func (r *openAiRepository) DeleteChatThread(threadID string) error {
	if threadID == "" {
		return common.ErrorSimpleMessage("thread id is empty.")
	}
	//Http service
	httpRequestRepository := httpRequest.NewHttpRepository()
	httpRequestService := httpRequest.NewHttpService(httpRequestRepository)
	openAiKey := os.Getenv("OPEN_AI_KEY")
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", openAiKey),
		"OpenAI-Beta":   "assistants=v2",
	}
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s", threadID)
	response, err := httpRequestService.Delete(url, nil, headers)
	if err != nil {
		return err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	var thread models.Thread
	err = json.Unmarshal([]byte(responseBody), &thread)
	if err != nil {
		return err
	}

	return nil
}
