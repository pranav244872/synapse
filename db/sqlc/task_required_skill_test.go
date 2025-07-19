package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)


////////////////////////////////////////////////////////////////////////

// TestAddSkillToTask simply uses the helper function, which already contains
// the necessary assertions for a successful creation.
func TestAddSkillToTask(t *testing.T) {
	createRandomTaskSkill(t)
}

////////////////////////////////////////////////////////////////////////

// TestRemoveSkillFromTask ensures a skill can be disassociated from a task.
func TestRemoveSkillFromTask(t *testing.T) {
	// 1. Setup: Create a task and associate a skill with it.
	task, skill, _ := createRandomTaskSkill(t)

	// 2. Execute: Remove the skill from the task.
	arg := RemoveSkillFromTaskParams{
		TaskID:  task.ID,
		SkillID: skill.ID,
	}
	err := testQueries.RemoveSkillFromTask(context.Background(), arg)
	require.NoError(t, err)

	// 3. Verify: Check that the skill is no longer listed for the task.
	skills, err := testQueries.GetSkillsForTask(context.Background(), task.ID)
	require.NoError(t, err)
	require.Empty(t, skills) // The list of skills should now be empty.
}

////////////////////////////////////////////////////////////////////////

// TestGetSkillsForTask checks retrieval of all skills for one task.
func TestGetSkillsForTask(t *testing.T) {
	// 1. Setup: Create one task.
	task := createRandomTask(t)

	// 2. Execute: Add 3 different required skills to this single task.
	for range 3 {
		skill := createRandomSkill(t)
		arg := AddSkillToTaskParams{
			TaskID:  task.ID,
			SkillID: skill.ID,
		}
		_, err := testQueries.AddSkillToTask(context.Background(), arg)
		require.NoError(t, err)
	}

	// 3. Verify: Retrieve all skills for the task.
	skills, err := testQueries.GetSkillsForTask(context.Background(), task.ID)
	require.NoError(t, err)
	require.Len(t, skills, 3) // Should find exactly 3 skills.

	for _, skill := range skills {
		require.NotEmpty(t, skill)
	}
}

////////////////////////////////////////////////////////////////////////

// TestGetTasksForSkill checks retrieval of all tasks that require one skill.
func TestGetTasksForSkill(t *testing.T) {
	// 1. Setup: Create one skill.
	skill := createRandomSkill(t)

	// 2. Execute: Create 5 different tasks that all require this single skill.
	for range 5 {
		task := createRandomTask(t)
		arg := AddSkillToTaskParams{
			TaskID:  task.ID,
			SkillID: skill.ID,
		}
		_, err := testQueries.AddSkillToTask(context.Background(), arg)
		require.NoError(t, err)
	}

	// 3. Verify: Retrieve all tasks that require the skill.
	tasks, err := testQueries.GetTasksForSkill(context.Background(), skill.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 5) // Should find exactly 5 tasks.

	for _, task := range tasks {
		require.NotEmpty(t, task)
	}
}

////////////////////////////////////////////////////////////////////////
