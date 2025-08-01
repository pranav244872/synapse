package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

////////////////////////////////////////////////////////////////////////
// Store Definition
////////////////////////////////////////////////////////////////////////

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
// Transaction: OnboardNewUserWithSkills
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

// OnboardNewUserWithSkills orchestrates a complex transaction to create a user and populate their profile.
func (s *Store) OnboardNewUserWithSkills(
	ctx context.Context,
	arg OnboardNewUserTxParams,
) (OnboardNewUserTxResult, error) {
	var result OnboardNewUserTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		// Step 1: Create the user.
		createdUser, err := q.CreateUser(ctx, arg.CreateUserParams)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		result.User = createdUser

		// Step 2: Check if there are any skills to process.
		if len(arg.SkillsWithProficiency) == 0 {
			return nil
		}

		// Step 3: Resolve all skill names to Skill objects.
		skillNames := make([]string, 0, len(arg.SkillsWithProficiency))
		for name := range arg.SkillsWithProficiency {
			skillNames = append(skillNames, name)
		}

		skillMap, err := s._resolveSkills(ctx, q, skillNames)
		if err != nil {
			return err
		}

		for name, skill := range skillMap {
			proficiency := arg.SkillsWithProficiency[name]
			userSkill, linkErr := q.AddSkillToUser(ctx, AddSkillToUserParams{
				UserID:      createdUser.ID,
				SkillID:     skill.ID,
				Proficiency: proficiency,
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
// Transaction: ProcessNewTask
////////////////////////////////////////////////////////////////////////

// ProcessNewTaskTxParams includes the pre-processed list of required skills.
type ProcessNewTaskTxParams struct {
	CreateTaskParams    CreateTaskParams
	RequiredSkillNames  []string
}

// ProcessNewTaskTxResult contains the result of the ProcessNewTask transaction.
type ProcessNewTaskTxResult struct {
	Task               Task
	TaskRequiredSkills []TaskRequiredSkill
}

// ProcessNewTask creates a task and automatically links required skills extracted from its description.
func (s *Store) ProcessNewTask(
	ctx context.Context,
	arg ProcessNewTaskTxParams,
) (ProcessNewTaskTxResult, error) {
	var result ProcessNewTaskTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		// Step 1: Create the task.
		createdTask, err := q.CreateTask(ctx, arg.CreateTaskParams)
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}
		result.Task = createdTask

		if len(arg.RequiredSkillNames) == 0 {
			return nil
		}

		// Step 2: Resolve skill names to Skill objects.
		skillMap, err := s._resolveSkills(ctx, q, arg.RequiredSkillNames)
		if err != nil {
			return err
		}

		// Step 3: Link all required skills to the task.
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
// Transaction: AssignTaskToUser
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

// AssignTaskToUser assigns a task to a user and marks them busy within a transaction.
func (s *Store) AssignTaskToUser(
	ctx context.Context,
	arg AssignTaskToUserTxParams,
) (AssignTaskToUserTxResult, error) {
	var result AssignTaskToUserTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		var err error

		// Step 1: Update task assignment and status.
		result.Task, err = q.UpdateTask(ctx, UpdateTaskParams{
			ID:         arg.TaskID,
			AssigneeID: pgtype.Int8{Int64: arg.UserID, Valid: true},
			Status:     NullTaskStatus{TaskStatus: "in_progress", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update task assignment: %w", err)
		}

		// Step 2: Update user availability to 'busy'.
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
// Transaction: CompleteTask
////////////////////////////////////////////////////////////////////////

// CompleteTaskTxParams contains the parameters for completing a task.
type CompleteTaskTxParams struct {
	TaskID int64
}

// CompleteTask marks a task as 'done' and resets the assignee's availability to 'available'.
func (s *Store) CompleteTask(ctx context.Context, arg CompleteTaskTxParams) error {
	return s.execTx(ctx, func(q *Queries) error {
		// Step 1: Get the task and find the assignee.
		task, err := q.GetTask(ctx, arg.TaskID)
		if err != nil {
			return fmt.Errorf("failed to get task for completion: %w", err)
		}

		// Step 2: Mark the task as done.
		_, err = q.UpdateTask(ctx, UpdateTaskParams{
			ID:          arg.TaskID,
			Status:      NullTaskStatus{TaskStatus: "done", Valid: true},
			CompletedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to mark task as done: %w", err)
		}

		// Step 3: Update assignee's availability to 'available'.
		if task.AssigneeID.Valid {
			_, err = q.UpdateUser(ctx, UpdateUserParams{
				ID:           task.AssigneeID.Int64,
				Availability: NullAvailabilityStatus{AvailabilityStatus: "available", Valid: true},
			})
			if err != nil {
				return fmt.Errorf("failed to update user availability: %w", err)
			}
		}

		return nil
	})
}

////////////////////////////////////////////////////////////////////////
// Transaction: CreateInvitation
////////////////////////////////////////////////////////////////////////

// CreateInvitationTxParams contains the input parameters for the CreateInvitation transaction.
// TeamID is now included to associate a manager invitation with a specific team.
type CreateInvitationTxParams struct {
	InviterID     int64
	EmailToInvite string
	RoleToInvite  UserRole
	TeamID        pgtype.Int8 // Use pgtype.Int8 to handle optional/NULL values. Required for manager invites.
}

// CreateInvitationTxResult contains the result of the CreateInvitation transaction.
type CreateInvitationTxResult struct {
	Invitation Invitation
}

var (
	ErrPermissionDenied           = errors.New("user does not have permission for this action")
	ErrDuplicateInvitation        = errors.New("a pending invitation for this email already exists")
	ErrInvalidRoleSequence        = errors.New("invitations can only be for a lower role in the hierarchy (admin -> manager -> engineer)")
	ErrTeamIDRequiredForManager   = errors.New("a team ID must be provided when inviting a manager")
	ErrManagerMustHaveTeam        = errors.New("a manager must be assigned to a team to invite engineers")
	ErrTeamNotFound               = errors.New("the specified team was not found")
	ErrTeamAlreadyHasManager      = errors.New("the specified team already has a manager assigned")
)

// CreateInvitationTx handles the creation of a new user invitation within a database transaction.
// It now enforces that all invitations are associated with a team.
func (s *Store) CreateInvitationTx(ctx context.Context, arg CreateInvitationTxParams) (CreateInvitationTxResult, error) {
	var result CreateInvitationTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		// Step 1: Validate inviter.
		inviter, err := q.GetUser(ctx, arg.InviterID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("inviter with ID %d not found", arg.InviterID)
			}
			return fmt.Errorf("failed to get inviter: %w", err)
		}

		var invitationTeamID pgtype.Int8

		// Step 2: Validate role hierarchy and determine the Team ID for the invitation.
		switch inviter.Role {
		case UserRoleAdmin:
			if arg.RoleToInvite != UserRoleManager {
				return fmt.Errorf("%w: admins can only invite managers", ErrInvalidRoleSequence)
			}
			// For manager invites, the TeamID must be explicitly provided in the arguments.
			if !arg.TeamID.Valid {
				return ErrTeamIDRequiredForManager
			}

			// Validate the provided team: it must exist and be vacant.
			team, err := q.GetTeam(ctx, arg.TeamID.Int64)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return fmt.Errorf("%w: team with ID %d", ErrTeamNotFound, arg.TeamID.Int64)
				}
				return fmt.Errorf("failed to get team: %w", err)
			}
			if team.ManagerID.Valid {
				return ErrTeamAlreadyHasManager
			}
			invitationTeamID = arg.TeamID

		case UserRoleManager:
			if arg.RoleToInvite != UserRoleEngineer {
				return fmt.Errorf("%w: managers can only invite engineers", ErrInvalidRoleSequence)
			}
			// For engineer invites, the team is automatically the manager's own team.
			if !inviter.TeamID.Valid {
				return ErrManagerMustHaveTeam
			}
			invitationTeamID = inviter.TeamID

		default:
			return fmt.Errorf("%w: user with role '%s' cannot send invitations", ErrPermissionDenied, inviter.Role)
		}

		// Step 3: Ensure no duplicate pending invitation for the same email.
		_, err = q.GetInvitationByEmail(ctx, arg.EmailToInvite)
		if err == nil {
			return ErrDuplicateInvitation
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to check for existing invitation: %w", err)
		}

		// Step 4: Generate a secure invitation token.
		token, err := uuid.NewRandom()
		if err != nil {
			return fmt.Errorf("failed to generate invitation token: %w", err)
		}

		// Step 5: Create the invitation record, now with a mandatory TeamID.
		createParams := CreateInvitationParams{
			Email:           arg.EmailToInvite,
			InvitationToken: token.String(),
			RoleToInvite:    arg.RoleToInvite,
			InviterID:       arg.InviterID,
			TeamID:          invitationTeamID, // Use the determined team ID
			ExpiresAt: pgtype.Timestamp{
				Time:  time.Now().Add(72 * time.Hour),
				Valid: true,
			},
		}

		invitation, err := q.CreateInvitation(ctx, createParams)
		if err != nil {
			return fmt.Errorf("failed to create invitation: %w", err)
		}

		result.Invitation = invitation
		return nil
	})
	return result, err
}

////////////////////////////////////////////////////////////////////////
// Private Helpers
////////////////////////////////////////////////////////////////////////

// _resolveSkills resolves a list of skill names into a map of skill name -> Skill object.
// Creates missing skills as 'unverified' and returns all.
func (s *Store) _resolveSkills(ctx context.Context, q *Queries, skillNames []string) (map[string]Skill, error) {
	if len(skillNames) == 0 {
		return make(map[string]Skill), nil
	}

	// Step 1: Batch fetch existing skills.
	existingSkills, err := q.ListSkillsByNames(ctx, skillNames)
	if err != nil {
		return nil, fmt.Errorf("failed to batch fetch skills: %w", err)
	}

	skillMap := make(map[string]Skill, len(skillNames))
	for _, s := range existingSkills {
		skillMap[s.SkillName] = s
	}

	// Step 2: Identify and batch-create new skills.
	var newSkillNames []string
	for _, name := range skillNames {
		if _, ok := skillMap[name]; !ok {
			newSkillNames = append(newSkillNames, name)
		}
	}

	if len(newSkillNames) > 0 {
		isVerifiedSlice := make([]bool, len(newSkillNames))
		createdSkills, err := q.CreateManySkills(ctx, CreateManySkillsParams{
			Column1: newSkillNames,
			Column2: isVerifiedSlice,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to batch create skills: %w", err)
		}
		for _, s := range createdSkills {
			skillMap[s.SkillName] = s
		}
	}

	return skillMap, nil
}
