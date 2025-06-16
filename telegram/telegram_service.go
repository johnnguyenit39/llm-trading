package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// TelegramService handles Telegram bot operations
// @Description Service for sending messages to Telegram channels
type TelegramService struct {
	botToken    string
	channelID   string
	botTokenV1  string
	channelIDV1 string
}

// SendMessageRequest represents the request body for sending a Telegram message
// @Description Request body for sending a Telegram message
type SendMessageRequest struct {
	// Chat ID where the message will be sent
	ChatID string `json:"chat_id" example:"-1001234567890"`
	// Message text to be sent
	Text string `json:"text" example:"Hello from J-AI-Trade!"`
	// Parse mode for the message (HTML, Markdown, etc.)
	ParseMode string `json:"parse_mode,omitempty" example:"HTML"`
}

func NewTelegramService() *TelegramService {
	return &TelegramService{
		botToken:    os.Getenv("J_AI_TRADE_BOT_V1"),
		channelID:   os.Getenv("J_AI_TRADE_BOT_V1_CHAN"),
		botTokenV1:  os.Getenv("J_AI_TRADE_BOT_V1"),
		channelIDV1: os.Getenv("J_AI_TRADE_BOT_V1_CHAN"),
	}
}

func (s *TelegramService) SendMessage(message string) error {
	return s.SendMessageToChannel(s.botToken, s.channelID, message)
}

func (s *TelegramService) SendMessageV1(message string) error {
	return s.SendMessageToChannel(s.botTokenV1, s.channelIDV1, message)
}

func (s *TelegramService) SendMessageToChannel(botToken, channelID, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	reqBody := SendMessageRequest{
		ChatID:    channelID,
		Text:      message,
		ParseMode: "HTML",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned non-200 status code: %d", resp.StatusCode)
	}

	return nil
}
