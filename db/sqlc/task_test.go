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

func TestCreateTask(t *testing.T) {
	createRandomTask(t)
}

////////////////////////////////////////////////////////////////////////

func TestGetTask(t *testing.T) {
	// Create a random task
	task1 := createRandomTask(t)
	require.NotEmpty(t, task1)

	// Retrieve the same task from the database
	task2, err := testQueries.GetTask(context.Background(), task1.ID)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, task2)

	require.Equal(t, task1.ID, task2.ID)
	require.Equal(t, task1.ProjectID, task2.ProjectID)
	require.Equal(t, task1.AssigneeID, task2.AssigneeID)
	require.Equal(t, task1.Title, task2.Title)
	require.Equal(t, task1.Description, task2.Description)
	require.Equal(t, task1.Status, task2.Status)
	require.Equal(t, task1.Priority, task2.Priority)
	require.WithinDuration(t, task1.CreatedAt.Time, task2.CreatedAt.Time, time.Second)
}

////////////////////////////////////////////////////////////////////////

func TestUpdateTask(t *testing.T) {
	// Create an initial task
	task1 := createRandomTask(t)
	require.NotEmpty(t, task1)

	// Prepare new data for a partial update
	newUser := createRandomUser(t)
	// Use the provided utility function to get a valid random status
	newStatus := TaskStatus(util.RandomStatus()) 
	
	arg := UpdateTaskParams{
		ID:    task1.ID,
		Title: pgtype.Text{String: util.RandomTaskTitle(), Valid: true},
		Status: NullTaskStatus{
			TaskStatus: newStatus,
			Valid:      true,
		},
		AssigneeID: pgtype.Int8{Int64: newUser.ID, Valid: true},
	}

	// Run the update query
	task2, err := testQueries.UpdateTask(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.NotEmpty(t, task2)

	// Assert updated fields are correct
	require.Equal(t, task1.ID, task2.ID)
	require.Equal(t, arg.Title.String, task2.Title)
	require.Equal(t, arg.Status.TaskStatus, task2.Status)
	require.Equal(t, arg.AssigneeID.Int64, task2.AssigneeID.Int64)

	// Assert non-updated fields remain unchanged (testing coalesce)
	require.Equal(t, task1.ProjectID.Int64, task2.ProjectID.Int64)
	require.Equal(t, task1.Description.String, task2.Description.String)
	require.Equal(t, task1.Priority, task2.Priority)
}

////////////////////////////////////////////////////////////////////////

func TestDeleteTask(t *testing.T) {
	// Create a task to delete
	task1 := createRandomTask(t)

	// Delete the task
	err := testQueries.DeleteTask(context.Background(), task1.ID)
	require.NoError(t, err)

	// Try to retrieve the deleted task
	task2, err := testQueries.GetTask(context.Background(), task1.ID)

	// Assert that an error is returned and the task object is empty
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.Empty(t, task2)
}

////////////////////////////////////////////////////////////////////////

func TestListTasks(t *testing.T) {
	// Create 10 tasks for pagination testing
	for range 10 {
		createRandomTask(t)
	}

	arg := ListTasksParams{
		Limit:  5,
		Offset: 5,
	}

	tasks, err := testQueries.ListTasks(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.Len(t, tasks, 5)

	for _, task := range tasks {
		require.NotEmpty(t, task)
	}
}

////////////////////////////////////////////////////////////////////////

func TestListTasksByProject(t *testing.T) {
	project := createRandomProject(t)

	// Create 6 tasks for this specific project
	for range 6 {
		// createRandomTask creates a new project each time, so we create one manually
		assignee := createRandomUser(t)
		arg := CreateTaskParams{
			ProjectID:  pgtype.Int8{Int64: project.ID, Valid: true},
			Title:      util.RandomTaskTitle(),
			Status:     TaskStatus(util.RandomStatus()),
			Priority:   TaskPriority(util.RandomPriority()),
			AssigneeID: pgtype.Int8{Int64: assignee.ID, Valid: true},
		}
		_, err := testQueries.CreateTask(context.Background(), arg)
		require.NoError(t, err)
	}

	arg := ListTasksByProjectParams{
		ProjectID: pgtype.Int8{Int64: project.ID, Valid: true},
		Limit:     5,
		Offset:    0,
	}

	tasks, err := testQueries.ListTasksByProject(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.Len(t, tasks, 5)

	for _, task := range tasks {
		require.NotEmpty(t, task)
		require.Equal(t, project.ID, task.ProjectID.Int64)
	}
}

////////////////////////////////////////////////////////////////////////

func TestListTasksByAssignee(t *testing.T) {
	assignee := createRandomUser(t)

	// Create 6 tasks for this specific assignee
	for range 6 {
		// createRandomTask creates a new user each time, so we create one manually
		project := createRandomProject(t)
		arg := CreateTaskParams{
			ProjectID:  pgtype.Int8{Int64: project.ID, Valid: true},
			Title:      util.RandomTaskTitle(),
			Status:     TaskStatus(util.RandomStatus()),
			Priority:   TaskPriority(util.RandomPriority()),
			AssigneeID: pgtype.Int8{Int64: assignee.ID, Valid: true},
		}
		_, err := testQueries.CreateTask(context.Background(), arg)
		require.NoError(t, err)
	}

	arg := ListTasksByAssigneeParams{
		AssigneeID: pgtype.Int8{Int64: assignee.ID, Valid: true},
		Limit:      5,
		Offset:     0,
	}

	tasks, err := testQueries.ListTasksByAssignee(context.Background(), arg)

	// Assertions
	require.NoError(t, err)
	require.Len(t, tasks, 5)

	for _, task := range tasks {
		require.NotEmpty(t, task)
		require.Equal(t, assignee.ID, task.AssigneeID.Int64)
	}
}

////////////////////////////////////////////////////////////////////////
