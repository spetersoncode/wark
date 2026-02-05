package db

import (
	"database/sql"
	"fmt"
	"time"
)

// AnalyticsRepo provides database operations for analytics queries.
type AnalyticsRepo struct {
	db *sql.DB
}

// NewAnalyticsRepo creates a new AnalyticsRepo.
func NewAnalyticsRepo(db *sql.DB) *AnalyticsRepo {
	return &AnalyticsRepo{db: db}
}

// AnalyticsFilter defines filters for analytics queries.
type AnalyticsFilter struct {
	ProjectKey string
	Since      *time.Time
	Until      *time.Time
}

// SuccessMetrics contains success-related metrics.
type SuccessMetrics struct {
	TotalClosed        int     `json:"total_closed"`
	CompletedCount     int     `json:"completed_count"`
	OtherResolutions   int     `json:"other_resolutions"`
	SuccessRate        float64 `json:"success_rate"`
	TicketsWithRetries int     `json:"tickets_with_retries"`
	TotalTickets       int     `json:"total_tickets"`
	RetryRate          float64 `json:"retry_rate"`
	AvgRetriesOnFailed float64 `json:"avg_retries_on_failed"`
}

// HumanInteractionMetrics contains human interaction metrics.
type HumanInteractionMetrics struct {
	TotalTickets          int     `json:"total_tickets"`
	HumanInterventions    int     `json:"human_interventions"`
	HumanInterventionRate float64 `json:"human_intervention_rate"`
	TotalInboxMessages    int     `json:"total_inbox_messages"`
	RespondedMessages     int     `json:"responded_messages"`
	AvgResponseTimeHours  float64 `json:"avg_response_time_hours"`
}

// CycleTimeByComplexity contains cycle time grouped by complexity.
type CycleTimeByComplexity struct {
	Complexity   string  `json:"complexity"`
	TicketCount  int     `json:"ticket_count"`
	AvgCycleHours float64 `json:"avg_cycle_hours"`
}

// ThroughputMetrics contains throughput metrics.
type ThroughputMetrics struct {
	CompletedToday  int `json:"completed_today"`
	CompletedWeek   int `json:"completed_week"`
	CompletedMonth  int `json:"completed_month"`
}

// WIPByStatus contains work in progress grouped by status.
type WIPByStatus struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// TrendDataPoint represents a single point in a trend.
type TrendDataPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// GetSuccessMetrics calculates success-related metrics.
func (r *AnalyticsRepo) GetSuccessMetrics(filter AnalyticsFilter) (*SuccessMetrics, error) {
	metrics := &SuccessMetrics{}

	// Build WHERE clause for filters
	where, args := r.buildFilterWhere(filter, "t")

	// Total closed and by resolution
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_closed,
			SUM(CASE WHEN resolution = 'completed' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN resolution != 'completed' OR resolution IS NULL THEN 1 ELSE 0 END) as other
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE t.status = 'closed' %s
	`, where)

	err := r.db.QueryRow(query, args...).Scan(
		&metrics.TotalClosed,
		&metrics.CompletedCount,
		&metrics.OtherResolutions,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get resolution metrics: %w", err)
	}

	if metrics.TotalClosed > 0 {
		metrics.SuccessRate = float64(metrics.CompletedCount) / float64(metrics.TotalClosed) * 100
	}

	// Retry metrics
	where2, args2 := r.buildFilterWhere(filter, "t")
	query2 := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN retry_count > 0 THEN 1 ELSE 0 END) as with_retries
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE t.status = 'closed' %s
	`, where2)

	err = r.db.QueryRow(query2, args2...).Scan(
		&metrics.TotalTickets,
		&metrics.TicketsWithRetries,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get retry metrics: %w", err)
	}

	if metrics.TotalTickets > 0 {
		metrics.RetryRate = float64(metrics.TicketsWithRetries) / float64(metrics.TotalTickets) * 100
	}

	// Avg retries on failed tickets
	where3, args3 := r.buildFilterWhere(filter, "t")
	query3 := fmt.Sprintf(`
		SELECT AVG(retry_count)
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE t.status = 'closed' 
		AND t.resolution != 'completed' 
		AND t.retry_count > 0 %s
	`, where3)

	var avgRetries sql.NullFloat64
	err = r.db.QueryRow(query3, args3...).Scan(&avgRetries)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get avg retries: %w", err)
	}
	if avgRetries.Valid {
		metrics.AvgRetriesOnFailed = avgRetries.Float64
	}

	return metrics, nil
}

// GetHumanInteractionMetrics calculates human interaction metrics.
func (r *AnalyticsRepo) GetHumanInteractionMetrics(filter AnalyticsFilter) (*HumanInteractionMetrics, error) {
	metrics := &HumanInteractionMetrics{}

	// Count tickets that ever hit human status (via activity log)
	where, args := r.buildFilterWhere(filter, "t")
	query := fmt.Sprintf(`
		SELECT 
			COUNT(DISTINCT t.id) as total_tickets
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE 1=1 %s
	`, where)

	err := r.db.QueryRow(query, args...).Scan(&metrics.TotalTickets)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get total tickets: %w", err)
	}

	// Count tickets that had human escalation
	where2, args2 := r.buildFilterWhere(filter, "t")
	query2 := fmt.Sprintf(`
		SELECT COUNT(DISTINCT a.ticket_id)
		FROM activity_log a
		JOIN tickets t ON a.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE a.action = 'escalated' %s
	`, where2)

	err = r.db.QueryRow(query2, args2...).Scan(&metrics.HumanInterventions)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get human interventions: %w", err)
	}

	if metrics.TotalTickets > 0 {
		metrics.HumanInterventionRate = float64(metrics.HumanInterventions) / float64(metrics.TotalTickets) * 100
	}

	// Inbox response time metrics
	where3, args3 := r.buildFilterWhere(filter, "t")
	query3 := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_messages,
			SUM(CASE WHEN m.responded_at IS NOT NULL THEN 1 ELSE 0 END) as responded,
			AVG(
				CASE WHEN m.responded_at IS NOT NULL 
				THEN (julianday(m.responded_at) - julianday(m.created_at)) * 24 
				ELSE NULL END
			) as avg_response_hours
		FROM inbox_messages m
		JOIN tickets t ON m.ticket_id = t.id
		JOIN projects p ON t.project_id = p.id
		WHERE 1=1 %s
	`, where3)

	var avgHours sql.NullFloat64
	err = r.db.QueryRow(query3, args3...).Scan(
		&metrics.TotalInboxMessages,
		&metrics.RespondedMessages,
		&avgHours,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get inbox metrics: %w", err)
	}
	if avgHours.Valid {
		metrics.AvgResponseTimeHours = avgHours.Float64
	}

	return metrics, nil
}

// GetCycleTimeByComplexity calculates average cycle time grouped by complexity.
func (r *AnalyticsRepo) GetCycleTimeByComplexity(filter AnalyticsFilter) ([]CycleTimeByComplexity, error) {
	where, args := r.buildFilterWhere(filter, "t")
	query := fmt.Sprintf(`
		SELECT 
			t.complexity,
			COUNT(*) as ticket_count,
			AVG((julianday(t.completed_at) - julianday(t.created_at)) * 24) as avg_hours
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE t.status = 'closed' 
		AND t.resolution = 'completed' 
		AND t.completed_at IS NOT NULL %s
		GROUP BY t.complexity
		ORDER BY 
			CASE t.complexity
				WHEN 'trivial' THEN 1
				WHEN 'small' THEN 2
				WHEN 'medium' THEN 3
				WHEN 'large' THEN 4
				WHEN 'xlarge' THEN 5
			END
	`, where)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get cycle time by complexity: %w", err)
	}
	defer rows.Close()

	var results []CycleTimeByComplexity
	for rows.Next() {
		var ct CycleTimeByComplexity
		var avgHours sql.NullFloat64
		if err := rows.Scan(&ct.Complexity, &ct.TicketCount, &avgHours); err != nil {
			return nil, fmt.Errorf("failed to scan cycle time: %w", err)
		}
		if avgHours.Valid {
			ct.AvgCycleHours = avgHours.Float64
		}
		results = append(results, ct)
	}

	return results, rows.Err()
}

// GetThroughputMetrics calculates throughput metrics.
func (r *AnalyticsRepo) GetThroughputMetrics(filter AnalyticsFilter) (*ThroughputMetrics, error) {
	metrics := &ThroughputMetrics{}

	// Base filter for project
	projectWhere := ""
	var projectArgs []interface{}
	if filter.ProjectKey != "" {
		projectWhere = " AND p.key = ?"
		projectArgs = append(projectArgs, filter.ProjectKey)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekAgo := today.AddDate(0, 0, -7)
	monthAgo := today.AddDate(0, -1, 0)

	// Completed today
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE t.status = 'closed' 
		AND t.resolution = 'completed' 
		AND t.completed_at >= ? %s
	`, projectWhere)
	args := append([]interface{}{FormatTime(today)}, projectArgs...)
	err := r.db.QueryRow(query, args...).Scan(&metrics.CompletedToday)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get today's throughput: %w", err)
	}

	// Completed this week
	args = append([]interface{}{FormatTime(weekAgo)}, projectArgs...)
	err = r.db.QueryRow(query, args...).Scan(&metrics.CompletedWeek)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get weekly throughput: %w", err)
	}

	// Completed this month
	args = append([]interface{}{FormatTime(monthAgo)}, projectArgs...)
	err = r.db.QueryRow(query, args...).Scan(&metrics.CompletedMonth)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get monthly throughput: %w", err)
	}

	return metrics, nil
}

// GetWIPByStatus returns current work in progress grouped by status.
func (r *AnalyticsRepo) GetWIPByStatus(filter AnalyticsFilter) ([]WIPByStatus, error) {
	where, args := r.buildFilterWhere(filter, "t")
	query := fmt.Sprintf(`
		SELECT t.status, COUNT(*) as count
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE t.status IN ('ready', 'working', 'blocked', 'human', 'review') %s
		GROUP BY t.status
		ORDER BY 
			CASE t.status
				WHEN 'working' THEN 1
				WHEN 'ready' THEN 2
				WHEN 'review' THEN 3
				WHEN 'blocked' THEN 4
				WHEN 'human' THEN 5
			END
	`, where)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get WIP by status: %w", err)
	}
	defer rows.Close()

	var results []WIPByStatus
	for rows.Next() {
		var wip WIPByStatus
		if err := rows.Scan(&wip.Status, &wip.Count); err != nil {
			return nil, fmt.Errorf("failed to scan WIP: %w", err)
		}
		results = append(results, wip)
	}

	return results, rows.Err()
}

// GetCompletionTrend returns daily completion counts for the last N days.
func (r *AnalyticsRepo) GetCompletionTrend(filter AnalyticsFilter, days int) ([]TrendDataPoint, error) {
	if days <= 0 {
		days = 30
	}

	where, args := r.buildFilterWhere(filter, "t")
	startDate := time.Now().AddDate(0, 0, -days)
	args = append([]interface{}{FormatTime(startDate)}, args...)

	query := fmt.Sprintf(`
		SELECT 
			date(t.completed_at) as completion_date,
			COUNT(*) as count
		FROM tickets t
		JOIN projects p ON t.project_id = p.id
		WHERE t.status = 'closed' 
		AND t.resolution = 'completed' 
		AND t.completed_at IS NOT NULL
		AND t.completed_at >= ? %s
		GROUP BY date(t.completed_at)
		ORDER BY completion_date
	`, where)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get completion trend: %w", err)
	}
	defer rows.Close()

	var results []TrendDataPoint
	for rows.Next() {
		var point TrendDataPoint
		var dateStr sql.NullString
		if err := rows.Scan(&dateStr, &point.Count); err != nil {
			return nil, fmt.Errorf("failed to scan trend point: %w", err)
		}
		if dateStr.Valid {
			point.Date = dateStr.String
			results = append(results, point)
		}
	}

	return results, rows.Err()
}

// buildFilterWhere builds the WHERE clause portion for filters.
// The alias parameter is the table alias for tickets (usually "t").
func (r *AnalyticsRepo) buildFilterWhere(filter AnalyticsFilter, alias string) (string, []interface{}) {
	where := ""
	var args []interface{}

	if filter.ProjectKey != "" {
		where += " AND p.key = ?"
		args = append(args, filter.ProjectKey)
	}
	if filter.Since != nil {
		where += fmt.Sprintf(" AND %s.created_at >= ?", alias)
		args = append(args, FormatTime(*filter.Since))
	}
	if filter.Until != nil {
		where += fmt.Sprintf(" AND %s.created_at < ?", alias)
		args = append(args, FormatTime(*filter.Until))
	}

	return where, args
}
