package util

import (
	"math/rand"
	"fmt"
	"strings"
)

const alpha = "abcdefghjklmnopqrstuvwxyz"

// RandomInt generates a random integer between min and max
func RandomInt(min, max int64) int64 {
    if max < min {
        min, max = max, min // swap if needed
    }
    return rand.Int63n(max-min+1) + min
}

// RandomString generates a random string of length n
func RandomString(n int) string {
	var sb strings.Builder	
	k := len(alpha)

	for range n {
		c := alpha[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// RandomName generates a random name which can be used for anything
func RandomName() string {
	return RandomString(6)
}

// RandomEmail generates a random email
func RandomEmail() string {
	return RandomString(7) + "@" + RandomString(6) + ".com"
}

// RandomAvailability generates a random string between "available" and "busy"
func RandomAvailability() string {
	options := []string{"available", "busy"}
	return options[rand.Intn(len(options))]
}

// RandomProficiency returns a random proficiency level: "beginner", "intermediate", or "expert"
func RandomProficiency() string {
	options := []string{"beginner", "intermediate", "expert"}
	return options[rand.Intn(len(options))]
}

// RandomProjectName generates a realistic, tech-sounding project name like "Operation NeuralFlux"
func RandomProjectName() string {
	prefixes := []string{
		"Operation", "Project", "Task", "Mission", "Deployment", "System", "Service",
	}

	techWords := []string{
		"Neural", "Quantum", "Matrix", "Fusion", "Cloud", "Data", "Crypto", "Binary",
		"AI", "Stream", "Graph", "Kernel", "Vector", "Signal", "Code", "API", "Node",
	}

	suffixWords := []string{
		"Core", "Flux", "Engine", "Nexus", "Grid", "Sync", "Hub", "Forge", "Stack", "Pipeline",
	}

	prefix := prefixes[rand.Intn(len(prefixes))]
	word1 := techWords[rand.Intn(len(techWords))]
	word2 := suffixWords[rand.Intn(len(suffixWords))]

	return prefix + " " + word1 + word2
}

// RandomTaskTitle returns a random, realistic dev task title like "Fix login bug" or "Refactor auth middleware"
func RandomTaskTitle() string {
	actions := []string{
		"Fix", "Update", "Refactor", "Improve", "Add", "Remove", "Optimize", "Debug", "Test", "Document",
	}

	targets := []string{
		"login bug", "authentication flow", "database queries", "API endpoints", "UI components",
		"error handling", "deployment scripts", "logging system", "performance issues", "configuration files",
	}

	action := actions[rand.Intn(len(actions))]
	target := targets[rand.Intn(len(targets))]

	return action + " " + target
}

// RandomTaskDescription returns a tech-sounding task description
func RandomTaskDescription() string {
	verbs := []string{
		"Develop", "Build", "Design", "Deploy", "Implement", "Launch", "Create", "Optimize", "Integrate", "Automate",
	}

	adjectives := []string{
		"scalable", "intelligent", "distributed", "cloud-native", "high-performance", "real-time", "modular", "secure",
	}

	nouns := []string{
		"data pipeline", "microservice", "analytics engine", "API gateway", "AI model", "monitoring system", "CI/CD workflow",
		"container platform", "task scheduler", "recommendation engine",
	}

	return fmt.Sprintf("%s a %s %s.",
		verbs[rand.Intn(len(verbs))],
		adjectives[rand.Intn(len(adjectives))],
		nouns[rand.Intn(len(nouns))],
	)
}

// RandomStatus returns a random task status: "open", "in_progress", or "done"
func RandomStatus() string {
	statuses := []string{"open", "in_progress", "done"}
	return statuses[rand.Intn(len(statuses))]
}

// RandomPriority returns a random priority level: "low", "medium", "high", or "critical"
func RandomPriority() string {
	priorities := []string{"low", "medium", "high", "critical"}
	return priorities[rand.Intn(len(priorities))]
}

