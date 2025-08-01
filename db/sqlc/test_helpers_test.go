package db

import (
	"context"
	"testing"
	"time"
	"github.com/pranav244872/synapse/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////

// Creates a random team without a manager.
func createRandomTeam(t *testing.T) Team {
	teamName := util.RandomName()

	// The CreateTeam function now requires a CreateTeamParams struct.
	// We set ManagerID to be invalid (NULL) for this basic helper.
	arg := CreateTeamParams{
		TeamName:  teamName,
		ManagerID: pgtype.Int8{Valid: false},
	}

	team, err := testQueries.CreateTeam(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, team)

	require.Equal(t, teamName, team.TeamName)
	require.False(t, team.ManagerID.Valid) // Assert the team has no manager.
	require.NotZero(t, team.ID)

	return team
}

////////////////////////////////////////////////////////////////////////

// create a team with a manager for testing.
func createRandomTeamWithManager(t *testing.T) (Team, User) {
	manager, _ := createRandomUser(t)
	teamName := util.RandomName()

	arg := CreateTeamParams{
		TeamName:  teamName,
		ManagerID: pgtype.Int8{Int64: manager.ID, Valid: true},
	}

	team, err := testQueries.CreateTeam(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, team)

	return team, manager
}

////////////////////////////////////////////////////////////////////////

// createRandomSkill creates a random skill with is_verified = false
func createRandomSkill(t *testing.T) Skill {
	skillName := util.RandomName()

	arg := CreateSkillParams{
		SkillName:  skillName,
		IsVerified: false, // default behavior for new skills
	}

	skill, err := testQueries.CreateSkill(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, skill)

	require.Equal(t, arg.SkillName, skill.SkillName)
	require.Equal(t, arg.IsVerified, skill.IsVerified)
	require.NotZero(t, skill.ID)

	return skill
}

////////////////////////////////////////////////////////////////////////

// Creates a random project which is a basic entity
func createRandomProject(t *testing.T) Project {
	team := createRandomTeam(t)

	desc := pgtype.Text{String: util.RandomString(50), Valid: true}

	arg := CreateProjectParams{
		ProjectName: util.RandomProjectName(),
		TeamID:      team.ID,
		Description: desc,
	}

	project, err := testQueries.CreateProject(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, project)
	require.Equal(t, arg.ProjectName, project.ProjectName)
	require.Equal(t, arg.TeamID, project.TeamID)
	require.Equal(t, arg.Description.String, project.Description.String)

	return project
}

////////////////////////////////////////////////////////////////////////

// Creates random user depends on "teams" (team_id)
// It returns both the User object and the original plaintext password
func createRandomUser(t *testing.T) (User, string) {
	password := util.RandomString(10)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword)

	team := createRandomTeam(t)

	arg := CreateUserParams{
		Name: pgtype.Text{
			String: util.RandomName(),
			Valid:  true,
		},
		Email: util.RandomEmail(),
		PasswordHash: hashedPassword,
		Role: UserRoleEngineer,
		TeamID: pgtype.Int8{
			Int64: team.ID,
			Valid: true,
		},
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)
	require.Equal(t, arg.Name, user.Name)
	require.Equal(t, arg.Email, user.Email)

	require.Equal(t, arg.TeamID.Int64, user.TeamID.Int64)

	return user, password
}

////////////////////////////////////////////////////////////////////////

// Creates random task, depends on "users" (assignee_id)
func createRandomTask(t *testing.T) Task {
	project := createRandomProject(t)
	assignee, _ := createRandomUser(t)

	arg := CreateTaskParams{
		ProjectID: pgtype.Int8{
			Int64: project.ID,
			Valid: true,
		},
		Title: util.RandomTaskTitle(),
		Description: pgtype.Text{
			String: util.RandomTaskDescription(),
			Valid:  true,
		},
		Status: TaskStatus(util.RandomStatus()),
		Priority: TaskPriority(util.RandomPriority()),
		AssigneeID: pgtype.Int8{
			Int64: assignee.ID,
			Valid: true,
		},
	}

	task, err := testQueries.CreateTask(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, task)

	require.Equal(t, arg.ProjectID.Int64, task.ProjectID.Int64)
	require.Equal(t, arg.AssigneeID.Int64, task.AssigneeID.Int64)
	require.Equal(t, arg.Title, task.Title)
	require.Equal(t, arg.Description.String, task.Description.String)
	require.Equal(t, arg.Status, task.Status)
	require.Equal(t, arg.Priority, task.Priority)
	require.NotZero(t, task.ID)
	require.NotZero(t, task.CreatedAt)

	return task
}

////////////////////////////////////////////////////////////////////////

// Creates a random user-skill relationship.
func createRandomUserSkill(t *testing.T) UserSkill {
    user, _ := createRandomUser(t)
    skill := createRandomSkill(t)

    arg := AddSkillToUserParams{
        UserID:      user.ID,
        SkillID:     skill.ID,
        Proficiency: ProficiencyLevel(util.RandomProficiency()), // Assumes a util.RandomProficiency() function
    }

    userSkill, err := testQueries.AddSkillToUser(context.Background(), arg)
    require.NoError(t, err)
    require.NotEmpty(t, userSkill)

    require.Equal(t, arg.UserID, userSkill.UserID)
    require.Equal(t, arg.SkillID, userSkill.SkillID)
    require.Equal(t, arg.Proficiency, userSkill.Proficiency)

    return userSkill
}

////////////////////////////////////////////////////////////////////////

// Creates a random task-skill relationship.
// Returns the created task, the skill, and the junction table record.
func createRandomTaskSkill(t *testing.T) (Task, Skill, TaskRequiredSkill) {
	task := createRandomTask(t)
	skill := createRandomSkill(t)

	arg := AddSkillToTaskParams{
		TaskID:  task.ID,
		SkillID: skill.ID,
	}

	taskSkill, err := testQueries.AddSkillToTask(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, taskSkill)

	require.Equal(t, task.ID, taskSkill.TaskID)
	require.Equal(t, skill.ID, taskSkill.SkillID)

	return task, skill, taskSkill
}

////////////////////////////////////////////////////////////////////////

// createRandomSkillAlias creates a random skill alias for testing.
// It depends on a helper function createRandomSkill(t) to satisfy the foreign key constraint.
func createRandomSkillAlias(t *testing.T) SkillAlias {
	// An alias must point to an existing skill.
	skill := createRandomSkill(t)

	arg := CreateSkillAliasParams{
		AliasName: util.RandomString(8), // e.g., "golang"
		SkillID:   skill.ID,
	}

	alias, err := testQueries.CreateSkillAlias(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, alias)

	require.Equal(t, arg.AliasName, alias.AliasName)
	require.Equal(t, arg.SkillID, alias.SkillID)

	return alias
}

////////////////////////////////////////////////////////////////////////

// createRandomInvitation is a helper function that creates a random team and user (inviter)
// and then creates a new invitation associated with that team. It contains all the
// necessary assertions for a successful creation. This is the foundation for all other tests.
func createRandomInvitation(t *testing.T) Invitation {
	// Step 1: Create a random user who will act as the inviter.
	// We assume a 'createRandomUser' helper function exists for this purpose.
	inviter, _ := createRandomUser(t)

	// Step 2: Create a random team to associate the invitation with.
	// We assume a 'createRandomTeam' helper function exists.
	team := createRandomTeam(t)

	// Step 3: Define the arguments for creating the invitation.
	// We use data from the newly created inviter and team.
	arg := CreateInvitationParams{
		Email:           util.RandomEmail(),
		InvitationToken: util.RandomString(32),
		RoleToInvite:    UserRoleEngineer,
		InviterID:       inviter.ID,
		ExpiresAt: pgtype.Timestamp{
			Time:  time.Now().Add(time.Hour * 24),
			Valid: true,
		},
		// Crucially, we associate the invitation with the team created in Step 2.
		TeamID: pgtype.Int8{
			Int64: team.ID,
			Valid: true,
		},
	}

	// Step 4: Execute the CreateInvitation query.
	invitation, err := testQueries.CreateInvitation(context.Background(), arg)

	// Step 5: Assert that the creation process was successful and the returned data is correct.
	require.NoError(t, err)
	require.NotEmpty(t, invitation)

	// Verify that the static fields were set correctly.
	require.Equal(t, arg.Email, invitation.Email)
	require.Equal(t, arg.InvitationToken, invitation.InvitationToken)
	require.Equal(t, arg.RoleToInvite, invitation.RoleToInvite)
	require.Equal(t, arg.InviterID, invitation.InviterID)

	// Verify that the TeamID was set correctly.
	require.True(t, invitation.TeamID.Valid)
	require.Equal(t, arg.TeamID.Int64, invitation.TeamID.Int64)

	// Verify the fields automatically set by the database.
	require.NotZero(t, invitation.ID)
	require.Equal(t, "pending", invitation.Status) // Default status should be 'pending'.
	require.NotZero(t, invitation.CreatedAt)
	require.WithinDuration(t, arg.ExpiresAt.Time, invitation.ExpiresAt.Time, time.Second)

	return invitation
}

////////////////////////////////////////////////////////////////////////

// createExpiredInvitation is a helper function that creates an invitation
// with an expiration timestamp that is already in the past.
func createExpiredInvitation(t *testing.T) Invitation {
	// Step 1: Create a random user (inviter) and a team.
	inviter, _ := createRandomUser(t)
	team := createRandomTeam(t)

	// Step 2: Define arguments for the invitation.
	arg := CreateInvitationParams{
		Email:           util.RandomEmail(),
		InvitationToken: util.RandomString(32),
		RoleToInvite:    UserRoleEngineer,
		InviterID:       inviter.ID,
		TeamID:          pgtype.Int8{Int64: team.ID, Valid: true},
		// **Key Difference**: Set the expiration time to one hour in the past.
		ExpiresAt: pgtype.Timestamp{
			Time:  time.Now().Add(-time.Hour),
			Valid: true,
		},
	}

	// Step 3: Execute the query to create the expired invitation.
	invitation, err := testQueries.CreateInvitation(context.Background(), arg)

	// Step 4: Assert that the creation was successful.
	require.NoError(t, err)
	require.NotEmpty(t, invitation)

	// Verify the data was set as expected.
	require.Equal(t, arg.Email, invitation.Email)
	require.NotZero(t, invitation.ID)
	// Check that the expired time was set correctly.
	require.WithinDuration(t, arg.ExpiresAt.Time, invitation.ExpiresAt.Time, time.Second)

	return invitation
}

////////////////////////////////////////////////////////////////////////
