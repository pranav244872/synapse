package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pranav244872/synapse/util"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////

func TestCreateTeam(t *testing.T) {
	// Case 1: Create a team with a manager
	manager, _ := createRandomUser(t)
	teamName1 := util.RandomName()
	arg1 := CreateTeamParams{
		TeamName:  teamName1,
		ManagerID: pgtype.Int8{Int64: manager.ID, Valid: true},
	}
	team1, err := testQueries.CreateTeam(context.Background(), arg1)
	require.NoError(t, err)
	require.NotEmpty(t, team1)
	require.Equal(t, teamName1, team1.TeamName)
	require.True(t, team1.ManagerID.Valid)
	require.Equal(t, manager.ID, team1.ManagerID.Int64)

	// Case 2: Create a team without a manager (using the helper)
	team2 := createRandomTeam(t)
	require.NotEmpty(t, team2)
	require.False(t, team2.ManagerID.Valid)
}

////////////////////////////////////////////////////////////////////////

func TestGetTeam(t *testing.T) {
	team1 := createRandomTeam(t) // Creates a team with a NULL manager_id
	require.NotEmpty(t, team1)

	team2, err := testQueries.GetTeam(context.Background(), team1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, team2)

	require.Equal(t, team1.ID, team2.ID)
	require.Equal(t, team1.TeamName, team2.TeamName)
	// Assert that the manager ID is also correctly fetched.
	require.Equal(t, team1.ManagerID, team2.ManagerID)
}

////////////////////////////////////////////////////////////////////////

func TestUpdateTeam(t *testing.T) {
	team1 := createRandomTeam(t) // Starts with no manager
	require.NotEmpty(t, team1)

	newManager, _ := createRandomUser(t)
	newName := util.RandomName()

	// UpdateTeamParams now uses pgtype for optional fields.
	arg := UpdateTeamParams{
		ID:        team1.ID,
		TeamName:  pgtype.Text{String: newName, Valid: true},
		ManagerID: pgtype.Int8{Int64: newManager.ID, Valid: true},
	}

	team2, err := testQueries.UpdateTeam(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, team2)

	// Assert that all fields were updated correctly.
	require.Equal(t, team1.ID, team2.ID)
	require.Equal(t, newName, team2.TeamName)
	require.True(t, team2.ManagerID.Valid)
	require.Equal(t, newManager.ID, team2.ManagerID.Int64)
}

////////////////////////////////////////////////////////////////////////

func TestDeleteTeam(t *testing.T) {
	// Create a team to delete
	team1 := createRandomTeam(t)

	// Delete the team
	err := testQueries.DeleteTeam(context.Background(), team1.ID)
	require.NoError(t, err)

	// Try to retrieve the deleted team
	team2, err := testQueries.GetTeam(context.Background(), team1.ID)

	// Assert that an error is returned and the team object is empty
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.Empty(t, team2)
}

////////////////////////////////////////////////////////////////////////

func TestListTeams(t *testing.T) {
	// Create 10 teams for pagination testing
	for range 10 {
		createRandomTeam(t)
	}

	arg := ListTeamsParams{
		Limit:  5,
		Offset: 5,
	}

	teams, err := testQueries.ListTeams(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.Len(t, teams, 5)

	for _, team := range teams {
		require.NotEmpty(t, team)
	}
}

////////////////////////////////////////////////////////////////////////

func TestGetTeamByManagerID(t *testing.T) {
	// 1. Create a team with a known manager.
	team1, manager := createRandomTeamWithManager(t)

	// 2. Fetch the team using the manager's ID.
	team2, err := testQueries.GetTeamByManagerID(context.Background(), team1.ManagerID)
	require.NoError(t, err)
	require.NotEmpty(t, team2)

	// 3. Assert the correct team was returned.
	require.Equal(t, team1.ID, team2.ID)
	require.Equal(t, manager.ID, team2.ManagerID.Int64)
}

////////////////////////////////////////////////////////////////////////

func TestListTeamsWithManagers(t *testing.T) {
	// 1. Create 5 teams, some with managers and some without.
	for i := 0; i < 3; i++ {
		createRandomTeamWithManager(t)
	}
	createRandomTeam(t)
	createRandomTeam(t)

	// 2. List all teams with their manager details.
	teams, err := testQueries.ListTeamsWithManagers(context.Background(), ListTeamsWithManagersParams{
		Limit:  5,
		Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, teams, 5)

	// 3. Verify the results.
	for _, team := range teams {
		require.NotEmpty(t, team)
		if team.ManagerID.Valid {
			// If a manager exists, their name and email should be populated.
			require.True(t, team.ManagerName.Valid)
			require.True(t, team.ManagerEmail.Valid)
		} else {
			// If no manager exists, their name and email should be NULL.
			require.False(t, team.ManagerName.Valid)
			require.False(t, team.ManagerEmail.Valid)
		}
	}
} 

////////////////////////////////////////////////////////////////////////
