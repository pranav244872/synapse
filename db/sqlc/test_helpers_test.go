package db

import (
	"context"
	"testing"

	"github.com/pranav244872/synapse/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////

// Creates a random team which is a basic entity
func createRandomTeam(t *testing.T) Team {
	teamName := util.RandomName()

	team, err := testQueries.CreateTeam(context.Background(), teamName)
	require.NoError(t, err)
	require.NotEmpty(t, team)

	require.Equal(t, teamName, team.TeamName)
	require.NotZero(t, team.ID)

	return team
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
func createRandomUser(t *testing.T) User {
	team := createRandomTeam(t)

	arg := CreateUserParams{
		Name: pgtype.Text{
			String: util.RandomName(),
			Valid:  true,
		},
		Email: util.RandomEmail(),
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
	return user
}

////////////////////////////////////////////////////////////////////////

// Creates random task, depends on "users" (assignee_id)
func createRandomTask(t *testing.T) Task {
	project := createRandomProject(t)
	assignee := createRandomUser(t)

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
    user := createRandomUser(t)
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
