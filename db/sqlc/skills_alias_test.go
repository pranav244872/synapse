// skill_aliases_test.go
package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pranav244872/synapse/util"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////

func TestCreateSkillAlias(t *testing.T) {
	createRandomSkillAlias(t)
}

////////////////////////////////////////////////////////////////////////

func TestGetSkillAlias(t *testing.T) {
	// Success Case
	alias1 := createRandomSkillAlias(t)
	alias2, err := testQueries.GetSkillAlias(context.Background(), alias1.AliasName)

	require.NoError(t, err)
	require.NotEmpty(t, alias2)
	require.Equal(t, alias1.AliasName, alias2.AliasName)
	require.Equal(t, alias1.SkillID, alias2.SkillID)

	// Error Case: Not Found
	nonExistentAlias, err := testQueries.GetSkillAlias(context.Background(), util.RandomString(10))
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.Empty(t, nonExistentAlias)
}

////////////////////////////////////////////////////////////////////////

func TestUpdateSkillAlias(t *testing.T) {
	alias1 := createRandomSkillAlias(t)

	// Create a new skill to re-map the alias to.
	skill2 := createRandomSkill(t)

	arg := UpdateSkillAliasParams{
		AliasName: alias1.AliasName,
		SkillID:   skill2.ID,
	}

	updatedAlias, err := testQueries.UpdateSkillAlias(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedAlias)

	// Verify the alias name remains the same, but the skill ID has changed.
	require.Equal(t, alias1.AliasName, updatedAlias.AliasName)
	require.Equal(t, skill2.ID, updatedAlias.SkillID)
}

////////////////////////////////////////////////////////////////////////

func TestDeleteSkillAlias(t *testing.T) {
	alias1 := createRandomSkillAlias(t)

	// Delete the alias
	err := testQueries.DeleteSkillAlias(context.Background(), alias1.AliasName)
	require.NoError(t, err)

	// Verify it's gone
	alias2, err := testQueries.GetSkillAlias(context.Background(), alias1.AliasName)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.Empty(t, alias2)
}

////////////////////////////////////////////////////////////////////////

func TestListSkillAliases(t *testing.T) {
	// Create 10 aliases for pagination testing.
	for range 10 {
		createRandomSkillAlias(t)
	}

	arg := ListSkillAliasesParams{
		Limit:  5,
		Offset: 5,
	}

	aliases, err := testQueries.ListSkillAliases(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, aliases, 5)

	for _, alias := range aliases {
		require.NotEmpty(t, alias)
	}
}

////////////////////////////////////////////////////////////////////////

func TestListAliasesForSkill(t *testing.T) {
	// Success Case
	skill := createRandomSkill(t)
	// Create 5 aliases specifically for this skill.
	for range 5 {
		arg := CreateSkillAliasParams{
			AliasName: util.RandomString(7),
			SkillID:   skill.ID,
		}
		_, err := testQueries.CreateSkillAlias(context.Background(), arg)
		require.NoError(t, err)
	}

	// Create another skill and alias to ensure we don't fetch them.
	createRandomSkillAlias(t)

	aliases, err := testQueries.ListAliasesForSkill(context.Background(), skill.ID)
	require.NoError(t, err)
	require.Len(t, aliases, 5)

	for _, alias := range aliases {
		require.NotEmpty(t, alias)
		require.Equal(t, skill.ID, alias.SkillID)
	}

	// Success Case: No Aliases Found
	skillWithNoAliases := createRandomSkill(t)
	aliases, err = testQueries.ListAliasesForSkill(context.Background(), skillWithNoAliases.ID)
	require.NoError(t, err)
	require.Empty(t, aliases) // Or require.Len(t, aliases, 0)
}
