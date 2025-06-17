package openai

import "io"

type UserService struct {
	repo OpenAiRepository
}

func NewOpenAiService(repo OpenAiRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateNewChatMessage(threadID, role, content string) error {
	return s.repo.CreateNewChatMessage(threadID, role, content)
}

func (s *UserService) GetMessageThread(threadID string) (*string, error) {
	return s.repo.GetMessageThread(threadID)
}

func (s *UserService) GetStreamMessageThread(threadID string, writer io.Writer, onFullMessageGenerated func(string)) error {
	return s.repo.GetStreamMessageThread(threadID, writer, onFullMessageGenerated)
}

func (s *UserService) CreateNewChatThread() (*string, error) {
	return s.repo.CreateNewChatThread()
}

func (s *UserService) DeleteChatThread(threadID string) error {
	return s.repo.DeleteChatThread(threadID)
}

func (s *UserService) GetMessageThreadWithContent(input string) (*string, error) {
	return s.repo.GetMessageThreadWithContent(input)
}
