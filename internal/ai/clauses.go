package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ClauseType string

const (
	ClauseHireRate        ClauseType = "hire_rate"
	ClauseDuration        ClauseType = "duration"
	ClauseDelivery        ClauseType = "delivery"
	ClauseRedelivery      ClauseType = "redelivery"
	ClausePaymentTerms    ClauseType = "payment_terms"
	ClauseLaytime         ClauseType = "laytime"
	ClauseDemurrage       ClauseType = "demurrage"
	ClauseOffHire         ClauseType = "off_hire"
	ClauseArbitration     ClauseType = "arbitration"
	ClauseCargo           ClauseType = "cargo"
	ClauseBunkers         ClauseType = "bunkers"
	ClauseInsurance       ClauseType = "insurance"
	ClauseTermination     ClauseType = "termination"
	ClauseMaintenance     ClauseType = "maintenance"
	ClauseOther           ClauseType = "other"
)

type ExtractedClause struct {
	Type         ClauseType `json:"type"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	Importance   string     `json:"importance"` // high, medium, low
	Summary      string     `json:"summary"`
	KeyPoints    []string   `json:"key_points,omitempty"`
	StartOffset  int        `json:"start_offset,omitempty"`
	EndOffset    int        `json:"end_offset,omitempty"`
}

type AnalysisResult struct {
	Clauses     []ExtractedClause `json:"clauses"`
	Summary     string            `json:"summary"`
	RiskFactors []string          `json:"risk_factors,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
}

type ClauseExtractor interface {
	ExtractClauses(ctx context.Context, documentText string) (*AnalysisResult, error)
	// ExtractTerms pulls specific commercial terms (hire rate, laytime, demurrage, etc.)
	// from a charter party and returns them as a flat JSON string.
	ExtractTerms(ctx context.Context, documentText string) (string, error)
}

const termsSystemPrompt = `You are an expert maritime lawyer specialising in charter party analysis.
Your job is to extract specific commercial terms and return them as a strict JSON object.

CRITICAL RULES:
- Respond with ONLY a valid JSON object. No markdown, no code fences, no explanation, no extra text.
- Numeric fields MUST be plain numbers (e.g. 26000), never strings, never formatted with commas or currency symbols.
- If a value cannot be found, set that field to null.
- hire_rate is the daily hire rate (USD/day or the contract currency per day). Extract just the number, e.g. 26000.
- freight_rate is the freight rate per metric tonne. Extract just the number.
- demurrage_rate and despatch_rate are per-day rates. Extract just the number.
- currency should be the 3-letter ISO code (e.g. "USD", "EUR").`

const termsUserTemplate = `Extract the following commercial terms from this charter party document.
Return ONLY a JSON object with exactly these fields (null where not found):

{
  "vessel_name": "string or null — full vessel name as written",
  "imo_number": "string or null — IMO number digits only",
  "vessel_type": "string or null — e.g. Bulk Carrier, Tanker, Container",
  "dwt": null_or_number — deadweight tonnage as a plain number e.g. 75000,
  "flag_state": "string or null — country of registration",
  "hire_rate": null_or_number — daily hire rate as a plain number e.g. 26000 (NOT "USD 26,000/day"),
  "freight_rate": null_or_number — freight rate per MT as a plain number e.g. 15.50,
  "cargo_type": "string or null — commodity name",
  "cargo_quantity": null_or_number — quantity in metric tonnes as a plain number e.g. 50000,
  "load_port": "string or null — loading port name",
  "discharge_port": "string or null — discharge port name",
  "laytime_allowed_hours": null_or_number — total allowed laytime in HOURS as a plain number e.g. 96,
  "demurrage_rate": null_or_number — demurrage rate per day as a plain number e.g. 25000,
  "despatch_rate": null_or_number — despatch rate per day as a plain number e.g. 12500,
  "currency": "string or null — 3-letter ISO currency code e.g. USD"
}

CHARTER PARTY TEXT:
%s`

type OpenAIExtractor struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func NewOpenAIExtractor(apiKey, model, baseURL string) *OpenAIExtractor {
	if model == "" {
		model = "gpt-4o-mini"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &OpenAIExtractor{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const systemPrompt = `You are an expert maritime lawyer and shipping industry specialist. Your task is to analyze charter party documents and extract key negotiation clauses.

For each clause you identify, provide:
1. The type of clause (hire_rate, duration, delivery, redelivery, payment_terms, laytime, demurrage, off_hire, arbitration, cargo, bunkers, insurance, termination, maintenance, or other)
2. A clear title
3. The exact text content from the document
4. Importance level (high, medium, low) based on typical negotiation priorities
5. A brief summary of what this clause means
6. Key points that parties typically negotiate

Focus on clauses that are:
- Financially significant (rates, payments, penalties)
- Operationally important (delivery/redelivery, maintenance, cargo)
- Risk-related (insurance, arbitration, termination)
- Time-sensitive (duration, laytime, demurrage)

Respond with valid JSON only, no markdown formatting.`

const userPromptTemplate = `Analyze the following charter party document and extract all key negotiation clauses. Return the results as a JSON object with this structure:

{
  "clauses": [
    {
      "type": "hire_rate",
      "title": "Daily Hire Rate",
      "content": "exact text from document",
      "importance": "high",
      "summary": "brief explanation",
      "key_points": ["point 1", "point 2"]
    }
  ],
  "summary": "overall document summary",
  "risk_factors": ["risk 1", "risk 2"],
  "suggestions": ["suggestion for negotiation 1"]
}

DOCUMENT TEXT:
%s`

func (e *OpenAIExtractor) ExtractClauses(ctx context.Context, documentText string) (*AnalysisResult, error) {
	if e.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	if len(documentText) > 100000 {
		documentText = documentText[:100000]
	}

	userPrompt := fmt.Sprintf(userPromptTemplate, documentText)

	reqBody := openAIRequest{
		Model: e.model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1,
		MaxTokens:   4096,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if openAIResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	content := openAIResp.Choices[0].Message.Content
	
	content = cleanJSONResponse(content)

	var result AnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse clause analysis: %w (content: %s)", err, content[:min(500, len(content))])
	}

	return &result, nil
}

func (e *OpenAIExtractor) ExtractTerms(ctx context.Context, documentText string) (string, error) {
	if e.apiKey == "" {
		return "", fmt.Errorf("API key not configured")
	}
	if len(documentText) > 80000 {
		documentText = documentText[:80000]
	}
	reqBody := openAIRequest{
		Model: e.model,
		Messages: []openAIMessage{
			{Role: "system", Content: termsSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(termsUserTemplate, documentText)},
		},
		Temperature: 0.0,
		MaxTokens:   2048,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r openAIResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}
	if r.Error != nil {
		return "", fmt.Errorf("API error: %s", r.Error.Message)
	}
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}
	return cleanJSONResponse(r.Choices[0].Message.Content), nil
}

func cleanJSONResponse(s string) string {
	start := 0
	end := len(s)
	
	for i := 0; i < len(s); i++ {
		if s[i] == '{' {
			start = i
			break
		}
	}
	
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '}' {
			end = i + 1
			break
		}
	}
	
	if start < end {
		return s[start:end]
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ────────────────────────────────────────────────────────────────────────────
// Gemini Extractor (Google AI - Free tier: 15 RPM, 1M tokens/day)
// ────────────────────────────────────────────────────────────────────────────

type GeminiExtractor struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewGeminiExtractor(apiKey, model string) *GeminiExtractor {
	if model == "" {
		model = "gemini-pro" // Original Gemini model
	}
	return &GeminiExtractor{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (e *GeminiExtractor) ExtractClauses(ctx context.Context, documentText string) (*AnalysisResult, error) {
	if e.apiKey == "" {
		return nil, fmt.Errorf("Gemini API key not configured")
	}

	if len(documentText) > 100000 {
		documentText = documentText[:100000]
	}

	// Combine system + user prompt for Gemini (no separate system message)
	fullPrompt := systemPrompt + "\n\n" + fmt.Sprintf(userPromptTemplate, documentText)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: fullPrompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     0.1,
			MaxOutputTokens: 4096,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:generateContent?key=%s", e.model, e.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status first
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Gemini API returned %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, string(body[:min(500, len(body))]))
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Gemini (body: %s)", string(body[:min(500, len(body))]))
	}

	content := geminiResp.Candidates[0].Content.Parts[0].Text
	content = cleanJSONResponse(content)

	var result AnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse clause analysis: %w (content: %s)", err, content[:min(500, len(content))])
	}

	return &result, nil
}

func (e *GeminiExtractor) ExtractTerms(ctx context.Context, documentText string) (string, error) {
	if e.apiKey == "" {
		return "", fmt.Errorf("Gemini API key not configured")
	}
	if len(documentText) > 80000 {
		documentText = documentText[:80000]
	}
	fullPrompt := termsSystemPrompt + "\n\n" + fmt.Sprintf(termsUserTemplate, documentText)
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: fullPrompt}}},
		},
		GenerationConfig: geminiGenerationConfig{Temperature: 0.0, MaxOutputTokens: 2048},
	}
	jsonBody, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:generateContent?key=%s", e.model, e.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Gemini API returned %d: %s", resp.StatusCode, string(body))
	}
	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}
	if geminiResp.Error != nil {
		return "", fmt.Errorf("Gemini error: %s", geminiResp.Error.Message)
	}
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini")
	}
	return cleanJSONResponse(geminiResp.Candidates[0].Content.Parts[0].Text), nil
}
