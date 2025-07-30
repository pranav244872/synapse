package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pranav244872/synapse/util"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////

func TestCreateInvitation(t *testing.T) {
	// The createRandomInvitation helper function already contains
	// all the necessary checks and assertions.
	createRandomInvitation(t)
}

////////////////////////////////////////////////////////////////////////

func TestGetInvitationByToken(t *testing.T) {
	// Create a random invitation to be fetched
	invitation1 := createRandomInvitation(t)

	// Fetch the invitation using its token
	invitation2, err := testQueries.GetInvitationByToken(context.Background(), invitation1.InvitationToken)

	// Assertions for a successful fetch
	require.NoError(t, err)
	require.NotEmpty(t, invitation2)

	// Compare the fetched invitation with the original one
	require.Equal(t, invitation1.ID, invitation2.ID)
	require.Equal(t, invitation1.Email, invitation2.Email)
	require.Equal(t, invitation1.InvitationToken, invitation2.InvitationToken)

	// Test case for a non-existent token
	invitation3, err := testQueries.GetInvitationByToken(context.Background(), util.RandomString(32))
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.Empty(t, invitation3)
}

////////////////////////////////////////////////////////////////////////

func TestGetInvitationByEmail(t *testing.T) {
	// Create a random invitation to be fetched
	invitation1 := createRandomInvitation(t)

	// Fetch the invitation using its email
	invitation2, err := testQueries.GetInvitationByEmail(context.Background(), invitation1.Email)

	// Assertions for a successful fetch
	require.NoError(t, err)
	require.NotEmpty(t, invitation2)

	// Compare the fetched invitation with the original one
	require.Equal(t, invitation1.ID, invitation2.ID)
	require.Equal(t, invitation1.Email, invitation2.Email)
	require.Equal(t, invitation1.InvitationToken, invitation2.InvitationToken)
	require.Equal(t, invitation1.RoleToInvite, invitation2.RoleToInvite)
	require.Equal(t, invitation1.InviterID, invitation2.InviterID)
	require.Equal(t, invitation1.Status, invitation2.Status)
	require.WithinDuration(t, invitation1.ExpiresAt.Time, invitation2.ExpiresAt.Time, time.Second)

	// Test case for a non-existent email
	invitation3, err := testQueries.GetInvitationByEmail(context.Background(), util.RandomEmail())
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.Empty(t, invitation3)
}

/////////////////////////////////////////////////////////////////////////

func TestUpdateInvitationStatus(t *testing.T) {
	// Create an invitation to be updated
	invitation1 := createRandomInvitation(t)
	require.Equal(t, "pending", invitation1.Status) // Check initial status

	// Define update parameters
	arg := UpdateInvitationStatusParams{
		ID:     invitation1.ID,
		Status: "accepted",
	}

	// Update the invitation status
	updatedInvitation, err := testQueries.UpdateInvitationStatus(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, updatedInvitation)

	// Check that the status was updated
	require.Equal(t, arg.Status, updatedInvitation.Status)

	// Check that other core fields remained the same
	require.Equal(t, invitation1.ID, updatedInvitation.ID)
	require.Equal(t, invitation1.Email, updatedInvitation.Email)
	require.Equal(t, invitation1.InvitationToken, updatedInvitation.InvitationToken)
}

