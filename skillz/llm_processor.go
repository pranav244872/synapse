// skillz/llm_processor.go
package skillz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"io"
	"net/http"
	"strings"
)

////////////////////////////////////////////////////////////////////////

// We define our LLM prompts as constants here to keep them organized and easy to modify.

const (
	// skillExtractionPrompt is used to instruct the LLM to find all potential skills in a block of text.
	// It explicitly asks for a raw, non-standardized JSON array.
	skillExtractionPrompt = `
Your goal is to act as an expert CTO and extract a list of underlying, reusable technical capabilities from the text below.

RULES:
1.  Extract specific, granular skills. Prefer multi-word skills that describe a capability over generic single-word terms.
    - GOOD: "React Hooks", "PostgreSQL Performance Tuning", "Docker Multi-stage Builds"
    - BAD: "React", "Database", "Docker"
2.  Do NOT extract phrases that only describe the business problem or a one-time action. The skill must be a reusable technical capability that an engineer would put on a resume.
    - Example Text: "The user needs to fix the login button alignment which is off on Firefox."
    - CORRECT SKILLS: ["CSS Flexbox", "Cross-browser Compatibility"]
    - INCORRECT SKILLS: ["Fixing login button", "Firefox alignment issue"]
3.  Identify programming languages, frameworks, libraries, databases, cloud services, tools, and technical concepts.
4.  Do NOT standardize or normalize aliases (e.g., if you see 'go' and 'golang', extract both).
5.  Return the result as a single, flat JSON array of strings.

Text: """
%s
"""`

	// proficiencyExtractionPrompt is used to estimate a user's proficiency for a given list of skills.
	// It instructs the model to return a JSON object mapping each skill to a specific proficiency level.
	proficiencyExtractionPrompt = `
Given the following resume text and this list of KNOWN skills that were found within it: %s.

Analyze the resume to estimate a proficiency level for each skill. The valid proficiency levels are 'beginner', 'intermediate', and 'expert'.
Base your estimation on the context of projects, frequency of mention, and stated years of experience.
Return a single JSON object that maps each skill from the known list to its estimated proficiency level.

Resume Text: """
%s
"""`
)

////////////////////////////////////////////////////////////////////////

// Anything with CallLLM method can act as a LLMClient
// LLMClient defines an interface for making LLM calls
type LLMClient interface {
	CallLLM(ctx context.Context, prompt string) (string, error)
}

////////////////////////////////////////////////////////////////////////

type GeminiLLMClient struct {
	apiKey string
	client *http.Client
}

////////////////////////////////////////////////////////////////////////

// GeminiResponse defines the structure of the JSON response we expect from the GeminiAPI
// We only map the fields we need to extract the model's text output
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

////////////////////////////////////////////////////////////////////////
// Struct and Constructor
////////////////////////////////////////////////////////////////////////

// LLMProcessor implements the Processor interface using a Large Language Model
// It holds the necessary configuration for making API calls and normalizing results
type LLMProcessor struct {
	aliasMap  map[string]string // The map for normalizing skills
	caser     cases.Caser       // A caser for handling unicode-correct title casing
	llmClient LLMClient
}

// NewLLMProcessor creates a new LLMProcessor using the provided aliasMap and an LLMClient (real or mock).
func NewLLMProcessor(aliasMap map[string]string, llmClient LLMClient) Processor {
	return &LLMProcessor{
		aliasMap: aliasMap,
		// We use cases.Title with english as the base language
		caser:     cases.Title(language.English),
		llmClient: llmClient,
	}
}

// In real code, you'd pass the real Gemini client
/*
llmClient := &GeminiLLMClient{apiKey: "your-key", client: &http.Client{}}
p := NewLLMProcessor(myAliasMap, llmClient)
*/

////////////////////////////////////////////////////////////////////////
// Public Methods (Interface Implementation)
////////////////////////////////////////////////////////////////////////

// ExtractAndNormalize is the public method that orchestrates the two-step process:
// 1. Call the LLM to get a raw list of potential skills
// 2. Normalize that list into a clean, standardized format
func (p *LLMProcessor) ExtractAndNormalize(ctx context.Context, text string) ([]string, error) {
	// 1. Build the specific prompt for this task
	prompt := fmt.Sprintf(skillExtractionPrompt, text)

	// 2. Call the LLM with the prompt.
	llmResponse, err := p.llmClient.CallLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("skill extraction LLM call failed: %w", err)
	}

	// 3. Parse the LLM's string response into a slice of strings.
	var rawSkills []string
	if err := json.Unmarshal([]byte(llmResponse), &rawSkills); err != nil {
		return nil, fmt.Errorf("failed to parse LLM skill output as JSON array: %s", llmResponse)
	}

	// 4. Normalize the raw skills into a clean, canonical format.
	return p.normalize(rawSkills), nil
}

// ExtractProficiencies orchestrates the process of estimating skill levels
func (p *LLMProcessor) ExtractProficiencies(ctx context.Context, text string, knownSkills []string) (map[string]string, error) {
	// 1. Convert the list of known skills into a JSON string to be embedded in the prompt.
	knownSkillsJSON, err := json.Marshal(knownSkills)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal known skills for prompt: %w", err)
	}

	// 2. Build the specific prompt for this task.
	prompt := fmt.Sprintf(proficiencyExtractionPrompt, string(knownSkillsJSON), text)

	// 3. Call the LLM with the prompt.
	llmResponse, err := p.llmClient.CallLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("proficiency extraction LLM call failed: %w", err)
	}

	// 4. Parse the LLM's string response into a map.
	var proficiencies map[string]string
	if err := json.Unmarshal([]byte(llmResponse), &proficiencies); err != nil {
		return nil, fmt.Errorf("failed to parse LLM proficiency output as JSON object: %s", llmResponse)
	}

	// 5. Validate the results to ensure only allowed proficiency values are used.
	p.validateProficiencies(proficiencies)

	return proficiencies, nil
}

////////////////////////////////////////////////////////////////////////
// Private Helper Methods
////////////////////////////////////////////////////////////////////////

// CallLLM implements the LLMClient interface using the Gemini API.
// It takes a prompt, handles the HTTP request/response, and returns the raw text output from the model.
func (g *GeminiLLMClient) CallLLM(ctx context.Context, prompt string) (string, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"
	requestBody := map[string]any{
		"contents": []map[string]any{{"parts": []map[string]string{{"text": prompt}}}},
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-goog-api-key", g.apiKey)

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API returned non-200 status: %s", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp GeminiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal api response: %w", err)
	}

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("unexpected LLM response format: no content found")
	}

	return apiResp.Candidates[0].Content.Parts[0].Text, nil
}

// normalize is a private method that takes a slice of raw stringsand
// converts them into a clean, deduplicated slice of canonical skill names
func (p *LLMProcessor) normalize(rawSkills []string) []string {
	// We use a map[string]struct{} as a Set to automatically handle duplicates
	normalizedSet := make(map[string]struct{})

	for _, raw := range rawSkills {
		// Standardize the lookup key by converting it into lowercase
		lookup := strings.ToLower(raw)

		// Check if the raw skill has a known alias in our map.
		if canonical, ok := p.aliasMap[lookup]; ok {
			// If yes, use the official canonical name
			normalizedSet[canonical] = struct{}{}
		} else {
			// If no, this is a new skill. We apply Title Case as a default
			// formatting rule. We use the 'caser' for Unicode-safe casing
			normalizedSet[p.caser.String(lookup)] = struct{}{}
		}
	}

	// Convert the set(map) back into a slice
	// The order is not guaranteed, but that's fine for this use case
	normalized := make([]string, 0, len(normalizedSet))
	for skill := range normalizedSet {
		normalized = append(normalized, skill)
	}

	return normalized
}

// validateProficiencies ensures that the proficiency levels returned by the LLM
// conform to our application's allowed values ('beginner', 'intermediate', 'expert').
// It modifies the map in-place, changing any invalid value to 'beginner'.
func (p *LLMProcessor) validateProficiencies(proficiencies map[string]string) {
	validLevels := map[string]struct{}{
		"beginner":     {},
		"intermediate": {},
		"expert":       {},
	}

	for skill, level := range proficiencies {
		if _, ok := validLevels[level]; !ok {
			proficiencies[skill] = "beginner" // Default to the safest value if invalid.
		}
	}

}

////////////////////////////////////////////////////////////////////////
