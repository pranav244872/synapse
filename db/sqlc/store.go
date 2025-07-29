package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides all functions to execute db queries and transactions.
type Store struct {
	*Queries
	dbpool *pgxpool.Pool
}

// NewStore creates a new Store.
func NewStore(dbpool *pgxpool.Pool) *Store {
	return &Store{
		dbpool:  dbpool,
		Queries: New(dbpool),
	}
}

// execTx executes a function within a database transaction.
func (s *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.dbpool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // Rollback is a no-op if the transaction has been committed.

	q := New(tx)
	err = fn(q)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

////////////////////////////////////////////////////////////////////////
// Transactional Methods
////////////////////////////////////////////////////////////////////////

// OnboardNewUserTxParams contains the parameters for the OnboardNewUserWithSkills transaction.
type OnboardNewUserTxParams struct {
    CreateUserParams      CreateUserParams
    SkillsWithProficiency map[string]ProficiencyLevel // e.g., {"Go": "expert", "PostgreSQL": "intermediate"}
}

// OnboardNewUserTxResult contains the result of the OnboardNewUserWithSkills transaction.
type OnboardNewUserTxResult struct {
	User       User
	UserSkills []UserSkill
}

// OnboardNewUserWithSkills orchestrates a complex transaction to create a user and populate their profile
// by extracting skills from their resume.
//
// It first calls the skillz.Processor to get a list of skills and proficiencies. Then, within a single
// database transaction, it:
// 1. Creates the user record.
// 2. Resolves all extracted skills, batch-creating any that don't exist.
// 3. Links each skill to the new user with the estimated proficiency.
func (s *Store) OnboardNewUserWithSkills(
	ctx context.Context,
	arg OnboardNewUserTxParams,
) (OnboardNewUserTxResult, error) {
	var result OnboardNewUserTxResult

	// Step 2: Execute the database operations within a single transaction.
	err := s.execTx(ctx, func(q *Queries) error {
		// Step 1: Create the user.
		createdUser, err := q.CreateUser(ctx, arg.CreateUserParams)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		result.User = createdUser

		// Step 2: Check if there are any skills to process
		if len(arg.SkillsWithProficiency) == 0 {
			return nil // No skills to process, transaction is complete.
		}

		// Step 3: Get the names from the map's key to resolve them.
		skillNames := make([]string, 0, len(arg.SkillsWithProficiency))
		for name := range arg.SkillsWithProficiency {
			skillNames = append(skillNames, name)
		}

		// Now, build the map using the definitive list of names.
		skillMap, err := s._resolveSkills(ctx, q, skillNames)
		if err != nil {
			return err
		}

		for name, skill := range skillMap {
			proficiency := arg.SkillsWithProficiency[name]
			userSkill, linkErr := q.AddSkillToUser(ctx, AddSkillToUserParams{
				UserID:      createdUser.ID,
				SkillID:     skill.ID,
				Proficiency: proficiency, // Use the proficiency from the input map
			})
			if linkErr != nil {
				return fmt.Errorf("failed to add skill '%s' to user: %w", name, linkErr)
			}
			result.UserSkills = append(result.UserSkills, userSkill)
		}

		return nil
	})

	return result, err
}

////////////////////////////////////////////////////////////////////////

// ProcessNewTaskTxParams include the pre-processed list of required skills. 
type ProcessNewTaskTxParams struct {
	CreateTaskParams    CreateTaskParams
	RequiredSkillNames []string
}

// ProcessNewTaskTxResult contains the result of the ProcessNewTask transaction.
type ProcessNewTaskTxResult struct {
	Task               Task
	TaskRequiredSkills []TaskRequiredSkill
}

// ProcessNewTask creates a task and automatically links required skills extracted from its description.
//
// It first calls the skillz.Processor to normalize skills from the task description. Then, within
// a single database transaction, it:
// 1. Creates the task record.
// 2. Resolves all extracted skills, batch-creating any that don't exist.
// 3. Links each required skill to the new task.
func (s *Store) ProcessNewTask(
	ctx context.Context,
	arg ProcessNewTaskTxParams,
) (ProcessNewTaskTxResult, error) {
	var result ProcessNewTaskTxResult

	// Step 2: Execute database writes in a transaction.
	err := s.execTx(ctx, func(q *Queries) error {
		// Create the task.
		createdTask, err := q.CreateTask(ctx, arg.CreateTaskParams)
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}
		result.Task = createdTask

		if len(arg.RequiredSkillNames) == 0 {
			return nil // No skills to process.
		}

		// Resolve skill IDs, creating any new skills in the process.
		skillMap, err := s._resolveSkills(ctx, q, arg.RequiredSkillNames)
		if err != nil {
			return err
		}

		// Link all required skills to the task.
		for _, skill := range skillMap {
			requiredSkill, linkErr := q.AddSkillToTask(ctx, AddSkillToTaskParams{
				TaskID:  createdTask.ID,
				SkillID: skill.ID,
			})
			if linkErr != nil {
				return fmt.Errorf("failed to link skill '%s' to task: %w", skill.SkillName, linkErr)
			}
			result.TaskRequiredSkills = append(result.TaskRequiredSkills, requiredSkill)
		}

		return nil
	})

	return result, err
}

////////////////////////////////////////////////////////////////////////

// AssignTaskToUserTxParams contains the parameters for assigning a task.
type AssignTaskToUserTxParams struct {
	TaskID int64
	UserID int64
}

// AssignTaskToUserTxResult contains the updated task and user from the assignment.
type AssignTaskToUserTxResult struct {
	Task Task
	User User
}

// AssignTaskToUser assigns a task to a user in a transaction. It updates the task's assignee
// and status ('in_progress'), and sets the user's availability to 'busy'.
func (s *Store) AssignTaskToUser(
	ctx context.Context,
	arg AssignTaskToUserTxParams,
) (AssignTaskToUserTxResult, error) {
	var result AssignTaskToUserTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		var err error
		// Update task to set assignee and status.
		result.Task, err = q.UpdateTask(ctx, UpdateTaskParams{
			ID:         arg.TaskID,
			AssigneeID: pgtype.Int8{Int64: arg.UserID, Valid: true},
			Status:     NullTaskStatus{TaskStatus: "in_progress", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update task assignment: %w", err)
		}

		// Update user's availability to 'busy'.
		result.User, err = q.UpdateUser(ctx, UpdateUserParams{
			ID:           arg.UserID,
			Availability: NullAvailabilityStatus{AvailabilityStatus: "busy", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update user availability: %w", err)
		}

		return nil
	})

	return result, err
}

////////////////////////////////////////////////////////////////////////

// CompleteTaskTxParams contains the parameters for completing a task.
type CompleteTaskTxParams struct {
	TaskID int64
}

// CompleteTask marks a task as 'done' and resets the assignee's availability to 'available'.
// This is a critical transaction for maintaining data consistency.
func (s *Store) CompleteTask(ctx context.Context, arg CompleteTaskTxParams) error {
	return s.execTx(ctx, func(q *Queries) error {
		// First, get the task to find out who the assignee is.
		task, err := q.GetTask(ctx, arg.TaskID)
		if err != nil {
			return fmt.Errorf("failed to get task for completion: %w", err)
		}

		// Mark the task as 'done' and set its completion timestamp.
		_, err = q.UpdateTask(ctx, UpdateTaskParams{
			ID:          arg.TaskID,
			Status:      NullTaskStatus{TaskStatus: "done", Valid: true},
			CompletedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to mark task as done: %w", err)
		}

		// If the task had an assignee, make them available again.
		if task.AssigneeID.Valid {
			_, err = q.UpdateUser(ctx, UpdateUserParams{
				ID:           task.AssigneeID.Int64,
				Availability: NullAvailabilityStatus{AvailabilityStatus: "available", Valid: true},
			})
			if err != nil {
				return fmt.Errorf("failed to update user availability on task completion: %w", err)
			}
		}

		return nil
	})
}

////////////////////////////////////////////////////////////////////////
// Private Helpers
////////////////////////////////////////////////////////////////////////

// _resolveSkills is a helper function to find existing skills and create new ones in batches.
// It takes a list of skill names and returns a map of name -> Skill object for easy lookup.
// New skills are created as 'unverified'. This function should be called within a transaction.
func (s *Store) _resolveSkills(ctx context.Context, q *Queries, skillNames []string) (map[string]Skill, error) {
	if len(skillNames) == 0 {
		return make(map[string]Skill), nil
	}

	// Step 1: Batch fetch all potentially existing skills.
	existingSkills, err := q.ListSkillsByNames(ctx, skillNames)
	if err != nil {
		return nil, fmt.Errorf("failed to batch fetch skills by name: %w", err)
	}

	skillMap := make(map[string]Skill, len(skillNames))
	for _, s := range existingSkills {
		skillMap[s.SkillName] = s
	}

	// Step 2: Identify which skills are new.
	var newSkillNames []string
	for _, name := range skillNames {
		if _, ok := skillMap[name]; !ok {
			newSkillNames = append(newSkillNames, name)
		}
	}

	// Step 3: Batch create all new skills as 'unverified'.
	if len(newSkillNames) > 0 {
		isVerifiedSlice := make([]bool, len(newSkillNames)) // All false
		createdSkills, createErr := q.CreateManySkills(ctx, CreateManySkillsParams{
			Column1: newSkillNames,
			Column2: isVerifiedSlice,
		})
		if createErr != nil {
			return nil, fmt.Errorf("failed to batch create skills: %w", createErr)
		}

		// Add the newly created skills to our map for the final result.
		for _, s := range createdSkills {
			skillMap[s.SkillName] = s
		}
	}

	return skillMap, nil
}

////////////////////////////////////////////////////////////////////////
