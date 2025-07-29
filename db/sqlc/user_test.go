package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pranav244872/synapse/util"
)

////////////////////////////////////////////////////////////////////////

func TestCreateUser(t *testing.T) {
	user, password := createRandomUser(t)
	err := util.CheckPasswordHash(password, user.PasswordHash)
	require.NoError(t, err)
}

////////////////////////////////////////////////////////////////////////

func TestGetUser(t *testing.T) {
	// Create a new random user
	user1, password := createRandomUser(t)
	require.NotEmpty(t, user1)

	// Retrieve the user from the DB
	user2, err := testQueries.GetUser(context.Background(), user1.ID)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.Name, user2.Name)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.TeamID, user2.TeamID)
	require.Equal(t, user1.Availability, user2.Availability)
	require.Equal(t, user1.PasswordHash, user2.PasswordHash)
	require.Equal(t, user1.Role, user2.Role)

	// Check that original password can be verified agains the retrieved user's hash
	require.NoError(t, util.CheckPasswordHash(password, user2.PasswordHash))
}

////////////////////////////////////////////////////////////////////////

func TestGetUserByEmail(t *testing.T) {
	// Create a new random user
	user1, _ := createRandomUser(t)
	require.NotEmpty(t, user1)

	// Retrieve the user from the DB using their email
	user2, err := testQueries.GetUserByEmail(context.Background(), user1.Email)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.Name, user2.Name)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.TeamID, user2.TeamID)
	require.Equal(t, user1.Availability, user2.Availability)
}

////////////////////////////////////////////////////////////////////////

func TestUpdateUser(t *testing.T) {
	user1, _ := createRandomUser(t)
	team2 := createRandomTeam(t)

	arg := UpdateUserParams{
		ID: user1.ID,
		Name: pgtype.Text{
			String: util.RandomName(),
			Valid:  true,
		},
		TeamID: pgtype.Int8{
			Int64: team2.ID,
			Valid: true,
		},
		// Availability is not updated, testing coalesce
	}

	updatedUser, err := testQueries.UpdateUser(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, updatedUser)

	require.Equal(t, user1.ID, updatedUser.ID)
	require.Equal(t, arg.Name, updatedUser.Name)
	require.Equal(t, user1.Email, updatedUser.Email) // Email should not change
	require.Equal(t, user1.Role, updatedUser.Role)
	require.Equal(t, user1.PasswordHash, updatedUser.PasswordHash)
	require.Equal(t, arg.TeamID, updatedUser.TeamID)
	require.Equal(t, user1.Availability, updatedUser.Availability) // Availability should not change
}

////////////////////////////////////////////////////////////////////////

func TestDeleteUser(t *testing.T) {
	// Create a user to delete
	user1, _ := createRandomUser(t)

	// Delete the user
	err := testQueries.DeleteUser(context.Background(), user1.ID)
	require.NoError(t, err)

	// Try to retrieve the deleted user
	user2, err := testQueries.GetUser(context.Background(), user1.ID)

	// Assert that an error is returned and the user object is empty
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.Empty(t, user2)
}

////////////////////////////////////////////////////////////////////////

func TestListUsers(t *testing.T) {
	// Create 10 users for testing pagination
	for range 10 {
		createRandomUser(t)
	}

	arg := ListUsersParams{
		Limit:  5,
		Offset: 5,
	}

	users, err := testQueries.ListUsers(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.Len(t, users, 5)

	for _, user := range users {
		require.NotEmpty(t, user)
	}
}

////////////////////////////////////////////////////////////////////////

func TestListUsersByTeam(t *testing.T) {
	team := createRandomTeam(t)

	// Create 6 users for the specific team
	for range 6 {
		hashedPassword, err := util.HashPassword(util.RandomString(10))
		require.NoError(t, err)

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

		_, err = testQueries.CreateUser(context.Background(), arg)
		require.NoError(t, err)
	}

	arg := ListUsersByTeamParams{
		TeamID: pgtype.Int8{
			Int64: team.ID,
			Valid: true,
		},
		Limit:  5,
		Offset: 0,
	}

	users, err := testQueries.ListUsersByTeam(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.Len(t, users, 5)

	for _, user := range users {
		require.NotEmpty(t, user)
		require.Equal(t, team.ID, user.TeamID.Int64)
	}
}
