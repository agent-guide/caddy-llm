package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"

	"github.com/agent-guide/caddy-llm/llm/authmanager/credential"
	"github.com/agent-guide/caddy-llm/llm/authmanager/manager"
	"github.com/agent-guide/caddy-llm/llm/provider"
)

type testProvider struct {
	generateResp *provider.GenerateResponse
	generateErr  error
	streamResp   *schema.StreamReader[*schema.Message]
	streamErr    error

	lastGenerateReq *provider.GenerateRequest
	lastStreamReq   *provider.GenerateRequest
}

func (p *testProvider) Generate(_ context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	p.lastGenerateReq = req
	return p.generateResp, p.generateErr
}

func (p *testProvider) Stream(_ context.Context, req *provider.GenerateRequest) (*schema.StreamReader[*schema.Message], error) {
	p.lastStreamReq = req
	return p.streamResp, p.streamErr
}

func (p *testProvider) ListModels(context.Context) ([]provider.ModelInfo, error) {
	return nil, nil
}

func (p *testProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{Streaming: true}
}

type testStatusError struct {
	msg    string
	status int
}

func (e testStatusError) Error() string   { return e.msg }
func (e testStatusError) StatusCode() int { return e.status }

func TestServeLLMApiMarksOpenAIStreamFailures(t *testing.T) {
	authMgr := manager.NewManager(nil, nil, nil)
	if err := authMgr.Register(context.Background(), &credential.Credential{
		ID:       "cred-openai-1",
		Provider: "openai",
	}); err != nil {
		t.Fatalf("register credential: %v", err)
	}

	handler := NewHandler(authMgr, &testProvider{
		streamErr: testStatusError{msg: "rate limit", status: http.StatusTooManyRequests},
	})

	body, err := json.Marshal(ChatCompletionRequest{
		Model:  "gpt-4o-mini",
		Stream: true,
		Messages: []ChatMessage{{
			Role:    "user",
			Content: "hello",
		}},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	if err := handler.ServeLLMApi(rec, req); err != nil {
		t.Fatalf("ServeLLMApi returned error: %v", err)
	}
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusBadGateway)
	}

	cred := authMgr.Get("cred-openai-1")
	if cred == nil {
		t.Fatal("credential not found after request")
	}
	if !cred.Quota.Exceeded {
		t.Fatal("expected quota state to be marked exceeded")
	}
	if !cred.Unavailable {
		t.Fatal("expected credential to be marked unavailable")
	}
	if cred.Status != credential.StatusError {
		t.Fatalf("expected credential status %q, got %q", credential.StatusError, cred.Status)
	}
	if cred.StatusMessage != "quota exceeded" {
		t.Fatalf("expected status message %q, got %q", "quota exceeded", cred.StatusMessage)
	}
	if cred.NextRetryAfter.IsZero() {
		t.Fatal("expected next retry time to be set")
	}

	modelState := cred.ModelStates["gpt-4o-mini"]
	if modelState == nil {
		t.Fatal("expected model state to be recorded")
	}
	if !modelState.Quota.Exceeded {
		t.Fatal("expected model quota state to be marked exceeded")
	}
	if !modelState.Unavailable {
		t.Fatal("expected model state to be marked unavailable")
	}
	if modelState.StatusMessage != "quota exceeded" {
		t.Fatalf("expected model status message %q, got %q", "quota exceeded", modelState.StatusMessage)
	}
}

func TestServeLLMApiReturnsChatCompletionResponse(t *testing.T) {
	prov := &testProvider{
		generateResp: &provider.GenerateResponse{
			Message: &schema.Message{
				Role:    schema.RoleType("assistant"),
				Content: "hello back",
				ResponseMeta: &schema.ResponseMeta{
					FinishReason: "stop",
					Usage: &schema.TokenUsage{
						PromptTokens:     3,
						CompletionTokens: 5,
						TotalTokens:      8,
					},
				},
			},
		},
	}

	authMgr := manager.NewManager(nil, nil, nil)
	if err := authMgr.Register(context.Background(), &credential.Credential{
		ID:       "cred-openai-2",
		Provider: "openai",
	}); err != nil {
		t.Fatalf("register credential: %v", err)
	}

	handler := NewHandler(authMgr, prov)

	body, err := json.Marshal(ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []ChatMessage{{
			Role:    "user",
			Content: "hello",
		}},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	if err := handler.ServeLLMApi(rec, req); err != nil {
		t.Fatalf("ServeLLMApi returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusOK)
	}

	if prov.lastGenerateReq == nil {
		t.Fatal("expected provider Generate to be called")
	}
	if prov.lastGenerateReq.Model != "gpt-4o-mini" {
		t.Fatalf("unexpected model: got %q want %q", prov.lastGenerateReq.Model, "gpt-4o-mini")
	}
	if len(prov.lastGenerateReq.Messages) != 1 || prov.lastGenerateReq.Messages[0].Content != "hello" {
		t.Fatalf("unexpected generated request messages: %+v", prov.lastGenerateReq.Messages)
	}

	var resp ChatCompletionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Object != "chat.completion" {
		t.Fatalf("unexpected object: got %q want %q", resp.Object, "chat.completion")
	}
	if resp.Model != "gpt-4o-mini" {
		t.Fatalf("unexpected model: got %q want %q", resp.Model, "gpt-4o-mini")
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("unexpected number of choices: got %d want 1", len(resp.Choices))
	}
	if resp.Choices[0].Message.Role != "assistant" {
		t.Fatalf("unexpected role: got %q want %q", resp.Choices[0].Message.Role, "assistant")
	}
	if resp.Choices[0].Message.Content != "hello back" {
		t.Fatalf("unexpected content: got %q want %q", resp.Choices[0].Message.Content, "hello back")
	}
	if resp.Choices[0].FinishReason != "stop" {
		t.Fatalf("unexpected finish reason: got %q want %q", resp.Choices[0].FinishReason, "stop")
	}
	if resp.Usage.PromptTokens != 3 || resp.Usage.CompletionTokens != 5 || resp.Usage.TotalTokens != 8 {
		t.Fatalf("unexpected usage: %+v", resp.Usage)
	}
}

func TestServeLLMApiStreamsOpenAIChunks(t *testing.T) {
	prov := &testProvider{
		streamResp: schema.StreamReaderFromArray([]*schema.Message{{
			Role:    schema.RoleType("assistant"),
			Content: "hello stream",
			ResponseMeta: &schema.ResponseMeta{
				FinishReason: "stop",
			},
		}}),
	}

	authMgr := manager.NewManager(nil, nil, nil)
	if err := authMgr.Register(context.Background(), &credential.Credential{
		ID:       "cred-openai-3",
		Provider: "openai",
	}); err != nil {
		t.Fatalf("register credential: %v", err)
	}

	handler := NewHandler(authMgr, prov)

	body, err := json.Marshal(ChatCompletionRequest{
		Model:  "gpt-4o-mini",
		Stream: true,
		Messages: []ChatMessage{{
			Role:    "user",
			Content: "hello",
		}},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	if err := handler.ServeLLMApi(rec, req); err != nil {
		t.Fatalf("ServeLLMApi returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("unexpected content type: got %q", got)
	}
	if prov.lastStreamReq == nil {
		t.Fatal("expected provider Stream to be called")
	}

	bodyText := rec.Body.String()
	if !strings.Contains(bodyText, "data: [DONE]") {
		t.Fatalf("expected done marker in stream body, got %q", bodyText)
	}

	firstLine := firstDataLine(bodyText)
	if firstLine == "" {
		t.Fatalf("expected at least one SSE data line, got %q", bodyText)
	}

	var chunk chatCompletionChunk
	if err := json.Unmarshal([]byte(firstLine), &chunk); err != nil {
		t.Fatalf("unmarshal stream chunk: %v", err)
	}
	if chunk.Object != "chat.completion.chunk" {
		t.Fatalf("unexpected object: got %q want %q", chunk.Object, "chat.completion.chunk")
	}
	if chunk.Model != "gpt-4o-mini" {
		t.Fatalf("unexpected model: got %q want %q", chunk.Model, "gpt-4o-mini")
	}
	if len(chunk.Choices) != 1 {
		t.Fatalf("unexpected number of choices: got %d want 1", len(chunk.Choices))
	}
	if chunk.Choices[0].Delta.Role != "assistant" {
		t.Fatalf("unexpected role delta: got %q want %q", chunk.Choices[0].Delta.Role, "assistant")
	}
	if chunk.Choices[0].Delta.Content != "hello stream" {
		t.Fatalf("unexpected content delta: got %q want %q", chunk.Choices[0].Delta.Content, "hello stream")
	}
	if chunk.Choices[0].FinishReason != "stop" {
		t.Fatalf("unexpected finish reason: got %q want %q", chunk.Choices[0].FinishReason, "stop")
	}
}

func firstDataLine(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "data: ") && line != "data: [DONE]" {
			return strings.TrimPrefix(line, "data: ")
		}
	}
	return ""
}
