// db/store_test.go
package db

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pranav244872/synapse/skillz" // Assuming this is the correct path
	"github.com/pranav244872/synapse/util"
	"github.com/stretchr/testify/require"
)

// mockLLMClient is a simple mock for the LLMClient interface.
// It allows us to control LLM responses during tests without making real API calls.
type mockLLMClient struct {
	// We can set multiple responses and serve them in order to handle multi-call tests.
	mockResponses []string
	mockErrs      []error
	callCount     int
}

// CallLLM returns the next mock response in the queue.
func (m *mockLLMClient) CallLLM(ctx context.Context, prompt string) (string, error) {
	if m.callCount >= len(m.mockResponses) {
		return "", fmt.Errorf("mock LLM client received too many calls")
	}

	response := m.mockResponses[m.callCount]
	var err error
	if m.callCount < len(m.mockErrs) {
		err = m.mockErrs[m.callCount]
	}

	m.callCount++
	return response, err
}

////////////////////////////////////////////////////////////////////////
// TRANSACTION TESTS
////////////////////////////////////////////////////////////////////////

func TestOnboardNewUserWithSkills(t *testing.T) {
	// --- ARRANGE ---
	store := NewStore(testPool)

	// Define our test skills. One is known to exist in the seed data ("Go"),
	// and the other is randomly generated to guarantee it's new.
	preExistingSkill := "Go"
	brandNewSkillName := util.RandomString(12) // e.g., "GjKdfiLpQsWc"

	// The mock LLM will return our predefined skill mix.
	mockClient := &mockLLMClient{
		mockResponses: []string{
			fmt.Sprintf(`["%s", "%s"]`, preExistingSkill, brandNewSkillName),
			fmt.Sprintf(`{"%s": "expert", "%s": "beginner"}`, preExistingSkill, brandNewSkillName),
		},
	}
	skillProcessor := skillz.NewLLMProcessor(map[string]string{}, mockClient)
	team := createRandomTeam(t)
	randomEmail := util.RandomEmail()

	// --- ACT ---
	params := OnboardNewUserTxParams{
		CreateUserParams: CreateUserParams{
			Name:   pgtype.Text{String: "New Hire", Valid: true},
			Email:  randomEmail,
			TeamID: pgtype.Int8{Int64: team.ID, Valid: true},
		},
		ResumeText: fmt.Sprintf("Experienced with %s and learning %s.", preExistingSkill, brandNewSkillName),
	}
	result, err := store.OnboardNewUserWithSkills(context.Background(), params, skillProcessor)

	// --- ASSERT ---
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.Equal(t, randomEmail, result.User.Email)
	require.Len(t, result.UserSkills, 2)

	// Verify the database state directly.
	// First, check that the brand new skill was created and is correctly marked as unverified.
	newlyCreatedSkill, err := testQueries.GetSkillByName(context.Background(), brandNewSkillName)
	require.NoError(t, err)
	require.NotEmpty(t, newlyCreatedSkill)
	require.False(t, newlyCreatedSkill.IsVerified, "A brand new skill should be created as unverified")

	// Now, verify the user's skill links and proficiencies.
	userSkills, err := testQueries.GetSkillsForUser(context.Background(), result.User.ID)
	require.NoError(t, err)
	require.Len(t, userSkills, 2)

	skillsMap := make(map[string]string)
	for _, s := range userSkills {
		skillsMap[s.SkillName] = string(s.Proficiency)
	}
	require.Equal(t, "expert", skillsMap[preExistingSkill])
	require.Equal(t, "beginner", skillsMap[brandNewSkillName])
}

////////////////////////////////////////////////////////////////////////

func TestProcessNewTask(t *testing.T) {
	// --- ARRANGE ---
	store := NewStore(testPool)

	// The mock response needs a specific, pre-existing skill ("Go") and a new one.
	_, _ = testQueries.CreateSkill(context.Background(), CreateSkillParams{SkillName: "Go", IsVerified: true})
	mockClient := &mockLLMClient{
		mockResponses: []string{`["Go", "Terraform Provisioning"]`},
	}
	skillProcessor := skillz.NewLLMProcessor(map[string]string{}, mockClient)
	// Use the helper to create a random project for the task.
	project := createRandomProject(t)

	// --- ACT ---
	params := ProcessNewTaskTxParams{
		CreateTaskParams: CreateTaskParams{
			ProjectID: pgtype.Int8{Int64: project.ID, Valid: true},
			Title:     "Deploy a new service",
			Status:    TaskStatusOpen,
			Priority:  TaskPriorityMedium,
		},
		Description: "Use Go and Terraform Provisioning to deploy a new service.",
	}
	result, err := store.ProcessNewTask(context.Background(), params, skillProcessor)

	// --- ASSERT ---
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.Len(t, result.TaskRequiredSkills, 2)

	// Verify the database state directly.
	newSkill, err := testQueries.GetSkillByName(context.Background(), "Terraform Provisioning")
	require.NoError(t, err)
	require.False(t, newSkill.IsVerified) // The new skill should be unverified.

	linkedSkills, err := testQueries.GetSkillsForTask(context.Background(), result.Task.ID)
	require.NoError(t, err)
	require.Len(t, linkedSkills, 2)
}

////////////////////////////////////////////////////////////////////////

func TestAssignTaskToUser_Concurrent(t *testing.T) {
	store := NewStore(testPool)
	n := 5 // Number of concurrent transactions

	// --- ARRANGE ---
	// Use helper functions to create 5 users and 5 tasks.
	// This makes the setup much cleaner than the previous manual loop.
	var users []User
	var tasks []Task
	project := createRandomProject(t) // Create one project for all tasks

	for i := range n {
		users = append(users, createRandomUser(t))
		// We create tasks manually here to avoid assigning them to a random user via the helper.
		task, err := testQueries.CreateTask(context.Background(), CreateTaskParams{
			ProjectID: pgtype.Int8{Int64: project.ID, Valid: true},
			Title:     fmt.Sprintf("Concurrent Task %d", i),
			Status:    TaskStatusOpen,
			Priority:  TaskPriorityMedium,
		})
		require.NoError(t, err)
		tasks = append(tasks, task)
	}

	// --- ACT ---
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := store.AssignTaskToUser(context.Background(), AssignTaskToUserTxParams{
				TaskID: tasks[i].ID,
				UserID: users[i].ID,
			})
			require.NoError(t, err)
		}(i)
	}
	wg.Wait()

	// --- ASSERT ---
	for i := range n {
		updatedTask, err := testQueries.GetTask(context.Background(), tasks[i].ID)
		require.NoError(t, err)
		require.Equal(t, users[i].ID, updatedTask.AssigneeID.Int64)
		require.Equal(t, TaskStatusInProgress, updatedTask.Status)

		updatedUser, err := testQueries.GetUser(context.Background(), users[i].ID)
		require.NoError(t, err)
		require.Equal(t, AvailabilityStatusBusy, updatedUser.Availability)
	}
}

////////////////////////////////////////////////////////////////////////

func TestCompleteTask(t *testing.T) {
	store := NewStore(testPool)

	// --- ARRANGE ---
	// This test requires a specific pre-existing state (user is busy, task is in-progress).
	// We use the helper to create the user, but then manually update their state.
	user := createRandomUser(t)
	project := createRandomProject(t)

	_, err := testQueries.UpdateUser(context.Background(), UpdateUserParams{
		ID:           user.ID,
		Availability: NullAvailabilityStatus{AvailabilityStatus: "busy", Valid: true},
	})
	require.NoError(t, err)


	task, err := testQueries.CreateTask(context.Background(), CreateTaskParams{
		Title:      "A Task to Complete",
		ProjectID:  pgtype.Int8{Int64: project.ID, Valid: true},
		Status:     TaskStatusInProgress,
		Priority:   TaskPriorityHigh,
		AssigneeID: pgtype.Int8{Int64: user.ID, Valid: true},
	})
	require.NoError(t, err)

	// --- ACT ---
	err = store.CompleteTask(context.Background(), CompleteTaskTxParams{TaskID: task.ID})

	// --- ASSERT ---
	require.NoError(t, err)

	// Verify the database state directly.
	completedTask, err := testQueries.GetTask(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, TaskStatusDone, completedTask.Status)
	require.True(t, completedTask.CompletedAt.Valid)
	require.WithinDuration(t, time.Now(), completedTask.CompletedAt.Time, 5*time.Second)

	freedUser, err := testQueries.GetUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, AvailabilityStatusAvailable, freedUser.Availability)
}
