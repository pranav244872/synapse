// skillz/llm_processor_test.go
package skillz_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"github.com/pranav244872/synapse/skillz"
)

// A mock is a stand-in for real dependency. Our LLMProcessr needs an LLMClient
// Instead of making real API call, we create a 'mock' LLMClient. This mock
// Lets us pretend what the API would return, giving us control over test conditions
type mockLLMClient struct {
	// We'll store the exact response string we want our mock API to return.
	mockResponse string
	// We'll also store a potential error, so we can test how our code handles API failures.
	mockErr error
}

// CallLLM is the method required by the LLMClient interface. Our mock implements it
// Instead of making real HTTP request, it just returns the predefined response and error
func (m *mockLLMClient) CallLLM(ctx context.Context, prompt string) (string, error) {
	return m.mockResponse, m.mockErr
}

// Helper function to create a map from a slice of strings.
// A test shouldn't fail just because ["Go", "Docker"] is not the same as ["Docker", "Go"].
// By converting both to a map (which acts like a Set), we can compare them accurately.
func stringSliceToMap(s []string) map[string]struct{} {
	m := make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}

////////////////////////////////////////////////////////////////////////
// Test for ExtractAndNormalize
////////////////////////////////////////////////////////////////////////

func TestLLMProcessor_ExtractAndNormalize(t *testing.T) {
	// We define our alias map once, as it will be used in all sub-tests.
	// This map standardizes different ways of writing the same skill.
	testAliasMap := map[string]string{
		"go":         "Go",
		"golang":     "Go",
		"k8s":        "Kubernetes",
		"postgres":   "PostgreSQL",
		"postgresql": "PostgreSQL",
	}

	// This is a "table-driven test". It's a common Go pattern where we define
	// a slice of test cases. This makes it easy to add new scenarios
	// without writing a whole new test function.
	testCases := []struct {
		name         string // A descriptive name for the test case.
		inputText    string // The input text we'll pass to our function.
		mockResponse string // The JSON string our mock LLM will "return".
		mockErr      error  // The error our mock LLM will "return".
		want         []string // The final, normalized slice we expect.
		wantErr      bool   // True if we expect our function to return an error.
	}{
		{
			name:      "Happy Path - Mixed Aliases and New Skills",
			inputText: "I am a developer skilled in golang, k8s, and foobar.",
			// Our mock LLM returns a raw list, including duplicates and non-standard names.
			mockResponse: `["golang", "k8s", "foobar", "go"]`,
			mockErr:      nil,
			// We expect a clean, standardized, and deduplicated list.
			// "golang" and "go" become "Go". "k8s" becomes "Kubernetes".
			// "foobar" is title-cased because it's not in our alias map.
			want:    []string{"Go", "Kubernetes", "Foobar"},
			wantErr: false,
		},
		{
			name:         "Error Case - LLM Call Fails",
			inputText:    "Some text.",
			mockResponse: "", // No response because the API call failed.
			mockErr:      errors.New("API is down"),
			want:         nil, // We expect no skills, just an error.
			wantErr:      true,
		},
		{
			name:         "Error Case - Malformed JSON from LLM",
			inputText:    "Some text.",
			mockResponse: `["unclosed array`, // This is not valid JSON.
			mockErr:      nil,
			want:         nil,
	 		wantErr:      true, // We expect a JSON parsing error.
		},
		{
			name:         "Edge Case - Empty LLM Response",
			inputText:    "No skills mentioned here.",
			mockResponse: `[]`, // LLM found nothing, returns an empty JSON array.
			mockErr:      nil,
			want:         []string{}, // We expect an empty slice, not an error.
			wantErr:      false,
		},
	}

	// We loop through each test case defined above.
	for _, tc := range testCases {
		// t.Run() creates a sub-test. This is great because it isolates each case
		// and gives us more organized test output.
		t.Run(tc.name, func(t *testing.T) {
			// --- ARRANGE ---
			// 1. Create our mock LLMClient with the desired response and error for this case.
			mockClient := &mockLLMClient{
				mockResponse: tc.mockResponse,
				mockErr:      tc.mockErr,
			}

			// 2. Create the LLMProcessor instance we want to test, injecting our mock client.
			// This is called "Dependency Injection".
			p := skillz.NewLLMProcessor(testAliasMap, mockClient)

			// --- ACT ---
			// 3. Call the method we are testing.
			got, err := p.ExtractAndNormalize(context.Background(), tc.inputText)

			// --- ASSERT ---
			// 4. Check if we got an error when we expected one, or vice-versa.
			if (err != nil) != tc.wantErr {
				t.Errorf("ExtractAndNormalize() error = %v, wantErr %v", err, tc.wantErr)
				return // If an error occurred unexpectedly, we stop this sub-test.
			}

			// 5. Compare the actual result ('got') with the expected result ('want').
			// We use our helper to convert slices to maps to ignore element order.
			gotMap := stringSliceToMap(got)
			wantMap := stringSliceToMap(tc.want)
			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("ExtractAndNormalize() got = %v, want %v", got, tc.want)
			}
		})
	}
}

////////////////////////////////////////////////////////////////////////
// Test for ExtractProficiencies
////////////////////////////////////////////////////////////////////////

func TestLLMProcessor_ExtractProficiencies(t *testing.T) {
	// A standard list of skills we pretend were already found in the resume.
	// This list will be passed to the function being tested.
	knownSkills := []string{"Go", "Docker", "PostgreSQL"}

	testCases := []struct {
		name         string
		mockResponse string
		mockErr      error
		want         map[string]string // The final map of skill->proficiency we expect.
		wantErr      bool
	}{
		{
			name: "Happy Path - Valid Proficiencies",
			// LLM returns a valid JSON object with the expected structure.
			mockResponse: `{"Go": "expert", "Docker": "intermediate", "PostgreSQL": "beginner"}`,
			mockErr:      nil,
			want: map[string]string{
				"Go":         "expert",
				"Docker":     "intermediate",
				"PostgreSQL": "beginner",
			},
			wantErr: false,
		},
		{
			name: "Validation Case - Invalid Proficiency Level Returned",
			// Here, the LLM hallucinates a new level "advanced".
			// Our code should gracefully handle this by defaulting it to "beginner".
			mockResponse: `{"Go": "expert", "Docker": "advanced"}`,
			mockErr:      nil,
			want: map[string]string{
				"Go":     "expert",
				"Docker": "beginner", // We expect "advanced" to be corrected to "beginner".
			},
			wantErr: false,
		},
		{
			name:         "Error Case - LLM Call Fails",
			mockResponse: "",
			mockErr:      errors.New("API timeout"),
			want:         nil,
			wantErr:      true,
		},
		{
			name:         "Error Case - Malformed JSON from LLM",
			mockResponse: `{"Go": "expert", }`, // Trailing comma can be invalid for some parsers.
			mockErr:      nil,
			want:         nil,
			wantErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// --- ARRANGE ---
			mockClient := &mockLLMClient{
				mockResponse: tc.mockResponse,
				mockErr:      tc.mockErr,
			}
			// We don't need an alias map for this test, so we can pass nil or an empty map.
			p := skillz.NewLLMProcessor(nil, mockClient)

			// --- ACT ---
			// We use a dummy resume text because the mock client doesn't actually use it.
			// The prompt is built, but our mock intercepts the call before it goes anywhere.
			got, err := p.ExtractProficiencies(context.Background(), "dummy resume text", knownSkills)

			// --- ASSERT ---
			if (err != nil) != tc.wantErr {
				t.Errorf("ExtractProficiencies() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			// reflect.DeepEqual is the standard way to compare complex types like maps and structs in tests.
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ExtractProficiencies() got = %v, want %v", got, tc.want)
			}
		})
	}
}
