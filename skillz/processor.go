// skillz/processor.go
package skillz

import (
	"context"
)

////////////////////////////////////////////////////////////////////////
// Interface Definition
////////////////////////////////////////////////////////////////////////

// Processor is the public contract for our skill processing logic.
// Using an interface allows us to easily swap out the implementation (e.g., for testing)
// without changing the code that uses it.
type Processor interface {
	// ExtractAndNormalize takes raw text and returns a clean slice of standardized skill strings.
	ExtractAndNormalize(ctx context.Context, text string) ([]string, error)

	// ExtractProficiencies takes raw text and a list of known skills, returning a map
	// of each skill to its estimated proficiency level.
	ExtractProficiencies(ctx context.Context, text string, knownSkills []string) (map[string]string, error)
}
