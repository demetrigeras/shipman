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
}

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
