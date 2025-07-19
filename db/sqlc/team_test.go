package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pranav244872/synapse/util"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////

func TestCreateTeam(t *testing.T) {
	createRandomTeam(t)
}

////////////////////////////////////////////////////////////////////////

func TestGetTeam(t *testing.T) {
	// Create a random team
	team1 := createRandomTeam(t)
	require.NotEmpty(t, team1)

	// Retrieve the same team from the database
	team2, err := testQueries.GetTeam(context.Background(), team1.ID)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, team2)

	require.Equal(t, team1.ID, team2.ID)
	require.Equal(t, team1.TeamName, team2.TeamName)
}

////////////////////////////////////////////////////////////////////////

func TestUpdateTeam(t *testing.T) {
	// Create a random team
	team1 := createRandomTeam(t)
	require.NotEmpty(t, team1)

	// Prepare parameters for the update
	arg := UpdateTeamParams{
		ID:       team1.ID,
		TeamName: util.RandomName(),
	}

	// Update the team
	team2, err := testQueries.UpdateTeam(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, team2)

	require.Equal(t, team1.ID, team2.ID)
	require.Equal(t, arg.TeamName, team2.TeamName)
	require.NotEqual(t, team1.TeamName, team2.TeamName)
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
