package gemini

// --- Request types ---

// GenerateContentRequest is the Gemini generateContent/streamGenerateContent request body.
type GenerateContentRequest struct {
	Contents         []Content        `json:"contents"`
	SystemInstruction *SystemContent  `json:"systemInstruction,omitempty"`
	Tools            []GeminiTool     `json:"tools,omitempty"`
	ToolConfig       *ToolConfig      `json:"toolConfig,omitempty"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
}

// Content is a single conversation turn.
type Content struct {
	Role  string `json:"role"` // "user" or "model"
	Parts []Part `json:"parts"`
}

// SystemContent holds the system instruction.
type SystemContent struct {
	Parts []Part `json:"parts"`
}

// Part is a typed element within a Content.
type Part struct {
	// text
	Text string `json:"text,omitempty"`

	// inline image
	InlineData *Blob `json:"inlineData,omitempty"`

	// function call (model output)
	FunctionCall *FunctionCall `json:"functionCall,omitempty"`

	// function response (user input)
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
}

// Blob holds inline binary data (e.g. images).
type Blob struct {
	MIMEType string `json:"mimeType"`
	Data     string `json:"data"` // base64
}

// FunctionCall is a function invocation requested by the model.
type FunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

// FunctionResponse provides the result of a function call.
type FunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

// GeminiTool wraps function declarations for the model.
type GeminiTool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations,omitempty"`
}

// FunctionDeclaration describes a callable function.
type FunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ToolConfig controls how the model selects tools.
type ToolConfig struct {
	FunctionCallingConfig *FunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

// FunctionCallingConfig specifies the tool calling mode.
type FunctionCallingConfig struct {
	Mode                 string   `json:"mode"`                           // "AUTO", "ANY", "NONE"
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"` // used with "ANY"
}

// GenerationConfig controls sampling parameters.
type GenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// --- Response types ---

// GenerateContentResponse is the response from generateContent.
type GenerateContentResponse struct {
	Candidates    []Candidate   `json:"candidates"`
	UsageMetadata UsageMetadata `json:"usageMetadata"`
	ModelVersion  string        `json:"modelVersion"`
}

// Candidate is a single generation candidate.
type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason"`
	Index        int     `json:"index"`
}

// UsageMetadata holds token counts.
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// --- Count tokens ---

// CountTokensRequest is the /v1beta/models/{model}:countTokens request body.
type CountTokensRequest struct {
	Contents []Content `json:"contents"`
}

// CountTokensResponse is the response from countTokens.
type CountTokensResponse struct {
	TotalTokens int `json:"totalTokens"`
}

// --- Models list ---

// ModelsResponse is the response from GET /v1beta/models.
type ModelsResponse struct {
	Models []ModelData `json:"models"`
}

// ModelData describes a single Gemini model.
type ModelData struct {
	Name                       string   `json:"name"` // "models/gemini-1.5-pro"
	DisplayName                string   `json:"displayName"`
	Description                string   `json:"description"`
	InputTokenLimit            int      `json:"inputTokenLimit"`
	OutputTokenLimit           int      `json:"outputTokenLimit"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
}
