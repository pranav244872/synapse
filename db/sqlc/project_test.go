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

// TestCreateProject tests the creation of a project, including edge cases.
func TestCreateProject(t *testing.T) {
	t.Run("SuccessWithDescription", func(t *testing.T) {
		// The helper function createRandomProject already provides a thorough test
		// for the happy path of creating a project with a valid description.
		createRandomProject(t)
	})

	t.Run("SuccessWithNullDescription", func(t *testing.T) {
		team := createRandomTeam(t)
		arg := CreateProjectParams{
			ProjectName: util.RandomProjectName(),
			TeamID:      team.ID,
			Description: pgtype.Text{Valid: false}, // Explicitly set description to NULL
		}

		project, err := testQueries.CreateProject(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, project)

		require.Equal(t, arg.ProjectName, project.ProjectName)
		require.Equal(t, arg.TeamID, project.TeamID)
		require.False(t, project.Description.Valid, "Description should be NULL")
	})

	t.Run("FailureWithNonExistentTeam", func(t *testing.T) {
		// Test for foreign key constraint violation
		arg := CreateProjectParams{
			ProjectName: util.RandomProjectName(),
			TeamID:      -1, // A team with ID -1 should not exist
			Description: pgtype.Text{Valid: false},
		}

		project, err := testQueries.CreateProject(context.Background(), arg)
		require.Error(t, err)
		// The specific error will depend on your RDBMS, but it should be a foreign key violation.
		require.Contains(t, err.Error(), "foreign key constraint")
		require.Empty(t, project)
	})
}

////////////////////////////////////////////////////////////////////////

// TestGetProject tests retrieving a single project.
func TestGetProject(t *testing.T) {
	project1 := createRandomProject(t)

	t.Run("Success", func(t *testing.T) {
		project2, err := testQueries.GetProject(context.Background(), project1.ID)
		require.NoError(t, err)
		require.NotEmpty(t, project2)

		require.Equal(t, project1.ID, project2.ID)
		require.Equal(t, project1.ProjectName, project2.ProjectName)
		require.Equal(t, project1.TeamID, project2.TeamID)
		require.Equal(t, project1.Description, project2.Description)
	})

	t.Run("NotFound", func(t *testing.T) {
		// Test retrieving a project with an ID that does not exist.
		project, err := testQueries.GetProject(context.Background(), -1)
		require.Error(t, err)
		require.ErrorIs(t, err, pgx.ErrNoRows, "Expected no rows error for non-existent ID")
		require.Empty(t, project)
	})
}

////////////////////////////////////////////////////////////////////////

// TestUpdateProject covers various update scenarios, including partial updates.
func TestUpdateProject(t *testing.T) {

	t.Run("SuccessFullUpdate", func(t *testing.T) {
		// Create a fresh project specifically for this test case.
		project1 := createRandomProject(t)

		// Update both name and description
		arg := UpdateProjectParams{
			ID:          project1.ID,
			TeamID:      project1.TeamID,
			ProjectName: util.RandomProjectName(),
			Description: pgtype.Text{String: "A new updated description.", Valid: true},
		}

		updatedProject, err := testQueries.UpdateProject(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, updatedProject)

		require.Equal(t, project1.ID, updatedProject.ID)
		require.Equal(t, project1.TeamID, updatedProject.TeamID)
		require.Equal(t, arg.ProjectName, updatedProject.ProjectName)
		require.Equal(t, arg.Description.String, updatedProject.Description.String)
		require.NotEqual(t, project1.ProjectName, updatedProject.ProjectName)
		require.NotEqual(t, project1.Description.String, updatedProject.Description.String)
	})

	t.Run("SuccessPartialUpdate_NameOnly", func(t *testing.T) {
		// Create a fresh project specifically for this test case.
		project1 := createRandomProject(t)

		// SQL uses COALESCE, allowing for partial updates.
		// To test updating only the name, we set the description to be NULL (invalid).
		arg := UpdateProjectParams{
			ID:          project1.ID,
			TeamID:      project1.TeamID,
			ProjectName: util.RandomProjectName(),
			Description: pgtype.Text{Valid: false}, // This should keep the old description
		}

		updatedProject, err := testQueries.UpdateProject(context.Background(), arg)
		require.NoError(t, err)

		// Verify that the name changed but the description did not.
		require.Equal(t, arg.ProjectName, updatedProject.ProjectName)
		require.NotEqual(t, project1.ProjectName, updatedProject.ProjectName)
		require.Equal(t, project1.Description, updatedProject.Description)
	})

	t.Run("Failure_WrongTeamID", func(t *testing.T) {
		// Create a fresh project specifically for this test case.
		project1 := createRandomProject(t)

		// Attempting to update a project with the correct ID but wrong team ID
		// This tests the authorization aspect of the query (WHERE id = $1 AND team_id = $2)
		arg := UpdateProjectParams{
			ID:          project1.ID,
			TeamID:      -1, // Invalid team ID
			ProjectName: "This should not be updated",
		}

		project, err := testQueries.UpdateProject(context.Background(), arg)
		require.Error(t, err, "Update should fail if team ID doesn't match")
		require.ErrorIs(t, err, pgx.ErrNoRows, "Expected no rows error because WHERE clause failed")
		require.Empty(t, project)
	})
}

////////////////////////////////////////////////////////////////////////

// TestDeleteProject verifies deletion logic.
func TestDeleteProject(t *testing.T) {
	project1 := createRandomProject(t)

	t.Run("Success", func(t *testing.T) {
		err := testQueries.DeleteProject(context.Background(), project1.ID)
		require.NoError(t, err)

		// Verify that the project is actually gone.
		project2, err := testQueries.GetProject(context.Background(), project1.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, pgx.ErrNoRows)
		require.Empty(t, project2)
	})

	t.Run("NotFound", func(t *testing.T) {
		// Deleting a non-existent ID should not produce an error, it just affects 0 rows.
		err := testQueries.DeleteProject(context.Background(), -1)
		require.NoError(t, err)
	})
}

////////////////////////////////////////////////////////////////////////

// TestListProjects tests pagination.
func TestListProjects(t *testing.T) {
	// Create a known number of projects to ensure the table isn't empty.
	for range 5 {
		createRandomProject(t)
	}

	t.Run("Success_Pagination", func(t *testing.T) {
		arg := ListProjectsParams{
			Limit:  5,
			Offset: 0, // Start from the beginning
		}

		projects, err := testQueries.ListProjects(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, projects) // Should find the projects we just created
		require.Len(t, projects, 5)

		for _, project := range projects {
			require.NotEmpty(t, project)
		}
	})

	t.Run("EdgeCase_OffsetTooHigh", func(t *testing.T) {
		// 1. Get the actual, current count of all projects in the DB.
		count, err := testQueries.CountProjects(context.Background())
		require.NoError(t, err)
		require.Greater(t, count, int64(0)) // Ensure there's something to offset past

		// 2. Use the total count as the offset. This guarantees it's out of bounds.
		arg := ListProjectsParams{
			Limit:  5,
			Offset: int32(count),
		}

		projects, err := testQueries.ListProjects(context.Background(), arg)
		require.NoError(t, err)
		
		// 3. This assertion will now reliably pass.
		require.Empty(t, projects, "Should return an empty slice when offset is out of bounds")
	})
}

////////////////////////////////////////////////////////////////////////
//  Tests for Team-Scoped Queries
////////////////////////////////////////////////////////////////////////

// TestGetProjectByIDAndTeam tests the query that retrieves a project scoped to a team.
func TestGetProjectByIDAndTeam(t *testing.T) {
	project1 := createRandomProject(t) // Belongs to team project1.TeamID
	otherTeam := createRandomTeam(t)

	t.Run("Success", func(t *testing.T) {
		arg := GetProjectByIDAndTeamParams{
			ID:     project1.ID,
			TeamID: project1.TeamID,
		}
		project2, err := testQueries.GetProjectByIDAndTeam(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, project2)
		require.Equal(t, project1.ID, project2.ID)
	})

	t.Run("Failure_WrongTeam", func(t *testing.T) {
		// Use correct project ID but the ID of a different team.
		arg := GetProjectByIDAndTeamParams{
			ID:     project1.ID,
			TeamID: otherTeam.ID,
		}
		project, err := testQueries.GetProjectByIDAndTeam(context.Background(), arg)
		require.Error(t, err)
		require.ErrorIs(t, err, pgx.ErrNoRows)
		require.Empty(t, project)
	})
}

////////////////////////////////////////////////////////////////////////

// TestDeleteProjectByTeam tests the team-scoped delete.
func TestDeleteProjectByTeam(t *testing.T) {
	project1 := createRandomProject(t)
	otherTeam := createRandomTeam(t)

	t.Run("Failure_WrongTeam", func(t *testing.T) {
		// Attempt to delete with correct project ID but wrong team ID.
		arg := DeleteProjectByTeamParams{
			ID:     project1.ID,
			TeamID: otherTeam.ID,
		}

		err := testQueries.DeleteProjectByTeam(context.Background(), arg)
		// Exec does not return an error if no rows are affected.
		require.NoError(t, err)

		// CRITICAL: Verify the project was NOT deleted.
		projectStillExists, err := testQueries.GetProject(context.Background(), project1.ID)
		require.NoError(t, err)
		require.NotEmpty(t, projectStillExists)
	})

	t.Run("Success", func(t *testing.T) {
		// Delete with the correct project and team ID.
		arg := DeleteProjectByTeamParams{
			ID:     project1.ID,
			TeamID: project1.TeamID,
		}
		err := testQueries.DeleteProjectByTeam(context.Background(), arg)
		require.NoError(t, err)

		// Verify it's gone.
		_, err = testQueries.GetProject(context.Background(), project1.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, pgx.ErrNoRows)
	})
}

////////////////////////////////////////////////////////////////////////

// TestListProjectsByTeam tests listing projects scoped to a single team.
func TestListProjectsByTeam(t *testing.T) {
	// Setup: Create two teams. Add 5 projects to teamA and 3 to teamB.
	teamA := createRandomTeam(t)
	teamB := createRandomTeam(t)
	
	for range 5 {
		testQueries.CreateProject(context.Background(), CreateProjectParams{
			ProjectName: util.RandomProjectName(),
			TeamID: teamA.ID,
		})
	}
	for range 3 {
		testQueries.CreateProject(context.Background(), CreateProjectParams{
			ProjectName: util.RandomProjectName(),
			TeamID: teamB.ID,
		})
	}

	t.Run("Success_ListForTeamA", func(t *testing.T) {
		arg := ListProjectsByTeamParams{
			TeamID: teamA.ID,
			Limit:  10, // Limit greater than number of projects
			Offset: 0,
		}
		projects, err := testQueries.ListProjectsByTeam(context.Background(), arg)
		require.NoError(t, err)
		require.Len(t, projects, 5, "Should only list the 5 projects from Team A")

		// Verify all returned projects actually belong to Team A
		for _, p := range projects {
			require.Equal(t, teamA.ID, p.TeamID)
		}
	})

	t.Run("Success_Pagination", func(t *testing.T) {
		arg := ListProjectsByTeamParams{
			TeamID: teamA.ID,
			Limit:  3,
			Offset: 2,
		}
		projects, err := testQueries.ListProjectsByTeam(context.Background(), arg)
		require.NoError(t, err)
		require.Len(t, projects, 3, "Should retrieve 3 projects with offset 2")
	})

	t.Run("Success_TeamWithNoProjects", func(t *testing.T) {
		teamC := createRandomTeam(t)
		arg := ListProjectsByTeamParams{
			TeamID: teamC.ID,
			Limit:  5,
			Offset: 0,
		}
		projects, err := testQueries.ListProjectsByTeam(context.Background(), arg)
		require.NoError(t, err)
		require.Empty(t, projects, "Should return an empty slice for a team with no projects")
	})
}

////////////////////////////////////////////////////////////////////////
