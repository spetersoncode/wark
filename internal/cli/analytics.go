package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spetersoncode/wark/internal/db"
	"github.com/spf13/cobra"
)

// Analytics command flags
var (
	analyticsProject   string
	analyticsSince     string
	analyticsUntil     string
	analyticsTrendDays int
)

func init() {
	analyticsCmd.Flags().StringVarP(&analyticsProject, "project", "p", "", "Filter by project")
	analyticsCmd.Flags().StringVar(&analyticsSince, "since", "", "Filter from date (YYYY-MM-DD)")
	analyticsCmd.Flags().StringVar(&analyticsUntil, "until", "", "Filter until date (YYYY-MM-DD)")
	analyticsCmd.Flags().IntVar(&analyticsTrendDays, "trend-days", 30, "Number of days for completion trend (1-365)")

	rootCmd.AddCommand(analyticsCmd)
}

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Display analytics and metrics",
	Long: `Display analytics data including success metrics, human interaction stats,
throughput, work in progress, and cycle times.

Metrics shown:
  - Success Metrics: success rate, retry rate, avg retries on failed
  - Human Interaction: intervention rate, inbox response time
  - Throughput: completed today/week/month
  - Work in Progress: tickets by status
  - Cycle Time: average hours by complexity

Examples:
  wark analytics                      # All analytics
  wark analytics --project WEBAPP     # Analytics for specific project
  wark analytics --since 2024-01-01   # Analytics since a date
  wark analytics --json               # Output as JSON`,
	Args: cobra.NoArgs,
	RunE: runAnalytics,
}

// AnalyticsResult represents the full analytics response
type AnalyticsResult struct {
	Success          *db.SuccessMetrics          `json:"success"`
	HumanInteraction *db.HumanInteractionMetrics `json:"human_interaction"`
	Throughput       *db.ThroughputMetrics       `json:"throughput"`
	WIP              []db.WIPByStatus            `json:"wip"`
	CycleTime        []db.CycleTimeByComplexity  `json:"cycle_time"`
	CompletionTrend  []db.TrendDataPoint         `json:"completion_trend"`
	Filter           AnalyticsFilter             `json:"filter"`
}

// AnalyticsFilter shows what filters were applied
type AnalyticsFilter struct {
	Project   string `json:"project,omitempty"`
	Since     string `json:"since,omitempty"`
	Until     string `json:"until,omitempty"`
	TrendDays int    `json:"trend_days"`
}

func runAnalytics(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	repo := db.NewAnalyticsRepo(database.DB)

	// Build filter
	filter := db.AnalyticsFilter{}
	resultFilter := AnalyticsFilter{TrendDays: analyticsTrendDays}

	if analyticsProject != "" {
		filter.ProjectKey = strings.ToUpper(analyticsProject)
		resultFilter.Project = filter.ProjectKey
	}

	if analyticsSince != "" {
		since, err := time.Parse("2006-01-02", analyticsSince)
		if err != nil {
			return fmt.Errorf("invalid --since date format (use YYYY-MM-DD): %w", err)
		}
		filter.Since = &since
		resultFilter.Since = analyticsSince
	}

	if analyticsUntil != "" {
		until, err := time.Parse("2006-01-02", analyticsUntil)
		if err != nil {
			return fmt.Errorf("invalid --until date format (use YYYY-MM-DD): %w", err)
		}
		filter.Until = &until
		resultFilter.Until = analyticsUntil
	}

	if analyticsTrendDays < 1 || analyticsTrendDays > 365 {
		return fmt.Errorf("--trend-days must be between 1 and 365")
	}

	result := AnalyticsResult{
		Filter: resultFilter,
	}

	// Get all metrics
	success, err := repo.GetSuccessMetrics(filter)
	if err != nil {
		return fmt.Errorf("failed to get success metrics: %w", err)
	}
	result.Success = success

	humanInteraction, err := repo.GetHumanInteractionMetrics(filter)
	if err != nil {
		return fmt.Errorf("failed to get human interaction metrics: %w", err)
	}
	result.HumanInteraction = humanInteraction

	throughput, err := repo.GetThroughputMetrics(filter)
	if err != nil {
		return fmt.Errorf("failed to get throughput metrics: %w", err)
	}
	result.Throughput = throughput

	wip, err := repo.GetWIPByStatus(filter)
	if err != nil {
		return fmt.Errorf("failed to get WIP metrics: %w", err)
	}
	result.WIP = wip

	cycleTime, err := repo.GetCycleTimeByComplexity(filter)
	if err != nil {
		return fmt.Errorf("failed to get cycle time metrics: %w", err)
	}
	result.CycleTime = cycleTime

	trend, err := repo.GetCompletionTrend(filter, analyticsTrendDays)
	if err != nil {
		return fmt.Errorf("failed to get completion trend: %w", err)
	}
	result.CompletionTrend = trend

	if IsJSON() {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Display formatted output
	printAnalytics(result)
	return nil
}

func printAnalytics(result AnalyticsResult) {
	title := "Wark Analytics"
	if result.Filter.Project != "" {
		title = fmt.Sprintf("Wark Analytics: %s", result.Filter.Project)
	}
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", 65))

	// Success Metrics
	fmt.Println()
	fmt.Println("Success Metrics")
	fmt.Println(strings.Repeat("-", 30))
	if result.Success != nil {
		fmt.Printf("  Success rate:       %5.1f%%  (%d/%d closed)\n",
			result.Success.SuccessRate,
			result.Success.CompletedCount,
			result.Success.TotalClosed)
		fmt.Printf("  Retry rate:         %5.1f%%  (%d tickets had retries)\n",
			result.Success.RetryRate,
			result.Success.TicketsWithRetries)
		if result.Success.AvgRetriesOnFailed > 0 {
			fmt.Printf("  Avg retries/failed: %5.1f\n", result.Success.AvgRetriesOnFailed)
		} else {
			fmt.Println("  Avg retries/failed:     -")
		}
	} else {
		fmt.Println("  No data")
	}

	// Human Interaction
	fmt.Println()
	fmt.Println("Human Interaction")
	fmt.Println(strings.Repeat("-", 30))
	if result.HumanInteraction != nil {
		fmt.Printf("  Intervention rate:  %5.1f%%  (%d/%d tickets)\n",
			result.HumanInteraction.HumanInterventionRate,
			result.HumanInteraction.HumanInterventions,
			result.HumanInteraction.TotalTickets)
		fmt.Printf("  Inbox messages:     %5d   (%d responded)\n",
			result.HumanInteraction.TotalInboxMessages,
			result.HumanInteraction.RespondedMessages)
		if result.HumanInteraction.AvgResponseTimeHours > 0 {
			fmt.Printf("  Avg response time:  %5.1fh\n", result.HumanInteraction.AvgResponseTimeHours)
		} else {
			fmt.Println("  Avg response time:      -")
		}
	} else {
		fmt.Println("  No data")
	}

	// Throughput
	fmt.Println()
	fmt.Println("Throughput")
	fmt.Println(strings.Repeat("-", 30))
	if result.Throughput != nil {
		fmt.Printf("  Today:              %5d\n", result.Throughput.CompletedToday)
		fmt.Printf("  This week:          %5d\n", result.Throughput.CompletedWeek)
		fmt.Printf("  This month:         %5d\n", result.Throughput.CompletedMonth)
	} else {
		fmt.Println("  No data")
	}

	// WIP by Status
	fmt.Println()
	fmt.Println("Work in Progress")
	fmt.Println(strings.Repeat("-", 30))
	if len(result.WIP) > 0 {
		total := 0
		for _, wip := range result.WIP {
			total += wip.Count
			fmt.Printf("  %-18s %5d\n", formatStatus(wip.Status)+":", wip.Count)
		}
		fmt.Printf("  %-18s %5d\n", "Total:", total)
	} else {
		fmt.Println("  No active work")
	}

	// Cycle Time by Complexity
	fmt.Println()
	fmt.Println("Cycle Time by Complexity")
	fmt.Println(strings.Repeat("-", 30))
	if len(result.CycleTime) > 0 {
		for _, ct := range result.CycleTime {
			fmt.Printf("  %-18s %5.1fh  (%d tickets)\n",
				formatComplexity(ct.Complexity)+":",
				ct.AvgCycleHours,
				ct.TicketCount)
		}
	} else {
		fmt.Println("  No completed tickets")
	}

	// Completion Trend (mini sparkline)
	if len(result.CompletionTrend) > 0 {
		fmt.Println()
		fmt.Printf("Completion Trend (last %d days)\n", result.Filter.TrendDays)
		fmt.Println(strings.Repeat("-", 30))
		printTrendSummary(result.CompletionTrend)
	}
}

func formatStatus(status string) string {
	switch status {
	case "in_progress":
		return "In Progress"
	case "ready":
		return "Ready"
	case "blocked":
		return "Blocked (deps)"
	case "human":
		return "Blocked (human)"
	case "review":
		return "Review"
	default:
		return strings.Title(status)
	}
}

func formatComplexity(complexity string) string {
	switch complexity {
	case "trivial":
		return "Trivial"
	case "small":
		return "Small"
	case "medium":
		return "Medium"
	case "large":
		return "Large"
	case "xlarge":
		return "X-Large"
	default:
		return strings.Title(complexity)
	}
}

func printTrendSummary(trend []db.TrendDataPoint) {
	if len(trend) == 0 {
		fmt.Println("  No completions in period")
		return
	}

	// Find min/max for scaling
	var total, max int
	for _, point := range trend {
		total += point.Count
		if point.Count > max {
			max = point.Count
		}
	}

	// Print summary stats
	daysWithCompletions := len(trend)
	avg := float64(total) / float64(daysWithCompletions)

	fmt.Printf("  Total completed:    %5d\n", total)
	fmt.Printf("  Days with activity: %5d\n", daysWithCompletions)
	fmt.Printf("  Avg per active day: %5.1f\n", avg)

	// Show last 5 days with activity
	if len(trend) > 0 {
		fmt.Println()
		fmt.Println("  Recent:")
		start := len(trend) - 5
		if start < 0 {
			start = 0
		}
		for _, point := range trend[start:] {
			bar := strings.Repeat("█", scaleBar(point.Count, max, 20))
			fmt.Printf("    %s  %s %d\n", point.Date, bar, point.Count)
		}
	}
}

func scaleBar(value, max, width int) int {
	if max == 0 {
		return 0
	}
	scaled := (value * width) / max
	if scaled == 0 && value > 0 {
		scaled = 1 // Ensure at least 1 bar for non-zero values
	}
	return scaled
}
