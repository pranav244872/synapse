package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pranav244872/synapse/util"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////

func TestCreateProject(t *testing.T) {
	createRandomProject(t)
}

////////////////////////////////////////////////////////////////////////

func TestGetProject(t *testing.T) {
	// Create a random project
	project1 := createRandomProject(t)
	require.NotEmpty(t, project1)

	// Retrieve the same project from the database
	project2, err := testQueries.GetProject(context.Background(), project1.ID)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, project2)

	require.Equal(t, project1.ID, project2.ID)
	require.Equal(t, project1.ProjectName, project2.ProjectName)
}

////////////////////////////////////////////////////////////////////////

func TestUpdateProject(t *testing.T) {
	// Create a random project
	project1 := createRandomProject(t)
	require.NotEmpty(t, project1)

	// Prepare parameters for the update
	arg := UpdateProjectParams{
		ID:          project1.ID,
		ProjectName: util.RandomProjectName(),
	}

	// Update the project
	project2, err := testQueries.UpdateProject(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, project2)

	require.Equal(t, project1.ID, project2.ID)
	require.Equal(t, arg.ProjectName, project2.ProjectName)
	require.NotEqual(t, project1.ProjectName, project2.ProjectName)
}

////////////////////////////////////////////////////////////////////////

func TestDeleteProject(t *testing.T) {
	// Create a project to delete
	project1 := createRandomProject(t)

	// Delete the project
	err := testQueries.DeleteProject(context.Background(), project1.ID)
	require.NoError(t, err)

	// Try to retrieve the deleted project
	project2, err := testQueries.GetProject(context.Background(), project1.ID)

	// Assert that an error is returned and the project object is empty
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.Empty(t, project2)
}

////////////////////////////////////////////////////////////////////////

func TestListProjects(t *testing.T) {
	// Create 10 projects for pagination testing
	for range 10 {
		createRandomProject(t)
	}

	arg := ListProjectsParams{
		Limit:  5,
		Offset: 5,
	}

	projects, err := testQueries.ListProjects(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.Len(t, projects, 5)

	for _, project := range projects {
		require.NotEmpty(t, project)
	}
}

////////////////////////////////////////////////////////////////////////
