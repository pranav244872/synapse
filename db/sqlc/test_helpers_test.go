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
	projectName := util.RandomProjectName()

	project, err := testQueries.CreateProject(context.Background(), projectName)

	require.NoError(t, err)
	require.NotEmpty(t, project)

	require.Equal(t, projectName, project.ProjectName)
	require.NotZero(t, project.ID)

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

// createRandomInvitation creates a new random invitation for testing.
// It creates a random user as the inviter.
func createRandomInvitation(t *testing.T) Invitation {
	// Create a user to act as the inviter
	inviter, _ := createRandomUser(t)

	arg := CreateInvitationParams{
		Email:           util.RandomEmail(),
		InvitationToken: util.RandomString(32),
		RoleToInvite:    UserRoleEngineer, // Defaulting to Engineer for tests
		InviterID:       inviter.ID,
		ExpiresAt: pgtype.Timestamp{
			Time:  time.Now().Add(24 * time.Hour), // Invitation valid for 24 hours
			Valid: true,
		},
	}

	invitation, err := testQueries.CreateInvitation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, invitation)

	// Verify the created invitation fields
	require.NotZero(t, invitation.ID)
	require.Equal(t, arg.Email, invitation.Email)
	require.Equal(t, arg.InvitationToken, invitation.InvitationToken)
	require.Equal(t, arg.RoleToInvite, invitation.RoleToInvite)
	require.Equal(t, arg.InviterID, invitation.InviterID)
	require.Equal(t, "pending", invitation.Status) // Assuming default status is 'pending'
	require.WithinDuration(t, arg.ExpiresAt.Time, invitation.ExpiresAt.Time, time.Second)
	require.NotZero(t, invitation.CreatedAt)
	// The line checking for invitation.UpdatedAt has been removed.

	return invitation
}

////////////////////////////////////////////////////////////////////////
