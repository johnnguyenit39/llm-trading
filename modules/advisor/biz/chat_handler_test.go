package biz

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"j_ai_trade/modules/advisor/model"
	adModel "j_ai_trade/modules/agent_decision/model"
)

// ---------- fakes ----------

type fakeBubble struct {
	mu       sync.Mutex
	started  bool
	initial  string
	appends  []string
	finished bool
	replaced string
	startErr error
}

func (b *fakeBubble) Start(_ context.Context, initial string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.started = true
	b.initial = initial
	return b.startErr
}

func (b *fakeBubble) Append(_ context.Context, cumulative string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.appends = append(b.appends, cumulative)
}

func (b *fakeBubble) Finish(_ context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.finished = true
}

func (b *fakeBubble) ReplaceWith(_ context.Context, text string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.replaced = text
}

func (b *fakeBubble) lastAppend() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.appends) == 0 {
		return ""
	}
	return b.appends[len(b.appends)-1]
}

type fakeTransport struct {
	mu        sync.Mutex
	sent      []string
	bubble    *fakeBubble
	sendErr   error
	updatesCh chan IncomingMessage
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{
		bubble:    &fakeBubble{},
		updatesCh: make(chan IncomingMessage),
	}
}

func (t *fakeTransport) Updates() <-chan IncomingMessage { return t.updatesCh }

func (t *fakeTransport) SendMessage(_ context.Context, _ string, text string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sent = append(t.sent, text)
	return t.sendErr
}

func (t *fakeTransport) NewBubble(_ string) MessageBubble { return t.bubble }

func (t *fakeTransport) KeepTyping(_ context.Context, _ string) func() {
	return func() {}
}

func (t *fakeTransport) Name() string { return "fake" }

func (t *fakeTransport) sentTexts() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, len(t.sent))
	copy(out, t.sent)
	return out
}

type fakeLLM struct {
	chunks  []string
	streamErr error
	called  int
	turns   []model.Turn
	mu      sync.Mutex
}

func (l *fakeLLM) Stream(_ context.Context, turns []model.Turn) (<-chan string, <-chan error) {
	l.mu.Lock()
	l.called++
	l.turns = append([]model.Turn(nil), turns...)
	l.mu.Unlock()

	out := make(chan string, len(l.chunks))
	errCh := make(chan error, 1)
	for _, c := range l.chunks {
		out <- c
	}
	close(out)
	if l.streamErr != nil {
		errCh <- l.streamErr
	}
	close(errCh)
	return out, errCh
}

func (l *fakeLLM) Name() string { return "fake-llm" }

type fakeStore struct {
	mu             sync.Mutex
	turns          map[string][]model.Turn
	greeted        map[string]bool
	lastSymbol     map[string]string
	alertsEnabled  map[string]bool
	clearedChats   []string
	loadErr        error
	tryGreetReturn bool
	tryGreetErr    error
	appendCalls    int
	setSymbolErr   error
	setAlertsErr   error
	areAlertsErr   error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		turns:          map[string][]model.Turn{},
		greeted:        map[string]bool{},
		lastSymbol:     map[string]string{},
		alertsEnabled:  map[string]bool{},
		tryGreetReturn: false,
	}
}

func (s *fakeStore) Load(_ context.Context, chatID string) ([]model.Turn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return nil, s.loadErr
	}
	return append([]model.Turn(nil), s.turns[chatID]...), nil
}

func (s *fakeStore) Append(_ context.Context, chatID string, turn model.Turn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appendCalls++
	s.turns[chatID] = append(s.turns[chatID], turn)
	return nil
}

func (s *fakeStore) Clear(_ context.Context, chatID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearedChats = append(s.clearedChats, chatID)
	delete(s.turns, chatID)
	return nil
}

func (s *fakeStore) TryGreet(_ context.Context, _ string) (bool, error) {
	return s.tryGreetReturn, s.tryGreetErr
}

func (s *fakeStore) MarkGreeted(_ context.Context, chatID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.greeted[chatID] = true
	return nil
}

func (s *fakeStore) GetLastSymbol(_ context.Context, chatID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSymbol[chatID], nil
}

func (s *fakeStore) SetLastSymbol(_ context.Context, chatID string, symbol string) error {
	if s.setSymbolErr != nil {
		return s.setSymbolErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSymbol[chatID] = symbol
	return nil
}

func (s *fakeStore) SetAlertsEnabled(_ context.Context, chatID string, enabled bool) error {
	if s.setAlertsErr != nil {
		return s.setAlertsErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertsEnabled[chatID] = enabled
	return nil
}

func (s *fakeStore) AreAlertsEnabled(_ context.Context, chatID string) (bool, error) {
	if s.areAlertsErr != nil {
		return false, s.areAlertsErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.alertsEnabled[chatID]
	if !ok {
		return true, nil // default-true semantics
	}
	return v, nil
}

func (s *fakeStore) ListAlertSubscribers(_ context.Context) ([]string, error) {
	return nil, nil
}

type fakeAnalyzer struct {
	result EnrichmentResult
	err    error
	called int
	mu     sync.Mutex
}

func (a *fakeAnalyzer) MaybeEnrich(_ context.Context, _ string, _ EnrichmentHints) (EnrichmentResult, error) {
	a.mu.Lock()
	a.called++
	a.mu.Unlock()
	return a.result, a.err
}

type fakeDecisionStore struct {
	mu    sync.Mutex
	saved []*adModel.AgentDecision
	err   error
}

func (d *fakeDecisionStore) Save(_ context.Context, row *adModel.AgentDecision) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil {
		return d.err
	}
	d.saved = append(d.saved, row)
	return nil
}

// ---------- helpers ----------

// newTestHandler wires the standard set of fakes. Individual tests
// mutate the returned fakes before calling handleMessage.
func newTestHandler() (*ChatHandler, *fakeTransport, *fakeStore, *fakeLLM) {
	tr := newFakeTransport()
	st := newFakeStore()
	llm := &fakeLLM{}
	h := NewChatHandler(tr, st, llm, NewUserFilter())
	return h, tr, st, llm
}

func msg(chatID, text string) IncomingMessage {
	return IncomingMessage{ChatID: chatID, UserID: "u1", IsDM: true, Text: text}
}

// ---------- tests ----------

func TestChatHandler_EmptyText_NoLLMCall(t *testing.T) {
	h, _, _, llm := newTestHandler()
	h.handleMessage(context.Background(), msg("c1", "   "))
	if llm.called != 0 {
		t.Fatalf("expected no LLM call on empty text, got %d", llm.called)
	}
}

func TestChatHandler_StartCommand_SendsWelcomeAndMarks(t *testing.T) {
	h, tr, st, llm := newTestHandler()
	h.handleMessage(context.Background(), msg("c1", "/start"))

	if llm.called != 0 {
		t.Fatalf("/start should not call LLM, got %d", llm.called)
	}
	sent := tr.sentTexts()
	if len(sent) != 1 || sent[0] != WelcomeMessage {
		t.Fatalf("expected welcome message, got %v", sent)
	}
	if !st.greeted["c1"] {
		t.Fatalf("expected greeted flag for c1")
	}
}

func TestChatHandler_ResetCommand_ClearsAndAcks(t *testing.T) {
	h, tr, st, llm := newTestHandler()
	st.turns["c1"] = []model.Turn{{Role: "user", Content: "hi"}}

	h.handleMessage(context.Background(), msg("c1", "/reset"))

	if llm.called != 0 {
		t.Fatalf("/reset should not call LLM")
	}
	if len(st.clearedChats) != 1 || st.clearedChats[0] != "c1" {
		t.Fatalf("expected Clear(c1), got %v", st.clearedChats)
	}
	sent := tr.sentTexts()
	if len(sent) != 1 || !strings.Contains(sent[0], "Đã xoá ngữ cảnh") {
		t.Fatalf("unexpected reset ack: %v", sent)
	}
}

func TestChatHandler_HelpCommand_DescribesBot(t *testing.T) {
	h, tr, _, _ := newTestHandler()
	h.handleMessage(context.Background(), msg("c1", "/help"))

	sent := tr.sentTexts()
	if len(sent) != 1 {
		t.Fatalf("expected one help message, got %d", len(sent))
	}
	for _, want := range []string{"/start", "/reset", "/analyze", "/alerts"} {
		if !strings.Contains(sent[0], want) {
			t.Fatalf("help missing %q: %s", want, sent[0])
		}
	}
}

func TestChatHandler_AlertsOn_PersistsAndAcks(t *testing.T) {
	h, tr, st, _ := newTestHandler()
	h.handleMessage(context.Background(), msg("c1", "/alerts on"))

	if v, ok := st.alertsEnabled["c1"]; !ok || !v {
		t.Fatalf("expected alertsEnabled[c1]=true, got ok=%v v=%v", ok, v)
	}
	sent := tr.sentTexts()
	if len(sent) != 1 || !strings.Contains(sent[0], "Đã bật") {
		t.Fatalf("unexpected alerts on ack: %v", sent)
	}
}

func TestChatHandler_AlertsOff_PersistsAndAcks(t *testing.T) {
	h, tr, st, _ := newTestHandler()
	st.alertsEnabled["c1"] = true

	h.handleMessage(context.Background(), msg("c1", "/alerts off"))

	if v, ok := st.alertsEnabled["c1"]; !ok || v {
		t.Fatalf("expected alertsEnabled[c1]=false, got ok=%v v=%v", ok, v)
	}
	sent := tr.sentTexts()
	if len(sent) != 1 || !strings.Contains(sent[0], "Đã tắt") {
		t.Fatalf("unexpected alerts off ack: %v", sent)
	}
}

func TestChatHandler_AlertsStatus_DefaultsToOn(t *testing.T) {
	h, tr, _, _ := newTestHandler()
	h.handleMessage(context.Background(), msg("c1", "/alerts"))

	sent := tr.sentTexts()
	if len(sent) != 1 || !strings.Contains(sent[0], "BẬT") {
		t.Fatalf("expected default-on status, got %v", sent)
	}
}

func TestChatHandler_AlertsOn_StorageErrSurfacesUserMessage(t *testing.T) {
	h, tr, st, _ := newTestHandler()
	st.setAlertsErr = errors.New("boom")

	h.handleMessage(context.Background(), msg("c1", "/alerts on"))

	sent := tr.sentTexts()
	if len(sent) != 1 || !strings.Contains(sent[0], "lưu cài đặt không được") {
		t.Fatalf("expected user-facing storage error, got %v", sent)
	}
}

func TestChatHandler_NormalFlow_StreamsAndPersists(t *testing.T) {
	h, tr, st, llm := newTestHandler()
	llm.chunks = []string{"Đợi ", "M5 đóng nến."}

	h.handleMessage(context.Background(), msg("c1", "phân tích XAU"))

	if llm.called != 1 {
		t.Fatalf("expected one LLM call, got %d", llm.called)
	}
	if !tr.bubble.started || !tr.bubble.finished {
		t.Fatalf("expected bubble started+finished: %+v", tr.bubble)
	}
	if got := tr.bubble.lastAppend(); got != "Đợi M5 đóng nến." {
		t.Fatalf("unexpected last append: %q", got)
	}
	// User + assistant turns both persisted.
	if got := len(st.turns["c1"]); got != 2 {
		t.Fatalf("expected 2 persisted turns, got %d", got)
	}
}

func TestChatHandler_GreetsBeforeFirstReply(t *testing.T) {
	h, tr, st, llm := newTestHandler()
	st.tryGreetReturn = true
	llm.chunks = []string{"Hi."}

	h.handleMessage(context.Background(), msg("c1", "hi"))

	sent := tr.sentTexts()
	if len(sent) == 0 || sent[0] != WelcomeMessage {
		t.Fatalf("expected welcome before reply, got %v", sent)
	}
}

func TestChatHandler_AnalyzerEnrichment_AckSentSymbolPinned(t *testing.T) {
	h, tr, st, llm := newTestHandler()
	llm.chunks = []string{"Reply."}
	an := &fakeAnalyzer{result: EnrichmentResult{
		Digest: "[MARKET_DATA]xau[/MARKET_DATA]",
		Ack:    "Đang kiểm tra XAUUSDT...",
		Symbol: "XAUUSDT",
	}}
	h.WithMarketAnalyzer(an)

	h.handleMessage(context.Background(), msg("c1", "phân tích"))

	if an.called != 1 {
		t.Fatalf("expected analyzer called, got %d", an.called)
	}
	sent := tr.sentTexts()
	if len(sent) != 1 || sent[0] != "Đang kiểm tra XAUUSDT..." {
		t.Fatalf("expected ack message, got %v", sent)
	}
	if st.lastSymbol["c1"] != "XAUUSDT" {
		t.Fatalf("expected last symbol pinned, got %q", st.lastSymbol["c1"])
	}
}

func TestChatHandler_AnalyzerError_FallsBackToChat(t *testing.T) {
	h, tr, _, llm := newTestHandler()
	llm.chunks = []string{"ok"}
	h.WithMarketAnalyzer(&fakeAnalyzer{err: errors.New("binance down")})

	h.handleMessage(context.Background(), msg("c1", "xau?"))

	if llm.called != 1 {
		t.Fatalf("expected LLM still called on analyzer error, got %d", llm.called)
	}
	if !tr.bubble.finished {
		t.Fatalf("expected bubble finished")
	}
}

func TestChatHandler_LLMStreamError_NoTokens_ReplacesWithApology(t *testing.T) {
	h, tr, st, llm := newTestHandler()
	llm.streamErr = errors.New("upstream 500")
	// no chunks → empty stream + error

	h.handleMessage(context.Background(), msg("c1", "x"))

	if tr.bubble.replaced == "" || !strings.Contains(tr.bubble.replaced, "trục trặc") {
		t.Fatalf("expected apology in ReplaceWith, got %q", tr.bubble.replaced)
	}
	// Empty assistant reply must NOT poison history.
	if len(st.turns["c1"]) != 0 {
		t.Fatalf("expected no turns persisted on hard fail, got %d", len(st.turns["c1"]))
	}
}

func TestChatHandler_LLMStreamError_AfterTokens_KeepsPartial(t *testing.T) {
	h, tr, st, llm := newTestHandler()
	llm.chunks = []string{"partial reply"}
	llm.streamErr = errors.New("late err")

	h.handleMessage(context.Background(), msg("c1", "x"))

	if tr.bubble.replaced != "" {
		t.Fatalf("should not ReplaceWith when partial content exists, got %q", tr.bubble.replaced)
	}
	if len(st.turns["c1"]) != 2 {
		t.Fatalf("expected user+assistant turns persisted, got %d", len(st.turns["c1"]))
	}
}

func TestChatHandler_DecisionFenceTriggersReplaceWith_AndPersists(t *testing.T) {
	h, tr, st, llm := newTestHandler()
	t.Setenv("ADVISOR_ACCOUNT_USDT", "0") // disable risk-sizing for stable assertions
	t.Setenv("ADVISOR_RISK_PCT", "0.5")
	llm.chunks = []string{
		"Vào BUY XAU.\n\n```json\n",
		`{"action":"BUY","symbol":"XAUUSDT","entry":2345.0,"stop_loss":2342.0,"take_profit":2350.0,"lot":0.05}`,
		"\n```",
	}
	ds := &fakeDecisionStore{}
	h.WithDecisionStore(ds)

	h.handleMessage(context.Background(), msg("c1", "xau?"))

	if tr.bubble.replaced == "" {
		t.Fatalf("expected formatted card via ReplaceWith")
	}
	if !strings.Contains(tr.bubble.replaced, "📋 Lệnh gợi ý") {
		t.Fatalf("formatted card missing header: %q", tr.bubble.replaced)
	}
	if len(ds.saved) != 1 {
		t.Fatalf("expected decision persisted once, got %d", len(ds.saved))
	}
	if got := ds.saved[0]; got.Symbol != "XAUUSDT" || got.Action != "BUY" || got.Entry != 2345.0 {
		t.Fatalf("unexpected saved row: %+v", got)
	}
	// Assistant turn stored should be the formatted card text, not raw JSON.
	turns := st.turns["c1"]
	if len(turns) != 2 {
		t.Fatalf("expected user+assistant turns, got %d", len(turns))
	}
	asst := turns[1].Content
	if strings.Contains(asst, "```json") {
		t.Fatalf("assistant turn should not contain raw fence: %q", asst)
	}
}

func TestChatHandler_PanicInAnalyzerRecovered(t *testing.T) {
	h, _, _, llm := newTestHandler()
	llm.chunks = []string{"x"}
	h.WithMarketAnalyzer(panickyAnalyzer{})

	// Must not crash the test process.
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.handleMessage(context.Background(), msg("c1", "x"))
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler hung after analyzer panic")
	}
}

type panickyAnalyzer struct{}

func (panickyAnalyzer) MaybeEnrich(_ context.Context, _ string, _ EnrichmentHints) (EnrichmentResult, error) {
	panic("synthetic")
}
