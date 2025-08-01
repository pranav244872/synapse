package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pranav244872/synapse/util"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////

// TestCreateInvitation tests the creation of a new invitation.
func TestCreateInvitation(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// The createRandomInvitation helper function is assumed to contain
		// all necessary assertions for a successful creation.
		createRandomInvitation(t)
	})

	t.Run("Failure_NonExistentInviter", func(t *testing.T) {
		// Test for foreign key constraint violation on 'inviter_id'.
		team := createRandomTeam(t)
		arg := CreateInvitationParams{
			Email:           util.RandomEmail(),
			InvitationToken: util.RandomString(32),
			RoleToInvite:    UserRoleEngineer,
			InviterID:       -1, // This user ID should not exist.
			ExpiresAt:       pgtype.Timestamp{Time: time.Now().Add(time.Hour * 24), Valid: true},
			TeamID:          pgtype.Int8{Int64: team.ID, Valid: true},
		}

		invitation, err := testQueries.CreateInvitation(context.Background(), arg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "foreign key constraint", "Error should indicate a foreign key violation")
		require.Empty(t, invitation)
	})
}

////////////////////////////////////////////////////////////////////////

// TestGetInvitationByToken tests retrieving an invitation by its unique token.
func TestGetInvitationByToken(t *testing.T) {
	// Create a standard, valid invitation for use in sub-tests.
	invitation1 := createRandomInvitation(t)

	t.Run("Success", func(t *testing.T) {
		// Fetch the invitation using its token.
		invitation2, err := testQueries.GetInvitationByToken(context.Background(), invitation1.InvitationToken)
		require.NoError(t, err)
		require.NotEmpty(t, invitation2)

		// Assert that the retrieved data matches the original.
		require.Equal(t, invitation1.ID, invitation2.ID)
		require.Equal(t, invitation1.Email, invitation2.Email)
		require.Equal(t, invitation1.TeamID, invitation2.TeamID)
	})

	t.Run("NotFound", func(t *testing.T) {
		// Attempt to fetch an invitation with a token that does not exist.
		invitation, err := testQueries.GetInvitationByToken(context.Background(), util.RandomString(32))
		require.Error(t, err)
		require.ErrorIs(t, err, pgx.ErrNoRows)
		require.Empty(t, invitation)
	})

	t.Run("Failure_Expired", func(t *testing.T) {
		// Create an invitation that has already expired.
		expiredInvitation := createExpiredInvitation(t)

		// The query searches for non-expired invitations, so this should fail.
		invitation, err := testQueries.GetInvitationByToken(context.Background(), expiredInvitation.InvitationToken)
		require.Error(t, err)
		require.ErrorIs(t, err, pgx.ErrNoRows, "Should not find an expired invitation")
		require.Empty(t, invitation)
	})

	t.Run("Failure_StatusNotPending", func(t *testing.T) {
		// Create an invitation and immediately update its status.
		acceptedInvitation := createRandomInvitation(t)
		_, err := testQueries.UpdateInvitationStatus(context.Background(), UpdateInvitationStatusParams{
			ID:     acceptedInvitation.ID,
			Status: "accepted",
		})
		require.NoError(t, err)

		// The query only looks for 'pending' invitations, so this should fail.
		invitation, err := testQueries.GetInvitationByToken(context.Background(), acceptedInvitation.InvitationToken)
		require.Error(t, err)
		require.ErrorIs(t, err, pgx.ErrNoRows, "Should not find an invitation with 'accepted' status")
		require.Empty(t, invitation)
	})
}

////////////////////////////////////////////////////////////////////////

// TestGetInvitationByEmail tests retrieving a pending invitation by email.
func TestGetInvitationByEmail(t *testing.T) {
	// Create a standard, valid invitation for use in sub-tests.
	invitation1 := createRandomInvitation(t)

	t.Run("Success", func(t *testing.T) {
		invitation2, err := testQueries.GetInvitationByEmail(context.Background(), invitation1.Email)
		require.NoError(t, err)
		require.NotEmpty(t, invitation2)

		// Assert that the retrieved data matches the original.
		require.Equal(t, invitation1.ID, invitation2.ID)
		require.Equal(t, invitation1.InvitationToken, invitation2.InvitationToken)
	})

	t.Run("NotFound", func(t *testing.T) {
		// Attempt to fetch an invitation for an email that was never invited.
		invitation, err := testQueries.GetInvitationByEmail(context.Background(), util.RandomEmail())
		require.Error(t, err)
		require.ErrorIs(t, err, pgx.ErrNoRows)
		require.Empty(t, invitation)
	})

	t.Run("Failure_StatusNotPending", func(t *testing.T) {
		// Create an invitation and update its status to 'accepted'.
		acceptedInvitation := createRandomInvitation(t)
		_, err := testQueries.UpdateInvitationStatus(context.Background(), UpdateInvitationStatusParams{
			ID:     acceptedInvitation.ID,
			Status: "accepted",
		})
		require.NoError(t, err)

		// The query only finds 'pending' invitations, so this should not be found.
		invitation, err := testQueries.GetInvitationByEmail(context.Background(), acceptedInvitation.Email)
		require.Error(t, err)
		require.ErrorIs(t, err, pgx.ErrNoRows)
		require.Empty(t, invitation)
	})
}

////////////////////////////////////////////////////////////////////////

// TestUpdateInvitationStatus tests changing the status of an invitation.
func TestUpdateInvitationStatus(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		invitation1 := createRandomInvitation(t)
		require.Equal(t, "pending", invitation1.Status) // Verify initial state.

		// Define and execute the update.
		arg := UpdateInvitationStatusParams{
			ID:     invitation1.ID,
			Status: "accepted",
		}
		updatedInvitation, err := testQueries.UpdateInvitationStatus(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, updatedInvitation)

		// Assert that the status was updated and other fields remain unchanged.
		require.Equal(t, arg.Status, updatedInvitation.Status)
		require.Equal(t, invitation1.ID, updatedInvitation.ID)
		require.Equal(t, invitation1.Email, updatedInvitation.Email)
		require.Equal(t, invitation1.TeamID, updatedInvitation.TeamID)
	})

	t.Run("NotFound", func(t *testing.T) {
		// Attempt to update an invitation with a non-existent ID.
		arg := UpdateInvitationStatusParams{
			ID:     -1,
			Status: "accepted",
		}
		invitation, err := testQueries.UpdateInvitationStatus(context.Background(), arg)
		require.Error(t, err)
		require.ErrorIs(t, err, pgx.ErrNoRows)
		require.Empty(t, invitation)
	})
}

////////////////////////////////////////////////////////////////////////
