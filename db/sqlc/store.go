package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pranav244872/synapse/skillz"
)

// Store provides all functions to execute db queries and transactions.
// It now holds a pgxpool.Pool, which satisfies the DBTX interface.
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
// This is a common helper function to add to the Store struct.
func (s *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.dbpool.Begin(ctx)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

////////////////////////////////////////////////////////////////////////

// OnboardNewUserTxParams contains the input parameters for the OnboardNewUserWithSkills transaction.
type OnboardNewUserTxParams struct {
	CreateUserParams CreateUserParams
	ResumeText       string
}

// OnboardNewUserTxResult contains the result of the OnboardNewUserWithSkills transaction.
type OnboardNewUserTxResult struct {
	User       User
	UserSkills []UserSkill
}

// OnboardNewUserWithSkills creates a new user and then uses the skillz processor to extract,
// normalize, and assign skills with proficiency levels based on their resume.
// it creates new skills as 'unverified' if they don't exist, ensuring all
// potential skills from a resume are captured.
func (s *Store) OnboardNewUserWithSkills(
	ctx context.Context,
	arg OnboardNewUserTxParams,
	skillProcessor skillz.Processor,
) (OnboardNewUserTxResult, error) {
	var result OnboardNewUserTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		var err error

		// Step 1: Create the user record in the database.
		result.User, err = q.CreateUser(ctx, arg.CreateUserParams)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// Step 2: Use the LLM to extract and normalize skills from the resume text.
		// The alias map for normalization should be loaded when the skillProcessor is initialized.
		normalizedSkills, err := skillProcessor.ExtractAndNormalize(ctx, arg.ResumeText)
		if err != nil {
			return fmt.Errorf(
				"failed to extract and normalize skills: %w",
				err,
			)
		}

		if len(normalizedSkills) == 0 {
			// No skills found, transaction is successful but there's nothing more to do.
			return nil
		}

		// Step 3: Use the LLM to get proficiency levels for the clean list of skills.
		proficiencies, err := skillProcessor.ExtractProficiencies(
			ctx,
			arg.ResumeText,
			normalizedSkills,
		)
		if err != nil {
			return fmt.Errorf("failed to extract proficiencies: %w", err)
		}

		// Step 4: Loop through the results and link the skills to the new user.
		for skillName, proficiency := range proficiencies {
			var skillID int64

			// Attempt to find the skill in our 'skills' table.
			skill, err := q.GetSkillByName(ctx, skillName)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					// If the skill is not found, create it as unverified.
					newSkill, createErr := q.CreateSkill(ctx, CreateSkillParams{
						SkillName:  skillName,
						IsVerified: false,
					})
					if createErr != nil {
						return fmt.Errorf(
							"failed to create new skill '%s': %w",
							skillName,
							createErr,
						)
					}
					skillID = newSkill.ID
				} else {
					// A different database error occured
					return fmt.Errorf(
						"failed to get skill by name '%s': %w",
						skillName,
						err,
					)
				}
			} else {
				// Skill already exists, use its ID.
				skillID = skill.ID
			}

			// Add the skill (whether old or new) and its proficiency to the user.
			userSkill, err := q.AddSkillToUser(ctx, AddSkillToUserParams{
				UserID:      result.User.ID,
				SkillID:     skillID,
				Proficiency: ProficiencyLevel(proficiency), // Cast string to the enum type
			})

			if err != nil {
				return fmt.Errorf(
					"failed to add skill '%s' to user: %w",
					skillName,
					err,
				)
			}
			result.UserSkills = append(result.UserSkills, userSkill)
		}

		return nil
	})

	return result, err
}

////////////////////////////////////////////////////////////////////////

// ProcessNewTaskTxParams contains the input parameters for the ProcessNewTask transaction.
type ProcessNewTaskTxParams struct {
	CreateTaskParams CreateTaskParams
	Description      string // Pass description separately for processing
}

// ProcessNewTaskTxResult contains the result of the ProcessNewTask transaction.
type ProcessNewTaskTxResult struct {
	Task               Task
	TaskRequiredSkills []TaskRequiredSkill
}

// ProcessNewTask creates a new task and links it to required skills extracted from its description.
// It will create new skills in the 'skills' table if they don't already exist, marking them as unverified.
func (s *Store) ProcessNewTask(
	ctx context.Context,
	arg ProcessNewTaskTxParams,
	skillProcessor skillz.Processor,
) (ProcessNewTaskTxResult, error) {
	var result ProcessNewTaskTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		var err error

		// Step 1: Create the task record.
		result.Task, err = q.CreateTask(ctx, arg.CreateTaskParams)
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		// Step 2: Extract and normalize skills from the task description.
		normalizedSkills, err := skillProcessor.ExtractAndNormalize(ctx, arg.Description)
		if err != nil {
			return fmt.Errorf("failed to extract and normalize skills: %w", err)
		}

		// Step 3: For each skill, find it or create it (as unverified), then link to the task.
		for _, skillName := range normalizedSkills {
			var skillID int64
			// Attempt to find the skill by its canonical name.
			skill, err := q.GetSkillByName(ctx, skillName)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					// Skill does not exist, so create it as 'unverified'.
					// This allows the system to learn new skills organically.
					newSkill, createErr := q.CreateSkill(ctx, CreateSkillParams{
						SkillName:  skillName,
						IsVerified: false,
					})
					if createErr != nil {
						return fmt.Errorf(
							"failed to create new skill '%s': %w",
							skillName,
							createErr,
						)
					}
					skillID = newSkill.ID
				} else {
					// A different database error occurred.
					return fmt.Errorf("failed to get skill '%s': %w", skillName, err)
				}
			} else {
				// Skill already exists, use its ID.
				skillID = skill.ID
			}

			// Step 4: Link the skill (whether old or new) to the task.
			requiredSkill, linkErr := q.AddSkillToTask(ctx, AddSkillToTaskParams{
				TaskID:  result.Task.ID,
				SkillID: skillID,
			})
			if linkErr != nil {
				// We can choose to continue or fail. Failing is safer to ensure data integrity.
				return fmt.Errorf(
					"failed to link skill '%s' to task: %w",
					skillName,
					linkErr,
				)
			}
			result.TaskRequiredSkills = append(result.TaskRequiredSkills, requiredSkill)
		}
		return nil
	})

	return result, err
}

////////////////////////////////////////////////////////////////////////

// AssignTaskToUserTxParams contains the input parameters for the AssignTaskToUser transaction.
type AssignTaskToUserTxParams struct {
	TaskID int64
	UserID int64
}

// AssignTaskToUserTxResult contains the result of the AssignTaskToUser transaction.
type AssignTaskToUserTxResult struct {
	User User
	Task Task
}

// AssignTaskToUser assigns a task to a user, setting the task status to 'in_progress' and the user's availability to 'busy'.
func (s *Store) AssignTaskToUser(
	ctx context.Context,
	arg AssignTaskToUserTxParams,
) (AssignTaskToUserTxResult, error) {
	var result AssignTaskToUserTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		var err error

		// Step 1: Update the task to set the assignee and change status.
		_, err = q.UpdateTask(ctx, UpdateTaskParams{
			ID:         arg.TaskID,
			AssigneeID: pgtype.Int8{Int64: arg.UserID, Valid: true},
			Status:     NullTaskStatus{TaskStatus: "in_progress", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update task assignment: %w", err)
		}

		// Step 2: Update the user's availability to reflect they are now busy.
		_, err = q.UpdateUser(ctx, UpdateUserParams{
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

// CompleteTaskTxParams contains the input parameters for the CompleteTask transaction.
type CompleteTaskTxParams struct {
	TaskID int64
}

// CompleteTask marks a task as 'done' and sets the assignee's availability back to 'available'.
// This transaction is critical for feeding historical data into the recommendation engine.
func (s *Store) CompleteTask(ctx context.Context, arg CompleteTaskTxParams) error {

	err := s.execTx(ctx, func(q *Queries) error {
		// Step 1: Get the current task details *before* updating.
		// We need the assignee_id to know which user to make available again.
		task, err := q.GetTask(ctx, arg.TaskID)
		if err != nil {
			return fmt.Errorf("failed to get task for completion: %w", err)
		}

		// Step 2: Update the task status to 'done' and set the completion timestamp.
		_, err = q.UpdateTask(ctx, UpdateTaskParams{
			ID:          arg.TaskID,
			Status:      NullTaskStatus{TaskStatus: "done", Valid: true},
			CompletedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to mark task as done: %w", err)
		}

		// Step 3: If the task had an assignee, update their availability back to 'available'.
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

	return err
}

////////////////////////////////////////////////////////////////////////
