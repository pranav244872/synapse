package db

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pranav244872/synapse/util"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------------
//                                 TRANSACTION TESTS
// ---------------------------------------------------------------------------------

func TestOnboardNewUserWithSkills(t *testing.T) {
	store := NewStore(testPool)

	// Test Case 1: Creates a user and links a mix of existing and new skills with proper proficiencies.
	t.Run("Happy Path - Mix of New and Existing Skills", func(t *testing.T) {
		preExistingSkill := createRandomSkill(t)         // Skill that already exists in DB
		brandNewSkillName := util.RandomString(12)       // Brand new skill
		team := createRandomTeam(t)
		randomEmail := util.RandomEmail()

		skillsToCreate := map[string]ProficiencyLevel{
			preExistingSkill.SkillName: ProficiencyLevelExpert,
			brandNewSkillName:          ProficiencyLevelBeginner,
		}

		params := OnboardNewUserTxParams{
			CreateUserParams: CreateUserParams{
				Name:         pgtype.Text{String: "Test Hire", Valid: true},
				Email:        randomEmail,
				PasswordHash: "hashed_password",
				TeamID:       pgtype.Int8{Int64: team.ID, Valid: true},
				Role:         UserRoleEngineer,
			},
			SkillsWithProficiency: skillsToCreate,
		}

		result, err := store.OnboardNewUserWithSkills(context.Background(), params)

		require.NoError(t, err)
		require.NotEmpty(t, result)
		require.Equal(t, randomEmail, result.User.Email)
		require.Len(t, result.UserSkills, 2)

		// Confirm that the new skill was created and marked as unverified
		newlyCreatedSkill, err := testQueries.GetSkillByName(context.Background(), brandNewSkillName)
		require.NoError(t, err)
		require.False(t, newlyCreatedSkill.IsVerified)

		// Confirm correct skill-to-user linkage and proficiency levels
		userSkills, err := testQueries.GetSkillsForUser(context.Background(), result.User.ID)
		require.NoError(t, err)
		require.Len(t, userSkills, 2)

		skillsMap := make(map[string]string)
		for _, s := range userSkills {
			skillsMap[s.SkillName] = string(s.Proficiency)
		}
		require.Equal(t, "expert", skillsMap[preExistingSkill.SkillName])
		require.Equal(t, "beginner", skillsMap[brandNewSkillName])
	})

	// Test Case 2: Creates a user even when no skills are provided.
	t.Run("Edge Case - No Skills Provided", func(t *testing.T) {
		team := createRandomTeam(t)
		randomEmail := util.RandomEmail()

		params := OnboardNewUserTxParams{
			CreateUserParams: CreateUserParams{
				Name:         pgtype.Text{String: "Non-Technical Hire", Valid: true},
				Email:        randomEmail,
				PasswordHash: "hashed_password",
				TeamID:       pgtype.Int8{Int64: team.ID, Valid: true},
				Role:         UserRoleManager,
			},
			SkillsWithProficiency: make(map[string]ProficiencyLevel),
		}

		result, err := store.OnboardNewUserWithSkills(context.Background(), params)

		require.NoError(t, err)
		require.NotEmpty(t, result.User)
		require.Empty(t, result.UserSkills)

		dbUser, err := testQueries.GetUserByEmail(context.Background(), params.CreateUserParams.Email)
		require.NoError(t, err)
		require.Equal(t, params.CreateUserParams.Email, dbUser.Email)
	})

	// Test Case 3: Ensures transaction is rolled back if user creation fails (e.g., duplicate email).
	t.Run("Failure - Duplicate Email Rolls Back Transaction", func(t *testing.T) {
		existingUser, _ := createRandomUser(t)

		skillsToCreate := map[string]ProficiencyLevel{
			"Go": ProficiencyLevelExpert,
		}

		params := OnboardNewUserTxParams{
			CreateUserParams: CreateUserParams{
				Name:         pgtype.Text{String: "Duplicate Hire", Valid: true},
				Email:        existingUser.Email, // Email conflict
				PasswordHash: "hashed_password",
				TeamID:       pgtype.Int8{Int64: existingUser.TeamID.Int64, Valid: true},
				Role:         UserRoleEngineer,
			},
			SkillsWithProficiency: skillsToCreate,
		}

		_, err := store.OnboardNewUserWithSkills(context.Background(), params)
		require.Error(t, err, "Transaction should fail due to unique constraint on email")
	})
}


func TestProcessNewTask(t *testing.T) {
	store := NewStore(testPool)

	t.Run("Happy Path - Creates Task and Links Skills", func(t *testing.T) {
		// --- ARRANGE ---
		project := createRandomProject(t)

		// We define the exact, clean list of skill names we want to link.
		skillNamesToLink := []string{"Go", "Terraform Provisioning"}

		params := ProcessNewTaskTxParams{
			CreateTaskParams: CreateTaskParams{
				ProjectID:   pgtype.Int8{Int64: project.ID, Valid: true},
				Title:       "Deploy a new service",
				Description: pgtype.Text{String: "Use Go and Terraform Provisioning to deploy.", Valid: true},
				Status:      TaskStatusOpen,
				Priority:    TaskPriorityMedium,
			},
			// Pass the pre-processed list of skills directly.
			RequiredSkillNames: skillNamesToLink,
		}

		// --- ACT ---
		result, err := store.ProcessNewTask(context.Background(), params)

		// --- ASSERT ---
		require.NoError(t, err)
		require.NotEmpty(t, result)
		require.Len(t, result.TaskRequiredSkills, 2, "Should have linked 2 skills")

		// Verify the new skill was created correctly (this assertion remains the same).
		newSkill, err := testQueries.GetSkillByName(context.Background(), "Terraform Provisioning")
		require.NoError(t, err)
		require.False(t, newSkill.IsVerified)

		// Verify the skills were actually linked to the task in the DB.
		linkedSkills, err := testQueries.GetSkillsForTask(context.Background(), result.Task.ID)
		require.NoError(t, err)
		require.Len(t, linkedSkills, 2)
	})
}

func TestAssignAndCompleteTaskLifecycle(t *testing.T) {
	store := NewStore(testPool)

	// --- ARRANGE ---
	// Create a user and a task that are initially unassociated.
	user, _ := createRandomUser(t)
	project := createRandomProject(t)
	task := createRandomTaskLocal(t, project.ID) // Creates an unassigned task.
	require.Equal(t, AvailabilityStatusAvailable, user.Availability)
	require.Equal(t, TaskStatusOpen, task.Status)

	t.Run("Lifecycle Step 1 - Assign Task To User", func(t *testing.T) {
		// --- ACT ---
		assignResult, err := store.AssignTaskToUser(context.Background(), AssignTaskToUserTxParams{
			TaskID: task.ID,
			UserID: user.ID,
		})

		// --- ASSERT ---
		require.NoError(t, err)
		// Check result struct
		require.Equal(t, task.ID, assignResult.Task.ID)
		require.Equal(t, user.ID, assignResult.User.ID)
		require.Equal(t, TaskStatusInProgress, assignResult.Task.Status)
		require.Equal(t, AvailabilityStatusBusy, assignResult.User.Availability)

		// Check DB state
		updatedTask, err := testQueries.GetTask(context.Background(), task.ID)
		require.NoError(t, err)
		require.Equal(t, user.ID, updatedTask.AssigneeID.Int64)
		require.Equal(t, TaskStatusInProgress, updatedTask.Status)

		updatedUser, err := testQueries.GetUser(context.Background(), user.ID)
		require.NoError(t, err)
		require.Equal(t, AvailabilityStatusBusy, updatedUser.Availability)
	})

	t.Run("Lifecycle Step 2 - Complete Task", func(t *testing.T) {
		// --- ACT ---
		err := store.CompleteTask(context.Background(), CompleteTaskTxParams{TaskID: task.ID})

		// --- ASSERT ---
		require.NoError(t, err)

		// Check DB state
		completedTask, err := testQueries.GetTask(context.Background(), task.ID)
		require.NoError(t, err)
		require.Equal(t, TaskStatusDone, completedTask.Status)
		require.True(t, completedTask.CompletedAt.Valid)
		require.WithinDuration(t, time.Now(), completedTask.CompletedAt.Time, 5*time.Second)

		freedUser, err := testQueries.GetUser(context.Background(), user.ID)
		require.NoError(t, err)
		require.Equal(t, AvailabilityStatusAvailable, freedUser.Availability)
	})

	t.Run("Edge Case - Complete Task with No Assignee", func(t *testing.T) {
		// --- ARRANGE ---
		// Create a new task that is not assigned to anyone.
		project := createRandomProject(t)
		unassignedTask := createRandomTaskLocal(t, project.ID)
		require.False(t, unassignedTask.AssigneeID.Valid)

		// --- ACT ---
		err := store.CompleteTask(context.Background(), CompleteTaskTxParams{TaskID: unassignedTask.ID})

		// --- ASSERT ---
		require.NoError(t, err, "Completing an unassigned task should not cause an error")

		// Verify task is marked as done.
		completedTask, err := testQueries.GetTask(context.Background(), unassignedTask.ID)
		require.NoError(t, err)
		require.Equal(t, TaskStatusDone, completedTask.Status)
		require.True(t, completedTask.CompletedAt.Valid)
	})
}

func TestAssignTaskToUser_Concurrent(t *testing.T) {
	// This test ensures that concurrent assignments are handled atomically and correctly.
	store := NewStore(testPool)
	n := 5 // Number of concurrent transactions

	// --- ARRANGE ---
	var users []User
	var tasks []Task
	project := createRandomProject(t)
	for range n {
		user, _ := createRandomUser(t)
		users = append(users, user)
		tasks = append(tasks, createRandomTaskLocal(t, project.ID))
	}

	// --- ACT ---
	var wg sync.WaitGroup
	errChan := make(chan error, n)

	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := store.AssignTaskToUser(context.Background(), AssignTaskToUserTxParams{
				TaskID: tasks[i].ID,
				UserID: users[i].ID,
			})
			if err != nil {
				errChan <- err
			}
		}(i)
	}
	wg.Wait()
	close(errChan)

	// --- ASSERT ---
	// First, check that no goroutine produced an error.
	for err := range errChan {
		require.NoError(t, err)
	}

	// Then, verify the final state of the database for each assignment.
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

// ---------------------------------------------------------------------------------
//                                 TEST HELPERS
// ---------------------------------------------------------------------------------

// Creates a random unassigned task for a given project.
func createRandomTaskLocal(t *testing.T, projectID int64) Task {
	arg := CreateTaskParams{
		ProjectID: pgtype.Int8{
			Int64: projectID,
			Valid: true,
		},
		Title: util.RandomTaskTitle(),
		Description: pgtype.Text{
			String: util.RandomTaskDescription(),
			Valid:  true,
		},
		Status:   TaskStatusOpen, // Default to open
		Priority: TaskPriority(util.RandomPriority()),
		AssigneeID: pgtype.Int8{ // Task is created unassigned
			Valid: false,
		},
	}

	task, err := testQueries.CreateTask(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, task)

	require.Equal(t, arg.ProjectID.Int64, task.ProjectID.Int64)
	require.False(t, task.AssigneeID.Valid)
	require.Equal(t, arg.Title, task.Title)

	return task
}
