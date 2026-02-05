package cli

import (
	"testing"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spetersoncode/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyticsSuccessMetrics(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)

	// Create closed tickets with different resolutions
	completedRes := models.ResolutionCompleted
	wontdoRes := models.ResolutionWontDo

	for i := 0; i < 7; i++ {
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Title:      "Completed ticket",
			Status:     models.StatusClosed,
			Resolution: &completedRes,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	for i := 0; i < 3; i++ {
		ticket := &models.Ticket{
			ProjectID:  project.ID,
			Title:      "Wontdo ticket",
			Status:     models.StatusClosed,
			Resolution: &wontdoRes,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	// Verify success metrics
	analyticsRepo := db.NewAnalyticsRepo(database.DB)
	metrics, err := analyticsRepo.GetSuccessMetrics(db.AnalyticsFilter{})
	require.NoError(t, err)

	assert.Equal(t, 10, metrics.TotalClosed)
	assert.Equal(t, 7, metrics.CompletedCount)
	assert.Equal(t, 3, metrics.OtherResolutions)
	assert.InDelta(t, 70.0, metrics.SuccessRate, 0.1)
}

func TestAnalyticsThroughput(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	completedRes := models.ResolutionCompleted
	now := time.Now()

	// Create tickets completed at various times
	// Today
	for i := 0; i < 3; i++ {
		ticket := &models.Ticket{
			ProjectID:   project.ID,
			Title:       "Today ticket",
			Status:      models.StatusClosed,
			Resolution:  &completedRes,
			CompletedAt: &now,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	// 3 days ago (within week)
	threeDaysAgo := now.AddDate(0, 0, -3)
	for i := 0; i < 5; i++ {
		ticket := &models.Ticket{
			ProjectID:   project.ID,
			Title:       "Week ticket",
			Status:      models.StatusClosed,
			Resolution:  &completedRes,
			CompletedAt: &threeDaysAgo,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	// Verify throughput
	analyticsRepo := db.NewAnalyticsRepo(database.DB)
	metrics, err := analyticsRepo.GetThroughputMetrics(db.AnalyticsFilter{})
	require.NoError(t, err)

	assert.Equal(t, 3, metrics.CompletedToday)
	assert.Equal(t, 8, metrics.CompletedWeek) // 3 today + 5 from 3 days ago
}

func TestAnalyticsWIPByStatus(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup project
	projectRepo := db.NewProjectRepo(database.DB)
	project := &models.Project{Key: "TEST", Name: "Test"}
	err := projectRepo.Create(project)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)

	// Create tickets in WIP statuses
	statuses := []struct {
		status models.Status
		count  int
	}{
		{models.StatusReady, 5},
		{models.StatusWorking, 3},
		{models.StatusBlocked, 2},
		{models.StatusHuman, 1},
		{models.StatusReview, 4},
	}

	for _, s := range statuses {
		for i := 0; i < s.count; i++ {
			ticket := &models.Ticket{
				ProjectID: project.ID,
				Title:     "WIP ticket",
				Status:    s.status,
			}
			err := ticketRepo.Create(ticket)
			require.NoError(t, err)
		}
	}

	// Verify WIP
	analyticsRepo := db.NewAnalyticsRepo(database.DB)
	wip, err := analyticsRepo.GetWIPByStatus(db.AnalyticsFilter{})
	require.NoError(t, err)

	// Check totals
	total := 0
	for _, w := range wip {
		total += w.Count
	}
	assert.Equal(t, 15, total)
}

func TestAnalyticsByProject(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()

	// Setup two projects
	projectRepo := db.NewProjectRepo(database.DB)
	project1 := &models.Project{Key: "PROJ1", Name: "Project 1"}
	project2 := &models.Project{Key: "PROJ2", Name: "Project 2"}
	err := projectRepo.Create(project1)
	require.NoError(t, err)
	err = projectRepo.Create(project2)
	require.NoError(t, err)

	ticketRepo := db.NewTicketRepo(database.DB)
	completedRes := models.ResolutionCompleted

	// Create tickets in project 1
	for i := 0; i < 5; i++ {
		ticket := &models.Ticket{
			ProjectID:  project1.ID,
			Title:      "Project 1 ticket",
			Status:     models.StatusClosed,
			Resolution: &completedRes,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	// Create tickets in project 2
	for i := 0; i < 10; i++ {
		ticket := &models.Ticket{
			ProjectID:  project2.ID,
			Title:      "Project 2 ticket",
			Status:     models.StatusClosed,
			Resolution: &completedRes,
		}
		err := ticketRepo.Create(ticket)
		require.NoError(t, err)
	}

	analyticsRepo := db.NewAnalyticsRepo(database.DB)

	// Filter by project 1
	metrics1, err := analyticsRepo.GetSuccessMetrics(db.AnalyticsFilter{ProjectKey: "PROJ1"})
	require.NoError(t, err)
	assert.Equal(t, 5, metrics1.TotalClosed)

	// Filter by project 2
	metrics2, err := analyticsRepo.GetSuccessMetrics(db.AnalyticsFilter{ProjectKey: "PROJ2"})
	require.NoError(t, err)
	assert.Equal(t, 10, metrics2.TotalClosed)

	// All projects
	metricsAll, err := analyticsRepo.GetSuccessMetrics(db.AnalyticsFilter{})
	require.NoError(t, err)
	assert.Equal(t, 15, metricsAll.TotalClosed)
}

func TestAnalyticsResultStruct(t *testing.T) {
	result := AnalyticsResult{
		Success: &db.SuccessMetrics{
			TotalClosed:     10,
			CompletedCount:  7,
			OtherResolutions: 3,
			SuccessRate:     70.0,
		},
		HumanInteraction: &db.HumanInteractionMetrics{
			TotalTickets:          100,
			HumanInterventions:    15,
			HumanInterventionRate: 15.0,
		},
		Throughput: &db.ThroughputMetrics{
			CompletedToday: 5,
			CompletedWeek:  25,
			CompletedMonth: 80,
		},
		WIP: []db.WIPByStatus{
			{Status: "ready", Count: 10},
			{Status: "working", Count: 5},
		},
		CycleTime: []db.CycleTimeByComplexity{
			{Complexity: "small", TicketCount: 20, AvgCycleHours: 2.5},
		},
		Filter: AnalyticsFilter{
			Project:   "TEST",
			TrendDays: 30,
		},
	}

	assert.Equal(t, 10, result.Success.TotalClosed)
	assert.Equal(t, 100, result.HumanInteraction.TotalTickets)
	assert.Equal(t, 5, result.Throughput.CompletedToday)
	assert.Len(t, result.WIP, 2)
	assert.Len(t, result.CycleTime, 1)
	assert.Equal(t, "TEST", result.Filter.Project)
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"working", "Working"},
		{"ready", "Ready"},
		{"blocked", "Blocked (deps)"},
		{"human", "Blocked (human)"},
		{"review", "Review"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := formatStatus(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatComplexity(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"trivial", "Trivial"},
		{"small", "Small"},
		{"medium", "Medium"},
		{"large", "Large"},
		{"xlarge", "X-Large"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := formatComplexity(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestScaleBar(t *testing.T) {
	tests := []struct {
		value    int
		max      int
		width    int
		expected int
	}{
		{0, 10, 20, 0},
		{10, 10, 20, 20},
		{5, 10, 20, 10},
		{1, 100, 20, 1}, // Non-zero value gets at least 1
		{0, 0, 20, 0},   // Zero max returns 0
	}

	for _, tc := range tests {
		result := scaleBar(tc.value, tc.max, tc.width)
		assert.Equal(t, tc.expected, result)
	}
}
