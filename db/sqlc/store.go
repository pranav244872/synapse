// db/store.go

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
// Transaction: CreateInvitationTx
////////////////////////////////////////////////////////////////////////

// CreateInvitationTxParams contains the input parameters for the CreateInvitation transaction.
type CreateInvitationTxParams struct {
	InviterID     int64       // ID of the user sending the invitation
	EmailToInvite string      // Email address of the invitee
	RoleToInvite  UserRole    // Role to assign to the invitee (manager or engineer)
	TeamID        pgtype.Int8 // Required for manager invites; auto-derived for engineer invites
}

// CreateInvitationTxResult contains the result of the CreateInvitation transaction.
type CreateInvitationTxResult struct {
	Invitation CreateInvitationRow // Full invitation details with inviter info
}

// Error definitions for invitation creation
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
// Enforces strict role hierarchy: admins can only invite managers, managers can only invite engineers.
// Ensures team assignment rules and prevents duplicate invitations.
func (s *Store) CreateInvitationTx(ctx context.Context, arg CreateInvitationTxParams) (CreateInvitationTxResult, error) {
	var result CreateInvitationTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		// Step 1: Validate inviter identity and permissions
		// Fetch the inviter from the database to verify their role and team assignment
		inviter, err := q.GetUser(ctx, arg.InviterID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("inviter with ID %d not found", arg.InviterID)
			}
			return fmt.Errorf("failed to get inviter: %w", err)
		}

		var invitationTeamID pgtype.Int8

		// Step 2: Validate role hierarchy and determine team assignment
		// The role hierarchy is: admin -> manager -> engineer
		switch inviter.Role {
		case UserRoleAdmin:
			// Admins can only invite managers
			if arg.RoleToInvite != UserRoleManager {
				return fmt.Errorf("%w: admins can only invite managers", ErrInvalidRoleSequence)
			}
			
			// For manager invites, the TeamID must be explicitly provided
			if !arg.TeamID.Valid {
				return ErrTeamIDRequiredForManager
			}

			// Validate the provided team: it must exist and not already have a manager
			team, err := q.GetTeam(ctx, arg.TeamID.Int64)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return fmt.Errorf("%w: team with ID %d", ErrTeamNotFound, arg.TeamID.Int64)
				}
				return fmt.Errorf("failed to get team: %w", err)
			}
			
			// Check if team already has a manager assigned
			if team.ManagerID.Valid {
				return ErrTeamAlreadyHasManager
			}
			
			invitationTeamID = arg.TeamID

		case UserRoleManager:
			// Managers can only invite engineers
			if arg.RoleToInvite != UserRoleEngineer {
				return fmt.Errorf("%w: managers can only invite engineers", ErrInvalidRoleSequence)
			}
			
			// For engineer invites, the team is automatically the manager's own team
			if !inviter.TeamID.Valid {
				return ErrManagerMustHaveTeam
			}
			
			invitationTeamID = inviter.TeamID

		default:
			// Only admins and managers can send invitations
			return fmt.Errorf("%w: user with role '%s' cannot send invitations", ErrPermissionDenied, inviter.Role)
		}

		// Step 3: Check for duplicate pending invitations
		// Prevent sending multiple invitations to the same email address
		_, err = q.GetInvitationByEmail(ctx, arg.EmailToInvite)
		if err == nil {
			// If we found an existing invitation, it's a duplicate
			return ErrDuplicateInvitation
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			// If error is not "no rows found", it's a real database error
			return fmt.Errorf("failed to check for existing invitation: %w", err)
		}

		// Step 4: Generate a secure invitation token
		// Using UUID for cryptographically secure token generation
		token, err := uuid.NewRandom()
		if err != nil {
			return fmt.Errorf("failed to generate invitation token: %w", err)
		}

		// Step 5: Set invitation expiration time
		// Invitations expire after 72 hours (3 days) from creation
		expirationTime := time.Now().Add(72 * time.Hour)

		// Step 6: Create the invitation record with all validated parameters
		createParams := CreateInvitationParams{
			Email:           arg.EmailToInvite,
			InvitationToken: token.String(),
			RoleToInvite:    arg.RoleToInvite,
			InviterID:       arg.InviterID,
			TeamID:          invitationTeamID, // Team determined based on inviter role
			ExpiresAt: pgtype.Timestamp{
				Time:  expirationTime,
				Valid: true,
			},
		}

		// Execute the database insertion
		invitation, err := q.CreateInvitation(ctx, createParams)
		if err != nil {
			return fmt.Errorf("failed to create invitation: %w", err)
		}

		// Convert the CreateInvitationRow to an Invitation struct for the result
		result.Invitation = invitation
		return nil
	})

	return result, err
}

////////////////////////////////////////////////////////////////////////
// Transaction: AcceptInvitationTx
////////////////////////////////////////////////////////////////////////

// AcceptInvitationTxParams contains the parameters for accepting an invitation.
type AcceptInvitationTxParams struct {
	InvitationToken       string                        // Token from the invitation email
	UserName              string                        // Display name for the new user
	PasswordHash          string                        // Pre-hashed password for the new user
	SkillsWithProficiency map[string]ProficiencyLevel   // Optional skills to associate with the user
}

// AcceptInvitationTxResult contains the result of accepting an invitation.
type AcceptInvitationTxResult struct {
	User       User         // The newly created user account
	UserSkills []UserSkill  // Skills associated with the user (if any provided)
}

// Error definitions for invitation acceptance
var (
	ErrInvitationNotPending = errors.New("invitation is not pending and cannot be accepted")
)

// AcceptInvitationTx handles the complete user onboarding flow when accepting an invitation.
// This includes creating the user account, assigning them to a team, updating team management
// if they're a manager, marking the invitation as accepted, and optionally adding skills.
func (s *Store) AcceptInvitationTx(ctx context.Context, arg AcceptInvitationTxParams) (AcceptInvitationTxResult, error) {
	var result AcceptInvitationTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		// Step 1: Validate the invitation token
		// Look up the invitation and ensure it's still valid and pending
		invitation, err := q.GetInvitationByToken(ctx, arg.InvitationToken)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvitationNotPending
			}
			return fmt.Errorf("failed to get invitation: %w", err)
		}

		// Step 2: Verify invitation is still pending
		// Only pending invitations can be accepted
		if invitation.Status != "pending" {
			return ErrInvitationNotPending
		}

		// Step 3: Create the new user account
		// Use information from the invitation (email, role, team) rather than trusting client input
		createUserParams := CreateUserParams{
			Name:         pgtype.Text{String: arg.UserName, Valid: true},
			Email:        invitation.Email,           // Email comes from invitation, not client
			PasswordHash: arg.PasswordHash,
			Role:         invitation.RoleToInvite,   // Role comes from invitation
			TeamID:       invitation.TeamID,         // Team assignment comes from invitation
		}

		user, err := q.CreateUser(ctx, createUserParams)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		result.User = user

		// Step 4: Handle manager team assignment
		// If the new user is a manager, assign them as the team's manager
		if invitation.RoleToInvite == UserRoleManager && invitation.TeamID.Valid {
			_, err := q.SetTeamManager(ctx, SetTeamManagerParams{
				ID:        invitation.TeamID.Int64,
				ManagerID: pgtype.Int8{Int64: user.ID, Valid: true},
			})
			if err != nil {
				return fmt.Errorf("failed to assign user as team manager: %w", err)
			}
		}

		// Step 5: Mark invitation as accepted
		// This prevents the invitation from being used again
		_, err = q.UpdateInvitationStatus(ctx, UpdateInvitationStatusParams{
			ID:     invitation.ID,
			Status: "accepted",
		})
		if err != nil {
			return fmt.Errorf("failed to mark invitation as accepted: %w", err)
		}

		// Step 6: Process optional skills
		// If the user provided skills during signup, add them to their profile
		if len(arg.SkillsWithProficiency) > 0 {
			// Extract skill names for bulk resolution
			skillNames := make([]string, 0, len(arg.SkillsWithProficiency))
			for name := range arg.SkillsWithProficiency {
				skillNames = append(skillNames, name)
			}

			// Resolve skill names to skill objects (creates new skills if they don't exist)
			skillMap, err := s._resolveSkills(ctx, q, skillNames)
			if err != nil {
				return fmt.Errorf("failed to resolve skills: %w", err)
			}

			// Associate each skill with the user at the specified proficiency level
			for name, skill := range skillMap {
				proficiency := arg.SkillsWithProficiency[name]
				userSkill, linkErr := q.AddSkillToUser(ctx, AddSkillToUserParams{
					UserID:      user.ID,
					SkillID:     skill.ID,
					Proficiency: proficiency,
				})
				if linkErr != nil {
					return fmt.Errorf("failed to add skill '%s' to user: %w", name, linkErr)
				}
				result.UserSkills = append(result.UserSkills, userSkill)
			}
		}

		return nil
	})

	return result, err
}

////////////////////////////////////////////////////////////////////////
// Transaction: SafeDeleteUserTx
////////////////////////////////////////////////////////////////////////

// SafeDeleteUserTxParams contains the parameters for safely deleting a user
type SafeDeleteUserTxParams struct {
	UserID int64
}

// SafeDeleteUserTxResult contains the result of the safe user deletion
type SafeDeleteUserTxResult struct {
	DeletedUser        User    // The user that was deleted
	UpdatedTasks       []Task  // Tasks that had assignee_id set to NULL
	UpdatedTeams       []Team  // Teams that had manager_id set to NULL
	RemovedSkills      int64   // Count of user_skills entries removed (CASCADE)
	RemovedInvitations int64   // Count of invitations removed (CASCADE)
}

// SafeDeleteUserTx safely removes a user and handles all cascading effects
// according to the database schema foreign key constraints:
// - tasks.assignee_id → users.id [SET NULL]: Tasks are unassigned and reset to "open"
// - teams.manager_id → users.id [SET NULL]: Teams become unmanaged
// - user_skills.user_id → users.id [CASCADE]: Skills are automatically removed
// - invitations.inviter_id → users.id [CASCADE]: Invitations are automatically removed
func (s *Store) SafeDeleteUserTx(ctx context.Context, arg SafeDeleteUserTxParams) (SafeDeleteUserTxResult, error) {
	var result SafeDeleteUserTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		// Step 1: Get the user to be deleted for validation and result
		user, err := q.GetUser(ctx, arg.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user for deletion: %w", err)
		}
		result.DeletedUser = user

		// Step 2: CRITICAL BUSINESS RULE - Prevent admin deletion for system integrity
		if user.Role == UserRoleAdmin {
			return fmt.Errorf("admin users cannot be deleted for system integrity")
		}

		// Step 3: Handle tasks assigned to this user (SET NULL per schema)
		// Get all tasks assigned to this user
		assignedTasks, err := q.ListTasksByAssignee(ctx, ListTasksByAssigneeParams{
			AssigneeID: pgtype.Int8{Int64: arg.UserID, Valid: true},
			Limit:      1000, // High limit to get all tasks
			Offset:     0,
		})
		if err != nil {
			return fmt.Errorf("failed to get assigned tasks: %w", err)
		}

		// Update each task to remove the assignee and reset status to "open"
		for _, task := range assignedTasks {
			updatedTask, err := q.UpdateTask(ctx, UpdateTaskParams{
				ID:          task.ID,
				AssigneeID:  pgtype.Int8{Valid: false}, // SET NULL
				Status:      NullTaskStatus{TaskStatus: "open", Valid: true}, // Reset to open
			})
			if err != nil {
				return fmt.Errorf("failed to unassign task %d: %w", task.ID, err)
			}
			result.UpdatedTasks = append(result.UpdatedTasks, updatedTask)
		}

		// Step 4: Handle teams managed by this user (SET NULL per schema)
		if user.Role == UserRoleManager {
			// Find team(s) managed by this user
			team, err := q.GetTeamByManagerID(ctx, pgtype.Int8{Int64: arg.UserID, Valid: true})
			if err == nil {
				// Team found, remove manager (SET NULL)
				updatedTeam, err := q.SetTeamManager(ctx, SetTeamManagerParams{
					ID:        team.ID,
					ManagerID: pgtype.Int8{Valid: false}, // SET NULL
				})
				if err != nil {
					return fmt.Errorf("failed to remove manager from team %d: %w", team.ID, err)
				}
				result.UpdatedTeams = append(result.UpdatedTeams, updatedTeam)
				
				// NOTE: Projects remain with the team (projects.team_id relationship intact)
				// The team still exists, it just doesn't have a manager
			}
			// If no team found (err != nil), it's fine - user might not manage any team
		}

		// Step 5: Count user skills before deletion (CASCADE will handle automatic removal)
		userSkills, err := q.GetSkillsForUser(ctx, arg.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user skills for counting: %w", err)
		}
		result.RemovedSkills = int64(len(userSkills))

		// Step 6: Count invitations before deletion (CASCADE will handle automatic removal)
		invitations, err := q.ListInvitationsByInviter(ctx, ListInvitationsByInviterParams{
			InviterID: arg.UserID,
			Limit:     1000,
			Offset:    0,
		})
		if err != nil {
			return fmt.Errorf("failed to get user invitations for counting: %w", err)
		}
		result.RemovedInvitations = int64(len(invitations))

		// Step 7: Finally, delete the user
		// The database CASCADE constraints will automatically handle:
		// - user_skills (DELETE CASCADE)
		// - invitations sent by user (DELETE CASCADE)
		err = q.DeleteUser(ctx, arg.UserID)
		if err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		return nil
	})

	return result, err
}

////////////////////////////////////////////////////////////////////////
// Transaction: GetUserDeletionImpactTx (Dry-Run)
////////////////////////////////////////////////////////////////////////

// GetUserDeletionImpactTxParams contains parameters for deletion impact analysis
type GetUserDeletionImpactTxParams struct {
	UserID int64 // ID of the user to analyze for deletion impact
}

// GetUserDeletionImpactTxResult contains the impact analysis without actual deletion
type GetUserDeletionImpactTxResult struct {
	User               User    // The user that would be deleted
	TasksToUnassign    []Task  // Tasks that would have assignee_id set to NULL
	TeamsToOrphan      []Team  // Teams that would have manager_id set to NULL
	SkillsToRemove     int64   // Count of user_skills entries that would be removed
	InvitationsToRemove int64  // Count of invitations that would be removed
	CanDelete          bool    // Whether deletion is allowed (false for admins)
	BlockingReason     string  // Reason why deletion is blocked (if CanDelete is false)
}

// GetUserDeletionImpactTx analyzes the impact of deleting a user without actually deleting them.
// This is a READ-ONLY transaction that provides comprehensive impact assessment for admin UI.
func (s *Store) GetUserDeletionImpactTx(ctx context.Context, arg GetUserDeletionImpactTxParams) (GetUserDeletionImpactTxResult, error) {
	var result GetUserDeletionImpactTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		// Step 1: Get the user for impact analysis
		user, err := q.GetUser(ctx, arg.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user for impact analysis: %w", err)
		}
		result.User = user

		// Step 2: Check if deletion is allowed (same business rule as actual deletion)
		if user.Role == UserRoleAdmin {
			result.CanDelete = false
			result.BlockingReason = "Admin users cannot be deleted for system integrity"
			// Still continue analysis to show what WOULD happen
		} else {
			result.CanDelete = true
		}

		// Step 3: Analyze task impact - find tasks that would be unassigned
		assignedTasks, err := q.ListTasksByAssignee(ctx, ListTasksByAssigneeParams{
			AssigneeID: pgtype.Int8{Int64: arg.UserID, Valid: true},
			Limit:      1000, // High limit to get all tasks
			Offset:     0,
		})
		if err != nil {
			return fmt.Errorf("failed to get assigned tasks for analysis: %w", err)
		}
		result.TasksToUnassign = assignedTasks

		// Step 4: Analyze team management impact
		if user.Role == UserRoleManager {
			// Find team(s) that would be orphaned
			team, err := q.GetTeamByManagerID(ctx, pgtype.Int8{Int64: arg.UserID, Valid: true})
			if err == nil {
				// Team found - it would be orphaned
				result.TeamsToOrphan = append(result.TeamsToOrphan, team)
			}
			// If no team found, no impact on team management
		}

		// Step 5: Count skills that would be removed
		userSkills, err := q.GetSkillsForUser(ctx, arg.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user skills for analysis: %w", err)
		}
		result.SkillsToRemove = int64(len(userSkills))

		// Step 6: Count invitations that would be removed
		invitations, err := q.ListInvitationsByInviter(ctx, ListInvitationsByInviterParams{
			InviterID: arg.UserID,
			Limit:     1000,
			Offset:    0,
		})
		if err != nil {
			return fmt.Errorf("failed to get user invitations for analysis: %w", err)
		}
		result.InvitationsToRemove = int64(len(invitations))

		return nil
	})

	return result, err
}


////////////////////////////////////////////////////////////////////////
// Transaction: ValidateUserRoleChangeTx
////////////////////////////////////////////////////////////////////////

// ValidateUserRoleChangeTxParams contains parameters for role change validation
type ValidateUserRoleChangeTxParams struct {
	UserID  int64     // ID of user whose role is being changed
	NewRole UserRole  // The role they're being changed to
	TeamID  *int64    // Optional team assignment (required for manager promotion)
}

// ValidateUserRoleChangeTxResult contains the result of role change validation
type ValidateUserRoleChangeTxResult struct {
	IsValid      bool   // Whether the role change is valid
	ErrorMessage string // Error message if not valid
	CurrentUser  User   // Current user information
	TargetTeam   *Team  // Target team if promoting to manager
	ManagedTeam  *Team  // Currently managed team if demoting from manager
}

// ValidateUserRoleChangeTx checks if a role change is valid according to business rules:
// 1. Admin roles cannot be changed (system integrity)
// 2. Cannot promote users to admin role (security)
// 3. Manager promotion requires a team without existing manager
// 4. Manager demotion requires handling of currently managed team
func (s *Store) ValidateUserRoleChangeTx(ctx context.Context, arg ValidateUserRoleChangeTxParams) (ValidateUserRoleChangeTxResult, error) {
	var result ValidateUserRoleChangeTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		// Get current user information
		user, err := q.GetUser(ctx, arg.UserID)
		if err != nil {
			return fmt.Errorf("user not found: %w", err)
		}
		result.CurrentUser = user

		// Apply business rule validations
		switch {
		case user.Role == arg.NewRole:
			// No change needed
			result.IsValid = false
			result.ErrorMessage = "user already has this role"
			return nil

		case user.Role == UserRoleAdmin:
			// BUSINESS RULE: Admin role cannot be changed (system integrity)
			result.IsValid = false
			result.ErrorMessage = "admin role cannot be changed"
			return nil

		case arg.NewRole == UserRoleAdmin:
			// BUSINESS RULE: Cannot promote users to admin role (security)
			result.IsValid = false
			result.ErrorMessage = "cannot promote users to admin role"
			return nil

		case arg.NewRole == UserRoleManager:
			// BUSINESS RULE: Manager promotion requires team assignment
			if arg.TeamID == nil {
				result.IsValid = false
				result.ErrorMessage = "team assignment required when promoting to manager"
				return nil
			}

			// BUSINESS RULE: Target team must exist and have no current manager
			team, err := q.GetTeam(ctx, *arg.TeamID)
			if err != nil {
				result.IsValid = false
				result.ErrorMessage = "target team not found"
				return nil
			}

			if team.ManagerID.Valid {
				result.IsValid = false
				result.ErrorMessage = "target team already has a manager"
				return nil
			}

			result.TargetTeam = &team

		case user.Role == UserRoleManager && arg.NewRole != UserRoleManager:
			// BUSINESS RULE: Manager demotion requires handling current team management
			team, err := q.GetTeamByManagerID(ctx, pgtype.Int8{Int64: arg.UserID, Valid: true})
			if err == nil {
				// User currently manages a team - this will be handled in the update
				result.ManagedTeam = &team
			}
			// If no team found, it's fine - user doesn't manage any team
		}

		// All validations passed
		result.IsValid = true
		return nil
	})

	return result, err
}

////////////////////////////////////////////////////////////////////////
// Transaction: ArchiveProjectTx
////////////////////////////////////////////////////////////////////////

// ArchiveProjectTxParams contains parameters for archiving a project
type ArchiveProjectTxParams struct {
	ProjectID int64
	TeamID    int64
}

// ArchiveProjectTxResult contains the result of archiving a project
type ArchiveProjectTxResult struct {
	ArchivedProject    Project
	ArchivedTasksCount int64
}

// Error definitions for project archiving
var (
	ErrProjectNotFound        = errors.New("project not found or access denied")
	ErrProjectAlreadyArchived = errors.New("project is already archived")
)

// ArchiveProjectTx archives a project and automatically archives all its tasks.
// Also frees up engineers who were assigned to tasks in this project.
func (s *Store) ArchiveProjectTx(ctx context.Context, arg ArchiveProjectTxParams) (ArchiveProjectTxResult, error) {
	var result ArchiveProjectTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		// Step 1: Validate project exists and belongs to team
		project, err := q.GetProjectByIDAndTeam(ctx, GetProjectByIDAndTeamParams{
			ID:     arg.ProjectID,
			TeamID: arg.TeamID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrProjectNotFound
			}
			return fmt.Errorf("failed to get project: %w", err)
		}

		// Step 2: Check if project is already archived
		if project.Archived {
			return ErrProjectAlreadyArchived
		}

		// Step 3: Count active tasks before archiving them
		activeTasksCount, err := q.CountActiveTasksByProject(ctx, pgtype.Int8{Int64: arg.ProjectID, Valid: true})
		if err != nil {
			return fmt.Errorf("failed to count active tasks: %w", err)
		}

		// Step 4: FREE UP ENGINEERS - Get all assigned engineers before archiving tasks
		if activeTasksCount > 0 {
			// Get all users assigned to tasks in this project
			assignedEngineers, err := q.GetAssignedEngineersForProject(ctx, pgtype.Int8{Int64: arg.ProjectID, Valid: true})
			if err != nil {
				return fmt.Errorf("failed to get assigned engineers: %w", err)
			}

			// Set all assigned engineers back to available
			for _, engineer := range assignedEngineers {
				_, err = q.UpdateUser(ctx, UpdateUserParams{
					ID:           engineer.Int64,
					Availability: NullAvailabilityStatus{AvailabilityStatus: AvailabilityStatusAvailable, Valid: true},
				})
				if err != nil {
					return fmt.Errorf("failed to free engineer %d: %w", engineer.Int64, err)
				}
			}
		}

		// Step 5: Archive all tasks in the project
		if activeTasksCount > 0 {
			err = q.ArchiveCompletedTasksByProject(ctx, pgtype.Int8{Int64: arg.ProjectID, Valid: true})
			if err != nil {
				return fmt.Errorf("failed to archive project tasks: %w", err)
			}
		}

		result.ArchivedTasksCount = activeTasksCount

		// Step 6: Archive the project
		archivedProject, err := q.ArchiveProject(ctx, ArchiveProjectParams{
			ID:     arg.ProjectID,
			TeamID: arg.TeamID,
		})
		if err != nil {
			return fmt.Errorf("failed to archive project: %w", err)
		}

		result.ArchivedProject = archivedProject
		return nil
	})

	return result, err
}

////////////////////////////////////////////////////////////////////////
// Transaction: CompleteTaskTx
////////////////////////////////////////////////////////////////////////

// CompleteTaskTxParams contains parameters for task completion
type CompleteTaskTxParams struct {
	TaskID int64
}

// CompleteTaskTxResult contains the result of task completion
type CompleteTaskTxResult struct {
	CompletedTask Task
	UpdatedUser   User
}

// CompleteTaskTx marks a task as completed and makes the user available again.
// This is called by engineers when they finish their work.
func (s *Store) CompleteTaskTx(ctx context.Context, arg CompleteTaskTxParams) (CompleteTaskTxResult, error) {
	var result CompleteTaskTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		// Step 1: Get the task and validate
		task, err := q.GetTask(ctx, arg.TaskID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}

		if task.Status == "done" {
			return errors.New("task is already completed")
		}

		if !task.AssigneeID.Valid {
			return errors.New("task is not assigned to anyone")
		}

		// Step 2: Mark task as completed
		completedTask, err := q.UpdateTask(ctx, UpdateTaskParams{
			ID:          arg.TaskID,
			Status:      NullTaskStatus{TaskStatus: "done", Valid: true},
			CompletedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to complete task: %w", err)
		}
		result.CompletedTask = completedTask

		// Step 3: Make user available again
		updatedUser, err := q.UpdateUser(ctx, UpdateUserParams{
			ID:           task.AssigneeID.Int64,
			Availability: NullAvailabilityStatus{AvailabilityStatus: "available", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update user availability: %w", err)
		}
		result.UpdatedUser = updatedUser

		return nil
	})

	return result, err
}

////////////////////////////////////////////////////////////////////////
// Private Helpers
////////////////////////////////////////////////////////////////////////

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
