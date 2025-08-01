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

////////////////////////////////////////////////////////////////////////////////
//                               TRANSACTION TESTS
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Test: OnboardNewUserWithSkills
////////////////////////////////////////////////////////////////////////////////

func TestOnboardNewUserWithSkills(t *testing.T) {
	store := NewStore(testPool)

	t.Run("Happy Path - Mix of New and Existing Skills", func(t *testing.T) {
		// Arrange
		preExistingSkill := createRandomSkill(t)
		brandNewSkillName := util.RandomString(12)
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

		// Act
		result, err := store.OnboardNewUserWithSkills(context.Background(), params)

		// Assert
		require.NoError(t, err)
		require.NotEmpty(t, result)
		require.Equal(t, randomEmail, result.User.Email)
		require.Len(t, result.UserSkills, 2)

		newlyCreatedSkill, err := testQueries.GetSkillByName(context.Background(), brandNewSkillName)
		require.NoError(t, err)
		require.False(t, newlyCreatedSkill.IsVerified)

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

	t.Run("Edge Case - No Skills Provided", func(t *testing.T) {
		// Arrange
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

		// Act
		result, err := store.OnboardNewUserWithSkills(context.Background(), params)

		// Assert
		require.NoError(t, err)
		require.NotEmpty(t, result.User)
		require.Empty(t, result.UserSkills)

		dbUser, err := testQueries.GetUserByEmail(context.Background(), params.CreateUserParams.Email)
		require.NoError(t, err)
		require.Equal(t, params.CreateUserParams.Email, dbUser.Email)
	})

	t.Run("Failure - Duplicate Email Rolls Back Transaction", func(t *testing.T) {
		// Arrange
		existingUser, _ := createRandomUser(t)

		params := OnboardNewUserTxParams{
			CreateUserParams: CreateUserParams{
				Name:         pgtype.Text{String: "Duplicate Hire", Valid: true},
				Email:        existingUser.Email,
				PasswordHash: "hashed_password",
				TeamID:       pgtype.Int8{Int64: existingUser.TeamID.Int64, Valid: true},
				Role:         UserRoleEngineer,
			},
			SkillsWithProficiency: map[string]ProficiencyLevel{"Go": ProficiencyLevelExpert},
		}

		// Act
		_, err := store.OnboardNewUserWithSkills(context.Background(), params)

		// Assert
		require.Error(t, err, "Transaction should fail due to unique constraint on email")
	})
}

////////////////////////////////////////////////////////////////////////////////
// Test: ProcessNewTask
////////////////////////////////////////////////////////////////////////////////

func TestProcessNewTask(t *testing.T) {
	store := NewStore(testPool)

	t.Run("Happy Path - Creates Task and Links Skills", func(t *testing.T) {
		// Arrange
		project := createRandomProject(t)
		skillNamesToLink := []string{"Go", "Terraform Provisioning"}

		params := ProcessNewTaskTxParams{
			CreateTaskParams: CreateTaskParams{
				ProjectID:   pgtype.Int8{Int64: project.ID, Valid: true},
				Title:       "Deploy a new service",
				Description: pgtype.Text{String: "Use Go and Terraform Provisioning to deploy.", Valid: true},
				Status:      TaskStatusOpen,
				Priority:    TaskPriorityMedium,
			},
			RequiredSkillNames: skillNamesToLink,
		}

		// Act
		result, err := store.ProcessNewTask(context.Background(), params)

		// Assert
		require.NoError(t, err)
		require.NotEmpty(t, result)
		require.Len(t, result.TaskRequiredSkills, 2)

		newSkill, err := testQueries.GetSkillByName(context.Background(), "Terraform Provisioning")
		require.NoError(t, err)
		require.False(t, newSkill.IsVerified)

		linkedSkills, err := testQueries.GetSkillsForTask(context.Background(), result.Task.ID)
		require.NoError(t, err)
		require.Len(t, linkedSkills, 2)
	})
}

////////////////////////////////////////////////////////////////////////////////
// Test: AssignTaskToUser and CompleteTask Lifecycle
////////////////////////////////////////////////////////////////////////////////

func TestAssignAndCompleteTaskLifecycle(t *testing.T) {
	store := NewStore(testPool)

	user, _ := createRandomUser(t)
	project := createRandomProject(t)
	task := createRandomTaskLocal(t, project.ID)
	require.Equal(t, AvailabilityStatusAvailable, user.Availability)
	require.Equal(t, TaskStatusOpen, task.Status)

	t.Run("Lifecycle Step 1 - Assign Task To User", func(t *testing.T) {
		// Act
		assignResult, err := store.AssignTaskToUser(context.Background(), AssignTaskToUserTxParams{
			TaskID: task.ID,
			UserID: user.ID,
		})

		// Assert
		require.NoError(t, err)
		require.Equal(t, task.ID, assignResult.Task.ID)
		require.Equal(t, user.ID, assignResult.User.ID)
		require.Equal(t, TaskStatusInProgress, assignResult.Task.Status)
		require.Equal(t, AvailabilityStatusBusy, assignResult.User.Availability)

		updatedTask, err := testQueries.GetTask(context.Background(), task.ID)
		require.NoError(t, err)
		require.Equal(t, user.ID, updatedTask.AssigneeID.Int64)
		require.Equal(t, TaskStatusInProgress, updatedTask.Status)

		updatedUser, err := testQueries.GetUser(context.Background(), user.ID)
		require.NoError(t, err)
		require.Equal(t, AvailabilityStatusBusy, updatedUser.Availability)
	})

	t.Run("Lifecycle Step 2 - Complete Task", func(t *testing.T) {
		// Act
		err := store.CompleteTask(context.Background(), CompleteTaskTxParams{TaskID: task.ID})

		// Assert
		require.NoError(t, err)

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
		project := createRandomProject(t)
		unassignedTask := createRandomTaskLocal(t, project.ID)
		require.False(t, unassignedTask.AssigneeID.Valid)

		// Act
		err := store.CompleteTask(context.Background(), CompleteTaskTxParams{TaskID: unassignedTask.ID})

		// Assert
		require.NoError(t, err)
		completedTask, err := testQueries.GetTask(context.Background(), unassignedTask.ID)
		require.NoError(t, err)
		require.Equal(t, TaskStatusDone, completedTask.Status)
		require.True(t, completedTask.CompletedAt.Valid)
	})
}

////////////////////////////////////////////////////////////////////////////////
// Test: AssignTaskToUser – Concurrency
////////////////////////////////////////////////////////////////////////////////

func TestAssignTaskToUser_Concurrent(t *testing.T) {
	store := NewStore(testPool)
	n := 5

	var users []User
	var tasks []Task
	project := createRandomProject(t)

	for range n {
		user, _ := createRandomUser(t)
		users = append(users, user)
		tasks = append(tasks, createRandomTaskLocal(t, project.ID))
	}

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

	for err := range errChan {
		require.NoError(t, err)
	}

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

////////////////////////////////////////////////////////////////////////////////
// Test: CreateInvitationTx
////////////////////////////////////////////////////////////////////////////////

func TestCreateInvitationTx(t *testing.T) {
	store := NewStore(testPool)

	t.Run("Success: Admin invites Manager to a vacant team", func(t *testing.T) {
		admin, _ := createRandomUserWithRole(t, UserRoleAdmin)
		vacantTeam := createRandomTeam(t) // A new team is vacant by default.
		inviteeEmail := util.RandomEmail()

		params := CreateInvitationTxParams{
			InviterID:     admin.ID,
			EmailToInvite: inviteeEmail,
			RoleToInvite:  UserRoleManager,
			TeamID:        pgtype.Int8{Int64: vacantTeam.ID, Valid: true},
		}

		result, err := store.CreateInvitationTx(context.Background(), params)
		require.NoError(t, err)
		require.NotEmpty(t, result)
		require.NotEmpty(t, result.Invitation)

		// Verify the created invitation from the DB.
		invitation, err := testQueries.GetInvitationByEmail(context.Background(), inviteeEmail)
		require.NoError(t, err)
		require.Equal(t, admin.ID, invitation.InviterID)
		require.Equal(t, UserRoleManager, invitation.RoleToInvite)
		require.Equal(t, "pending", invitation.Status)
		require.True(t, invitation.TeamID.Valid)
		require.Equal(t, vacantTeam.ID, invitation.TeamID.Int64)
	})

	t.Run("Success: Manager invites Engineer to their own team", func(t *testing.T) {
		manager, team := createRandomManagerWithTeam(t) // Helper creates a manager assigned to a team.
		inviteeEmail := util.RandomEmail()

		params := CreateInvitationTxParams{
			InviterID:     manager.ID,
			EmailToInvite: inviteeEmail,
			RoleToInvite:  UserRoleEngineer,
			// TeamID is omitted as it's inferred from the manager.
		}

		result, err := store.CreateInvitationTx(context.Background(), params)
		require.NoError(t, err)
		require.NotEmpty(t, result)

		// Verify the invitation is correctly assigned to the manager's team.
		invitation, err := testQueries.GetInvitationByEmail(context.Background(), inviteeEmail)
		require.NoError(t, err)
		require.Equal(t, manager.ID, invitation.InviterID)
		require.Equal(t, UserRoleEngineer, invitation.RoleToInvite)
		require.True(t, invitation.TeamID.Valid)
		require.Equal(t, team.ID, invitation.TeamID.Int64)
	})

	t.Run("Failure: Engineer attempts to invite", func(t *testing.T) {
		engineer, _ := createRandomUserWithRole(t, UserRoleEngineer)
		params := CreateInvitationTxParams{
			InviterID:     engineer.ID,
			EmailToInvite: util.RandomEmail(),
			RoleToInvite:  UserRoleEngineer,
		}

		_, err := store.CreateInvitationTx(context.Background(), params)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrPermissionDenied)
	})

	t.Run("Failure: Invalid role sequence (Admin to Engineer)", func(t *testing.T) {
		admin, _ := createRandomUserWithRole(t, UserRoleAdmin)
		params := CreateInvitationTxParams{
			InviterID:     admin.ID,
			EmailToInvite: util.RandomEmail(),
			RoleToInvite:  UserRoleEngineer, // Admins can only invite Managers.
		}

		_, err := store.CreateInvitationTx(context.Background(), params)
		require.Error(t, err)
		require.ErrorContains(t, err, ErrInvalidRoleSequence.Error())
	})

	t.Run("Failure: Duplicate pending invitation for an email", func(t *testing.T) {
		admin, _ := createRandomUserWithRole(t, UserRoleAdmin)
		vacantTeam := createRandomTeam(t)
		inviteeEmail := util.RandomEmail()

		// First invitation should succeed.
		_, err := store.CreateInvitationTx(context.Background(), CreateInvitationTxParams{
			InviterID:     admin.ID,
			EmailToInvite: inviteeEmail,
			RoleToInvite:  UserRoleManager,
			TeamID:        pgtype.Int8{Int64: vacantTeam.ID, Valid: true},
		})
		require.NoError(t, err)

		// Second invitation for the same email should fail.
		_, err = store.CreateInvitationTx(context.Background(), CreateInvitationTxParams{
			InviterID:     admin.ID,
			EmailToInvite: inviteeEmail,
			RoleToInvite:  UserRoleManager,
			TeamID:        pgtype.Int8{Int64: vacantTeam.ID, Valid: true},
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDuplicateInvitation)
	})

	t.Run("Failure: Admin invites Manager without providing a TeamID", func(t *testing.T) {
		admin, _ := createRandomUserWithRole(t, UserRoleAdmin)
		params := CreateInvitationTxParams{
			InviterID:     admin.ID,
			EmailToInvite: util.RandomEmail(),
			RoleToInvite:  UserRoleManager,
			TeamID:        pgtype.Int8{Valid: false}, // Explicitly omit TeamID.
		}

		_, err := store.CreateInvitationTx(context.Background(), params)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrTeamIDRequiredForManager)
	})

	t.Run("Failure: Manager without a team invites an Engineer", func(t *testing.T) {
		// Create a manager but do not assign them to any team.
		managerWithoutTeam, _ := createRandomUserWithRoleAndNoTeam(t, UserRoleManager)

		params := CreateInvitationTxParams{
			InviterID:     managerWithoutTeam.ID,
			EmailToInvite: util.RandomEmail(),
			RoleToInvite:  UserRoleEngineer,
		}

		_, err := store.CreateInvitationTx(context.Background(), params)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrManagerMustHaveTeam)
	})

	t.Run("Failure: Admin invites Manager to a non-existent team", func(t *testing.T) {
		admin, _ := createRandomUserWithRole(t, UserRoleAdmin)
		params := CreateInvitationTxParams{
			InviterID:     admin.ID,
			EmailToInvite: util.RandomEmail(),
			RoleToInvite:  UserRoleManager,
			TeamID:        pgtype.Int8{Int64: -99, Valid: true}, // Non-existent team ID.
		}

		_, err := store.CreateInvitationTx(context.Background(), params)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrTeamNotFound)
	})

	t.Run("Failure: Admin invites Manager to an already occupied team", func(t *testing.T) {
		admin, _ := createRandomUserWithRole(t, UserRoleAdmin)
		// Create a team that is already managed.
		_, occupiedTeam := createRandomManagerWithTeam(t)

		params := CreateInvitationTxParams{
			InviterID:     admin.ID,
			EmailToInvite: util.RandomEmail(),
			RoleToInvite:  UserRoleManager,
			TeamID:        pgtype.Int8{Int64: occupiedTeam.ID, Valid: true},
		}

		_, err := store.CreateInvitationTx(context.Background(), params)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrTeamAlreadyHasManager)
	})

	t.Run("Failure: Inviter not found", func(t *testing.T) {
		nonExistentInviterID := int64(99999999)
		params := CreateInvitationTxParams{
			InviterID:     nonExistentInviterID,
			EmailToInvite: util.RandomEmail(),
			RoleToInvite:  UserRoleManager,
		}

		_, err := store.CreateInvitationTx(context.Background(), params)
		require.Error(t, err)
		require.Contains(t, err.Error(), "inviter with ID 99999999 not found")
	})
}

////////////////////////////////////////////////////////////////////////////////
//                               TEST HELPERS
////////////////////////////////////////////////////////////////////////////////

// createRandomTaskLocal creates a task in a given project with random values.
func createRandomTaskLocal(t *testing.T, projectID int64) Task {
	arg := CreateTaskParams{
		ProjectID: pgtype.Int8{Int64: projectID, Valid: true},
		Title:     util.RandomTaskTitle(),
		Description: pgtype.Text{
			String: util.RandomTaskDescription(),
			Valid:  true,
		},
		Status:     TaskStatusOpen,
		Priority:   TaskPriority(util.RandomPriority()),
		AssigneeID: pgtype.Int8{Valid: false},
	}

	task, err := testQueries.CreateTask(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, task)

	require.Equal(t, arg.ProjectID.Int64, task.ProjectID.Int64)
	require.False(t, task.AssigneeID.Valid)
	require.Equal(t, arg.Title, task.Title)

	return task
}

// createRandomUserWithRole creates a user with a given role and sets it explicitly.
func createRandomUserWithRole(t *testing.T, role UserRole) (User, string) {
	user, password := createRandomUser(t)
	updatedUser, err := testQueries.UpdateUser(context.Background(), UpdateUserParams{
		ID:   user.ID,
		Role: NullUserRole{UserRole: role, Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, role, updatedUser.Role)
	return updatedUser, password
}

// createRandomUserWithRoleAndNoTeam creates a user with a role and ensures they are not assigned to any team.
func createRandomUserWithRoleAndNoTeam(t *testing.T, role UserRole) (User, string) {
	user, password := createRandomUserWithRole(t, role)

	// Explicitly remove user from team to ensure team_id is NULL.
	unassignedUser, err := testQueries.RemoveUserFromTeam(context.Background(), user.ID)
	require.NoError(t, err)

	require.False(t, unassignedUser.TeamID.Valid, "User should not have a team ID")
	require.Equal(t, role, unassignedUser.Role)

	return unassignedUser, password
}

// createRandomManagerWithTeam sets up a team and assigns a manager to it.
// It updates both the user and the team to ensure referential integrity.
func createRandomManagerWithTeam(t *testing.T) (User, Team) {
	team := createRandomTeam(t)

	// Create a user with manager role.
	manager, _ := createRandomUserWithRole(t, UserRoleManager)

	// Assign manager to team by updating the user's team_id.
	updatedManager, err := testQueries.UpdateUser(context.Background(), UpdateUserParams{
		ID:     manager.ID,
		TeamID: pgtype.Int8{Int64: team.ID, Valid: true},
	})
	require.NoError(t, err)
	require.True(t, updatedManager.TeamID.Valid)
	require.Equal(t, team.ID, updatedManager.TeamID.Int64)

	// Set team's manager_id to the user.
	updatedTeam, err := testQueries.UpdateTeam(context.Background(), UpdateTeamParams{
		ID:        team.ID,
		ManagerID: pgtype.Int8{Int64: updatedManager.ID, Valid: true},
	})
	require.NoError(t, err)
	require.True(t, updatedTeam.ManagerID.Valid)
	require.Equal(t, updatedManager.ID, updatedTeam.ManagerID.Int64)

	return updatedManager, updatedTeam
}
