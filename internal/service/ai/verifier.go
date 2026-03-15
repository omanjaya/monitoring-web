package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

// Provider endpoints and default models
var providerDefaults = map[string]struct {
	BaseURL      string
	DefaultModel string
}{
	"groq": {
		BaseURL:      "https://api.groq.com/openai/v1/chat/completions",
		DefaultModel: "llama-3.3-70b-versatile",
	},
	"mistral": {
		BaseURL:      "https://api.mistral.ai/v1/chat/completions",
		DefaultModel: "mistral-small-latest",
	},
	"anthropic": {
		BaseURL:      "https://api.anthropic.com/v1/messages",
		DefaultModel: "claude-haiku-4-5-20251001",
	},
}

type Verifier struct {
	cfg        *config.AIConfig
	httpClient *http.Client
}

func NewVerifier(cfg *config.AIConfig) *Verifier {
	return &Verifier{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (v *Verifier) IsEnabled() bool {
	return v.cfg.Enabled && v.cfg.APIKey != ""
}

// Detection represents a single detection to verify
type Detection struct {
	PatternName    string  `json:"pattern_name"`
	Category       string  `json:"category"`
	MatchedContent string  `json:"matched_content"`
	Context        string  `json:"context"`
	URL            string  `json:"url"`
	Confidence     float64 `json:"confidence"`
}

// VerificationResult contains the AI's verdict for each detection
type VerificationResult struct {
	Index           int     `json:"index"`
	IsFalsePositive bool    `json:"is_false_positive"`
	Confidence      float64 `json:"confidence"`
	Reason          string  `json:"reason"`
}

// VerifyDetections sends detections to AI for false positive analysis.
func (v *Verifier) VerifyDetections(ctx context.Context, websiteName, websiteURL string, detections []Detection) ([]VerificationResult, error) {
	if !v.IsEnabled() || len(detections) == 0 {
		return nil, nil
	}

	prompt := v.buildPrompt(websiteName, websiteURL, detections)

	response, err := v.callAPI(ctx, prompt)
	if err != nil {
		logger.Error().Err(err).Str("provider", v.cfg.Provider).Msg("AI verification API call failed")
		return nil, err
	}

	results, err := v.parseResponse(response)
	if err != nil {
		logger.Error().Err(err).Str("response", truncate(response, 500)).Msg("Failed to parse AI verification response")
		return nil, err
	}

	return results, nil
}

func (v *Verifier) buildPrompt(websiteName, websiteURL string, detections []Detection) string {
	var sb strings.Builder

	sb.WriteString("Kamu adalah analis keamanan siber untuk website pemerintah Indonesia.\n")
	sb.WriteString(fmt.Sprintf("Website: %s (%s)\n\n", websiteName, websiteURL))
	sb.WriteString("Analisis deteksi berikut dan tentukan mana yang FALSE POSITIVE (bukan ancaman sebenarnya) dan mana yang TRUE POSITIVE (ancaman nyata).\n\n")
	sb.WriteString("KONTEKS PENTING:\n")
	sb.WriteString("- Ini adalah website pemerintah Provinsi Bali (baliprov.go.id)\n")
	sb.WriteString("- Website pemerintah sering mengandung kata-kata seperti 'transfer', 'deposit', 'bonus', 'game', 'anggaran' dalam konteks resmi/pemerintahan\n")
	sb.WriteString("- Konten CSS/JavaScript (gradient names, class names, variable names) BUKAN indikasi judi\n")
	sb.WriteString("- Berita tentang razia/penertiban judi BUKAN indikasi website disusupi judi\n")
	sb.WriteString("- Perhatikan konteks sekitar matched content untuk menentukan apakah ini konten judi/malware asli atau konten legitimate\n\n")
	sb.WriteString("DETEKSI YANG PERLU DIVERIFIKASI:\n\n")

	for i, d := range detections {
		sb.WriteString(fmt.Sprintf("--- Deteksi #%d ---\n", i))
		sb.WriteString(fmt.Sprintf("Pattern: %s\n", d.PatternName))
		sb.WriteString(fmt.Sprintf("Kategori: %s\n", d.Category))
		sb.WriteString(fmt.Sprintf("Matched: %s\n", d.MatchedContent))
		sb.WriteString(fmt.Sprintf("Context: %s\n", truncate(d.Context, 300)))
		sb.WriteString(fmt.Sprintf("URL: %s\n\n", d.URL))
	}

	sb.WriteString("Berikan jawaban HANYA dalam format JSON array, tanpa teks tambahan apapun.\n")
	sb.WriteString("Setiap elemen:\n")
	sb.WriteString(`- "index": nomor deteksi (0-based)` + "\n")
	sb.WriteString(`- "is_false_positive": true jika false positive, false jika ancaman nyata` + "\n")
	sb.WriteString(`- "confidence": 0.0-1.0 seberapa yakin kamu` + "\n")
	sb.WriteString(`- "reason": penjelasan singkat 1 kalimat` + "\n\n")
	sb.WriteString("Contoh output:\n")
	sb.WriteString(`[{"index":0,"is_false_positive":true,"confidence":0.95,"reason":"Kata 'war' berasal dari CSS gradient warm bukan konten judi"}]`)

	return sb.String()
}

func (v *Verifier) callAPI(ctx context.Context, prompt string) (string, error) {
	provider := v.cfg.Provider
	if provider == "" {
		provider = "groq"
	}

	switch provider {
	case "anthropic":
		return v.callAnthropic(ctx, prompt)
	case "groq", "mistral":
		return v.callOpenAICompatible(ctx, prompt, provider)
	default:
		return "", fmt.Errorf("unsupported AI provider: %s (supported: groq, mistral, anthropic)", provider)
	}
}

// OpenAI-compatible API (used by Groq and Mistral)
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
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
		Type    string `json:"type"`
	} `json:"error"`
}

func (v *Verifier) callOpenAICompatible(ctx context.Context, prompt, provider string) (string, error) {
	defaults, ok := providerDefaults[provider]
	if !ok {
		return "", fmt.Errorf("unknown provider: %s", provider)
	}

	model := v.cfg.Model
	if model == "" {
		model = defaults.DefaultModel
	}

	reqBody := openAIRequest{
		Model: model,
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: "Kamu adalah analis keamanan siber. Jawab HANYA dengan JSON array yang valid, tanpa markdown, tanpa penjelasan tambahan.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   2048,
		Temperature: 0.1, // Low temperature for consistent JSON output
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", defaults.BaseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.cfg.APIKey)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, truncate(string(respBody), 300))
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// Anthropic API (different format)
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (v *Verifier) callAnthropic(ctx context.Context, prompt string) (string, error) {
	model := v.cfg.Model
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}

	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: 2048,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", v.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, truncate(string(respBody), 300))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return apiResp.Content[0].Text, nil
}

func (v *Verifier) parseResponse(response string) ([]VerificationResult, error) {
	// Extract JSON array from response (handle markdown code blocks, extra text)
	jsonStr := response

	// Remove markdown code blocks if present
	jsonStr = strings.TrimSpace(jsonStr)
	if strings.HasPrefix(jsonStr, "```") {
		// Remove opening ```json or ```
		if idx := strings.Index(jsonStr, "\n"); idx != -1 {
			jsonStr = jsonStr[idx+1:]
		}
		// Remove closing ```
		if idx := strings.LastIndex(jsonStr, "```"); idx != -1 {
			jsonStr = jsonStr[:idx]
		}
	}

	// Find the JSON array
	if idx := strings.Index(jsonStr, "["); idx != -1 {
		jsonStr = jsonStr[idx:]
	}
	if idx := strings.LastIndex(jsonStr, "]"); idx != -1 {
		jsonStr = jsonStr[:idx+1]
	}

	jsonStr = strings.TrimSpace(jsonStr)

	var results []VerificationResult
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return results, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
