package service

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
)

// MilestoneService provides business logic for milestone operations.
type MilestoneService struct {
	milestoneRepo *db.MilestoneRepo
	projectRepo   *db.ProjectRepo
}

// NewMilestoneService creates a new MilestoneService.
func NewMilestoneService(database *sql.DB) *MilestoneService {
	return &MilestoneService{
		milestoneRepo: db.NewMilestoneRepo(database),
		projectRepo:   db.NewProjectRepo(database),
	}
}

// MilestoneError represents a domain-specific error from the milestone service.
type MilestoneError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *MilestoneError) Error() string {
	return e.Message
}

// Error codes for milestone operations
const (
	ErrCodeMilestoneNotFound    = "MILESTONE_NOT_FOUND"
	ErrCodeProjectNotFound      = "PROJECT_NOT_FOUND"
	ErrCodeMilestoneExists      = "MILESTONE_EXISTS"
	ErrCodeInvalidKey           = "INVALID_KEY"
	ErrCodeInvalidStatus        = "INVALID_STATUS"
	ErrCodeInvalidName          = "INVALID_NAME"
	ErrCodeMilestoneDatabase    = "DATABASE_ERROR"
)

func newMilestoneError(code, message string, details map[string]interface{}) *MilestoneError {
	return &MilestoneError{Code: code, Message: message, Details: details}
}

// CreateInput holds the input for creating a milestone.
type CreateInput struct {
	ProjectKey string
	Key        string
	Name       string
	Goal       string
	TargetDate *time.Time
}

// Create creates a new milestone for a project.
func (s *MilestoneService) Create(input CreateInput) (*models.Milestone, error) {
	// Validate key
	if err := models.ValidateMilestoneKey(input.Key); err != nil {
		return nil, newMilestoneError(ErrCodeInvalidKey, err.Error(), nil)
	}

	// Validate name
	if input.Name == "" {
		return nil, newMilestoneError(ErrCodeInvalidName, "milestone name cannot be empty", nil)
	}

	// Look up project
	project, err := s.projectRepo.GetByKey(input.ProjectKey)
	if err != nil {
		return nil, newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to get project: %v", err), nil)
	}
	if project == nil {
		return nil, newMilestoneError(ErrCodeProjectNotFound,
			fmt.Sprintf("project not found: %s", input.ProjectKey),
			map[string]interface{}{"project_key": input.ProjectKey})
	}

	// Check if milestone already exists
	exists, err := s.milestoneRepo.Exists(project.ID, input.Key)
	if err != nil {
		return nil, newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to check milestone existence: %v", err), nil)
	}
	if exists {
		return nil, newMilestoneError(ErrCodeMilestoneExists,
			fmt.Sprintf("milestone already exists: %s/%s", input.ProjectKey, input.Key),
			map[string]interface{}{
				"project_key":   input.ProjectKey,
				"milestone_key": input.Key,
			})
	}

	// Create the milestone
	milestone, err := s.milestoneRepo.Create(project.ID, input.Key, input.Name, input.Goal, input.TargetDate)
	if err != nil {
		return nil, newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to create milestone: %v", err), nil)
	}

	milestone.ProjectKey = project.Key
	return milestone, nil
}

// Get retrieves a milestone by ID.
func (s *MilestoneService) Get(id int64) (*models.Milestone, error) {
	milestone, err := s.milestoneRepo.Get(id)
	if err != nil {
		return nil, newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to get milestone: %v", err), nil)
	}
	if milestone == nil {
		return nil, newMilestoneError(ErrCodeMilestoneNotFound, "milestone not found", map[string]interface{}{"id": id})
	}
	return milestone, nil
}

// GetByKey retrieves a milestone by project key and milestone key.
func (s *MilestoneService) GetByKey(projectKey, milestoneKey string) (*models.Milestone, error) {
	milestone, err := s.milestoneRepo.GetByKey(projectKey, milestoneKey)
	if err != nil {
		return nil, newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to get milestone: %v", err), nil)
	}
	if milestone == nil {
		return nil, newMilestoneError(ErrCodeMilestoneNotFound,
			fmt.Sprintf("milestone not found: %s/%s", projectKey, milestoneKey),
			map[string]interface{}{
				"project_key":   projectKey,
				"milestone_key": milestoneKey,
			})
	}
	return milestone, nil
}

// List retrieves milestones with optional project filter.
func (s *MilestoneService) List(projectKey string) ([]models.MilestoneWithStats, error) {
	var projectID *int64

	if projectKey != "" {
		project, err := s.projectRepo.GetByKey(projectKey)
		if err != nil {
			return nil, newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to get project: %v", err), nil)
		}
		if project == nil {
			return nil, newMilestoneError(ErrCodeProjectNotFound,
				fmt.Sprintf("project not found: %s", projectKey),
				map[string]interface{}{"project_key": projectKey})
		}
		projectID = &project.ID
	}

	milestones, err := s.milestoneRepo.List(projectID)
	if err != nil {
		return nil, newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to list milestones: %v", err), nil)
	}

	return milestones, nil
}

// UpdateInput holds the input for updating a milestone.
type UpdateInput struct {
	Name       *string
	Goal       *string
	TargetDate *time.Time
	Status     *string
	ClearTargetDate bool // Explicitly clear target date
}

// Update updates a milestone.
func (s *MilestoneService) Update(id int64, input UpdateInput) (*models.Milestone, error) {
	// Verify milestone exists
	milestone, err := s.milestoneRepo.Get(id)
	if err != nil {
		return nil, newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to get milestone: %v", err), nil)
	}
	if milestone == nil {
		return nil, newMilestoneError(ErrCodeMilestoneNotFound, "milestone not found", map[string]interface{}{"id": id})
	}

	// Build updates map
	updates := make(map[string]any)

	if input.Name != nil {
		if *input.Name == "" {
			return nil, newMilestoneError(ErrCodeInvalidName, "milestone name cannot be empty", nil)
		}
		updates["name"] = *input.Name
	}

	if input.Goal != nil {
		updates["goal"] = *input.Goal
	}

	if input.Status != nil {
		if err := models.ValidateMilestoneStatus(*input.Status); err != nil {
			return nil, newMilestoneError(ErrCodeInvalidStatus, err.Error(), nil)
		}
		updates["status"] = *input.Status
	}

	if input.ClearTargetDate {
		updates["target_date"] = nil
	} else if input.TargetDate != nil {
		updates["target_date"] = input.TargetDate
	}

	if len(updates) == 0 {
		return milestone, nil
	}

	updated, err := s.milestoneRepo.Update(id, updates)
	if err != nil {
		return nil, newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to update milestone: %v", err), nil)
	}

	return updated, nil
}

// Delete deletes a milestone.
func (s *MilestoneService) Delete(id int64) error {
	// Verify milestone exists
	milestone, err := s.milestoneRepo.Get(id)
	if err != nil {
		return newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to get milestone: %v", err), nil)
	}
	if milestone == nil {
		return newMilestoneError(ErrCodeMilestoneNotFound, "milestone not found", map[string]interface{}{"id": id})
	}

	if err := s.milestoneRepo.Delete(id); err != nil {
		return newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to delete milestone: %v", err), nil)
	}

	return nil
}

// GetLinkedTickets retrieves all tickets linked to a milestone.
func (s *MilestoneService) GetLinkedTickets(id int64) ([]models.Ticket, error) {
	// Verify milestone exists
	milestone, err := s.milestoneRepo.Get(id)
	if err != nil {
		return nil, newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to get milestone: %v", err), nil)
	}
	if milestone == nil {
		return nil, newMilestoneError(ErrCodeMilestoneNotFound, "milestone not found", map[string]interface{}{"id": id})
	}

	tickets, err := s.milestoneRepo.GetLinkedTickets(id)
	if err != nil {
		return nil, newMilestoneError(ErrCodeMilestoneDatabase, fmt.Sprintf("failed to get linked tickets: %v", err), nil)
	}

	return tickets, nil
}

// Achieve marks a milestone as achieved.
func (s *MilestoneService) Achieve(id int64) (*models.Milestone, error) {
	status := models.MilestoneStatusAchieved
	return s.Update(id, UpdateInput{Status: &status})
}

// Abandon marks a milestone as abandoned.
func (s *MilestoneService) Abandon(id int64) (*models.Milestone, error) {
	status := models.MilestoneStatusAbandoned
	return s.Update(id, UpdateInput{Status: &status})
}

// Reopen marks a milestone as open.
func (s *MilestoneService) Reopen(id int64) (*models.Milestone, error) {
	status := models.MilestoneStatusOpen
	return s.Update(id, UpdateInput{Status: &status})
}
