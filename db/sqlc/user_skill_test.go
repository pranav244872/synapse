package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/pranav244872/synapse/util"
)

////////////////////////////////////////////////////////////////////////

// TestAddSkillToUser just uses the helper function which already contains the necessary assertions.
func TestAddSkillToUser(t *testing.T) {
	createRandomUserSkill(t)
}

////////////////////////////////////////////////////////////////////////

func TestUpdateUserSkillProficiency(t *testing.T) {
	// 1. Create a user-skill relationship with a KNOWN proficiency.
	user := createRandomUser(t)
	skill := createRandomSkill(t)
	
	initialArg := AddSkillToUserParams{
		UserID:      user.ID,
		SkillID:     skill.ID,
		Proficiency: ProficiencyLevelBeginner, // Use a fixed, non-expert value
	}
	userSkill1, err := testQueries.AddSkillToUser(context.Background(), initialArg)
	require.NoError(t, err)
	require.Equal(t, ProficiencyLevelBeginner, userSkill1.Proficiency)

	// 2. Prepare to update the proficiency to "expert"
	updateArg := UpdateUserSkillProficiencyParams{
		UserID:      userSkill1.UserID,
		SkillID:     userSkill1.SkillID,
		Proficiency: ProficiencyLevelExpert, // Update to the target level
	}

	// 3. Update the proficiency
	userSkill2, err := testQueries.UpdateUserSkillProficiency(context.Background(), updateArg)
	require.NoError(t, err)
	require.NotEmpty(t, userSkill2)

	// 4. Assertions
	require.Equal(t, userSkill1.UserID, userSkill2.UserID)
	require.Equal(t, userSkill1.SkillID, userSkill2.SkillID)
	// Assert the new proficiency is correct
	require.Equal(t, ProficiencyLevelExpert, userSkill2.Proficiency)
	// Assert the new proficiency is different from the old one
	require.NotEqual(t, userSkill1.Proficiency, userSkill2.Proficiency)
}

////////////////////////////////////////////////////////////////////////

func TestRemoveSkillFromUser(t *testing.T) {
	// Create a user-skill relationship
	userSkill := createRandomUserSkill(t)

	// Prepare parameters for removal
	arg := RemoveSkillFromUserParams{
		UserID:  userSkill.UserID,
		SkillID: userSkill.SkillID,
	}

	// Remove the skill from the user
	err := testQueries.RemoveSkillFromUser(context.Background(), arg)
	require.NoError(t, err)

	// Try to retrieve the skills for that user
	skills, err := testQueries.GetSkillsForUser(context.Background(), userSkill.UserID)
	require.NoError(t, err)

	// Assert that the list of skills is now empty
	require.Empty(t, skills)
}

////////////////////////////////////////////////////////////////////////

func TestGetSkillsForUser(t *testing.T) {
	// Create a user
	user := createRandomUser(t)

	// Add 3 different skills to this user
	for range 3 {
		skill := createRandomSkill(t)
		arg := AddSkillToUserParams{
			UserID:      user.ID,
			SkillID:     skill.ID,
			Proficiency: ProficiencyLevel(util.RandomProficiency()),
		}
		_, err := testQueries.AddSkillToUser(context.Background(), arg)
		require.NoError(t, err)
	}

	// Retrieve the skills for the user
	skills, err := testQueries.GetSkillsForUser(context.Background(), user.ID)
	require.NoError(t, err)

	// Assert that exactly 3 skills were returned
	require.Len(t, skills, 3)
	for _, skill := range skills {
		require.NotEmpty(t, skill)
	}
}

////////////////////////////////////////////////////////////////////////

func TestGetUsersWithSkill(t *testing.T) {
	// Create a skill
	skill := createRandomSkill(t)

	// Create 5 different users and assign the same skill to them
	for range 5 {
		user := createRandomUser(t)
		arg := AddSkillToUserParams{
			UserID:      user.ID,
			SkillID:     skill.ID,
			Proficiency: ProficiencyLevel(util.RandomProficiency()),
		}
		_, err := testQueries.AddSkillToUser(context.Background(), arg)
		require.NoError(t, err)
	}

	// Retrieve all users who have this skill
	users, err := testQueries.GetUsersWithSkill(context.Background(), skill.ID)
	require.NoError(t, err)

	// Assert that exactly 5 users were returned
	require.Len(t, users, 5)
	for _, user := range users {
		require.NotEmpty(t, user)
	}
}

////////////////////////////////////////////////////////////////////////
