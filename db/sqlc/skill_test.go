package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pranav244872/synapse/util"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////

func TestCreateSkill(t *testing.T) {
	createRandomSkill(t)
}

////////////////////////////////////////////////////////////////////////

func TestGetSkill(t *testing.T) {
	// Create a random skill
	skill1 := createRandomSkill(t)
	require.NotEmpty(t, skill1)

	// Retrieve the same skill from the database
	skill2, err := testQueries.GetSkill(context.Background(), skill1.ID)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, skill2)

	require.Equal(t, skill1.ID, skill2.ID)
	require.Equal(t, skill1.SkillName, skill2.SkillName)
	require.Equal(t, skill1.IsVerified, skill2.IsVerified)
}

////////////////////////////////////////////////////////////////////////

func TestUpdateSkill(t *testing.T) {
	// Create a random skill
	skill1 := createRandomSkill(t)
	require.NotEmpty(t, skill1)

	// Prepare parameters for the update
	arg := UpdateSkillParams{
		ID:        skill1.ID,
		SkillName: util.RandomName(),
	}

	// Update the skill
	skill2, err := testQueries.UpdateSkill(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, skill2)

	require.Equal(t, skill1.ID, skill2.ID)
	require.Equal(t, arg.SkillName, skill2.SkillName)
	require.NotEqual(t, skill1.SkillName, skill2.SkillName)
	require.Equal(t, skill1.IsVerified, skill2.IsVerified)
}

////////////////////////////////////////////////////////////////////////

func TestDeleteSkill(t *testing.T) {
	// Create a skill to delete
	skill1 := createRandomSkill(t)

	// Delete the skill
	err := testQueries.DeleteSkill(context.Background(), skill1.ID)
	require.NoError(t, err)

	// Try to retrieve the deleted skill
	skill2, err := testQueries.GetSkill(context.Background(), skill1.ID)

	// Assert that an error is returned and the skill object is empty
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.Empty(t, skill2)
}

////////////////////////////////////////////////////////////////////////

func TestListSkills(t *testing.T) {
	// Create 10 skills for pagination testing
	for range 10 {
		createRandomSkill(t)
	}

	arg := ListSkillsParams{
		Limit:  5,
		Offset: 5,
	}

	skills, err := testQueries.ListSkills(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.Len(t, skills, 5)

	for _, skill := range skills {
		require.NotEmpty(t, skill)
	}
}

////////////////////////////////////////////////////////////////////////

func TestUpdateSkillVerification(t *testing.T) {
	skill := createRandomSkill(t)
	require.False(t, skill.IsVerified)

	arg := UpdateSkillVerificationParams{
		ID:         skill.ID,
		IsVerified: true,
	}

	updatedSkill, err := testQueries.UpdateSkillVerification(context.Background(), arg)
	require.NoError(t, err)
	require.True(t, updatedSkill.IsVerified)
	require.Equal(t, skill.ID, updatedSkill.ID)
	require.Equal(t, skill.SkillName, updatedSkill.SkillName)
}

////////////////////////////////////////////////////////////////////////
