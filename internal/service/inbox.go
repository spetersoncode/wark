// Package service provides business logic services for wark.
package service

import (
	"fmt"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/errors"
	"github.com/spetersoncode/wark/internal/models"
)

// InboxService provides business logic for inbox message operations.
// It encapsulates the 3-step response handling flow:
// 1. Record response in DB
// 2. Transition ticket from 'human' → 'ready' if applicable
// 3. Log activity
type InboxService struct {
	inboxRepo    *db.InboxRepo
	ticketRepo   *db.TicketRepo
	claimRepo    *db.ClaimRepo
	activityRepo *db.ActivityRepo
}

// NewInboxService creates a new InboxService.
func NewInboxService(inboxRepo *db.InboxRepo, ticketRepo *db.TicketRepo, claimRepo *db.ClaimRepo, activityRepo *db.ActivityRepo) *InboxService {
	return &InboxService{
		inboxRepo:    inboxRepo,
		ticketRepo:   ticketRepo,
		claimRepo:    claimRepo,
		activityRepo: activityRepo,
	}
}

// RespondResult contains the result of responding to an inbox message.
type RespondResult struct {
	Message       *models.InboxMessage
	TicketUpdated bool
	PreviousStatus models.Status
	NewStatus     models.Status
}

// SendResult contains the result of sending an inbox message.
type SendResult struct {
	Message        *models.InboxMessage
	StatusChanged  bool
	PreviousStatus models.Status
	NewStatus      models.Status
	ClaimReleased  bool
}

// Respond records a response to an inbox message and handles ticket state transitions.
// It performs the 3-step flow:
// 1. Record response in DB via inboxRepo.Respond()
// 2. Transition ticket from 'human' → 'ready' if applicable
// 3. Log activity via activityRepo
func (s *InboxService) Respond(messageID int64, response string) (*RespondResult, error) {
	if response == "" {
		return nil, errors.InvalidArgs("response is required")
	}

	// Step 1: Get the message from DB
	message, err := s.inboxRepo.GetByID(messageID)
	if err != nil {
		return nil, errors.WrapInternal(err, "failed to get message")
	}
	if message == nil {
		return nil, errors.NotFound("message #%d not found", messageID)
	}

	// Check if already responded
	if message.RespondedAt != nil {
		return nil, errors.StateError("message #%d has already been responded to", messageID)
	}

	// Step 2: Record response
	if err := s.inboxRepo.Respond(messageID, response); err != nil {
		return nil, errors.WrapInternal(err, "failed to record response")
	}

	// Step 3: Get the associated ticket
	ticket, err := s.ticketRepo.GetByID(message.TicketID)
	if err != nil {
		return nil, errors.WrapInternal(err, "failed to get ticket")
	}

	result := &RespondResult{
		Message:        message,
		TicketUpdated:  false,
		PreviousStatus: models.StatusHuman,
		NewStatus:      models.StatusHuman,
	}

	// Step 4: If ticket.Status == StatusHuman, transition to Ready
	if ticket != nil && ticket.Status == models.StatusHuman {
		result.PreviousStatus = ticket.Status
		ticket.Status = models.StatusReady
		ticket.RetryCount = 0          // Reset retry count on human response
		ticket.HumanFlagReason = ""    // Clear the flag reason
		if err := s.ticketRepo.Update(ticket); err != nil {
			return nil, errors.WrapInternal(err, "failed to update ticket")
		}
		result.TicketUpdated = true
		result.NewStatus = ticket.Status
	}

	// Step 5: Log activity
	if err := s.activityRepo.LogActionWithDetails(
		message.TicketID,
		models.ActionHumanResponded,
		models.ActorTypeHuman,
		"",
		"Responded to message",
		map[string]interface{}{
			"inbox_message_id": messageID,
			"message_type":     string(message.MessageType),
			"message":          message.Content,
			"response":         response,
		},
	); err != nil {
		return nil, errors.WrapInternal(err, "failed to log activity")
	}

	// Reload message to get updated responded_at
	message, _ = s.inboxRepo.GetByID(messageID)
	result.Message = message

	return result, nil
}

// Send creates an inbox message and handles ticket escalation.
// It performs:
// 1. Validate ticket exists
// 2. Create inbox message via inboxRepo.Create()
// 3. Transition ticket to 'human' status if not already
// 4. Release any active claim
// 5. Log activity
func (s *InboxService) Send(ticketID int64, msgType models.MessageType, content, workerID string) (*SendResult, error) {
	if content == "" {
		return nil, errors.InvalidArgs("message content is required")
	}

	if !msgType.IsValid() {
		return nil, errors.InvalidArgs("invalid message type: %s", msgType)
	}

	// Step 1: Validate ticket exists
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, errors.WrapInternal(err, "failed to get ticket")
	}
	if ticket == nil {
		return nil, errors.NotFound("ticket not found")
	}

	// Step 2: Get active claim (needed for worker ID and release)
	claim, _ := s.claimRepo.GetActiveByTicketID(ticket.ID)

	// Use provided worker ID or get from current claim
	actualWorkerID := workerID
	if actualWorkerID == "" && claim != nil {
		actualWorkerID = claim.WorkerID
	}

	// Step 3: Create inbox message
	inboxMsg := models.NewInboxMessage(ticket.ID, msgType, content, actualWorkerID)
	if err := s.inboxRepo.Create(inboxMsg); err != nil {
		return nil, errors.WrapInternal(err, "failed to create message")
	}

	result := &SendResult{
		Message:        inboxMsg,
		StatusChanged:  false,
		PreviousStatus: ticket.Status,
		NewStatus:      ticket.Status,
		ClaimReleased:  false,
	}

	// Step 4: Transition ticket to human status (escalation flow)
	// Only escalate for message types that require a response
	if msgType.RequiresResponse() && ticket.Status != models.StatusHuman && ticket.Status != models.StatusClosed {
		result.PreviousStatus = ticket.Status
		ticket.Status = models.StatusHuman
		if err := s.ticketRepo.Update(ticket); err != nil {
			return nil, errors.WrapInternal(err, "failed to update ticket status")
		}
		result.StatusChanged = true
		result.NewStatus = ticket.Status
	}

	// Step 5: Release any active claim (only if status changed to human)
	if result.StatusChanged && claim != nil {
		if err := s.claimRepo.Release(claim.ID, models.ClaimStatusReleased); err != nil {
			// Log warning but don't fail - message was already sent
			// In production, you'd want proper logging here
			_ = err
		} else {
			result.ClaimReleased = true
			// Log the claim release as a separate activity for visibility
			s.activityRepo.LogActionWithDetails(
				ticket.ID,
				models.ActionReleased,
				models.ActorTypeAgent,
				claim.WorkerID,
				"Claim released (escalation)",
				map[string]interface{}{
					"worker_id": claim.WorkerID,
					"reason":    "escalation",
				},
			)
		}
	}

	// Step 6: Log activity
	summary := fmt.Sprintf("Sent %s message", msgType)
	if result.StatusChanged {
		summary = fmt.Sprintf("Escalated: %s → %s", result.PreviousStatus, result.NewStatus)
	}
	activityDetails := map[string]interface{}{
		"message_type":     string(msgType),
		"inbox_message_id": inboxMsg.ID,
		"message":          content,
	}
	if result.StatusChanged {
		activityDetails["previous_status"] = string(result.PreviousStatus)
		activityDetails["new_status"] = string(result.NewStatus)
	}
	if result.ClaimReleased {
		activityDetails["claim_released"] = true
	}
	if err := s.activityRepo.LogActionWithDetails(
		ticket.ID,
		models.ActionEscalated,
		models.ActorTypeAgent,
		actualWorkerID,
		summary,
		activityDetails,
	); err != nil {
		return nil, errors.WrapInternal(err, "failed to log activity")
	}

	return result, nil
}
