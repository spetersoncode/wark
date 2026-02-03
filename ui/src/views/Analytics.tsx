import { BarChart3, Clock, TrendingUp, Users } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { useRefreshShortcut } from "../components/KeyboardShortcutsProvider";
import { AnalyticsSkeleton } from "../components/skeletons";
import {
	type AnalyticsResult,
	ApiError,
	getAnalytics,
	listProjects,
	type ProjectWithStats,
} from "../lib/api";
import { useAutoRefresh } from "../lib/hooks";
import { cn } from "../lib/utils";

export default function Analytics() {
	const [analytics, setAnalytics] = useState<AnalyticsResult | null>(null);
	const [projects, setProjects] = useState<ProjectWithStats[]>([]);
	const [selectedProject, setSelectedProject] = useState<string>("");
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);

	const fetchData = useCallback(async () => {
		try {
			const [analyticsData, projectsData] = await Promise.all([
				getAnalytics(selectedProject ? { project: selectedProject } : undefined),
				listProjects(),
			]);
			setAnalytics(analyticsData);
			setProjects(projectsData);
			setError(null);
		} catch (e) {
			if (e instanceof ApiError) {
				setError(`API Error: ${e.message}`);
			} else {
				setError("Failed to connect to wark server");
			}
		} finally {
			setLoading(false);
		}
	}, [selectedProject]);

	useEffect(() => {
		fetchData();
	}, [fetchData]);

	// Auto-refresh every 10 seconds when tab is visible
	const { refresh } = useAutoRefresh(fetchData, [fetchData]);

	// Register "r" keyboard shortcut for refresh
	useRefreshShortcut(refresh);

	function handleProjectChange(e: React.ChangeEvent<HTMLSelectElement>) {
		setSelectedProject(e.target.value);
		setLoading(true);
	}

	if (loading) {
		return <AnalyticsSkeleton />;
	}

	if (error) {
		return (
			<div className="flex items-center gap-3 p-4 border border-error/20 bg-error/5 rounded-lg animate-in fade-in duration-200">
				<div className="w-10 h-10 rounded-full bg-error/10 flex items-center justify-center flex-shrink-0">
					<BarChart3 className="w-5 h-5 text-error" />
				</div>
				<div className="flex-1">
					<p className="font-medium text-error">Failed to load analytics</p>
					<p className="text-sm text-error/80">{error}</p>
				</div>
				<button
					type="button"
					onClick={refresh}
					className="px-3 py-1.5 text-sm rounded-md text-error hover:bg-error/10 transition-colors"
				>
					Retry
				</button>
			</div>
		);
	}

	if (!analytics) return null;

	return (
		<div className="space-y-8">
			{/* Header with filters */}
			<div className="flex items-center justify-between">
				<h2 className="text-2xl font-bold">Analytics</h2>
				<select
					value={selectedProject}
					onChange={handleProjectChange}
					className="px-3 py-2 text-sm rounded-md bg-[var(--card)] border border-[var(--border)] focus:outline-none focus:ring-2 focus:ring-[var(--primary)]"
				>
					<option value="">All Projects</option>
					{projects.map((p) => (
						<option key={p.key} value={p.key}>
							{p.name}
						</option>
					))}
				</select>
			</div>

			{/* Success Metrics */}
			<Section title="Success Metrics" icon={<TrendingUp className="w-5 h-5" />}>
				<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
					<MetricCard
						label="Success Rate"
						value={`${analytics.success.success_rate.toFixed(1)}%`}
						subtext={`${analytics.success.completed_count} / ${analytics.success.total_closed} closed`}
						color="text-green-600 dark:text-green-400"
					/>
					<MetricCard
						label="Retry Rate"
						value={`${analytics.success.retry_rate.toFixed(1)}%`}
						subtext={`${analytics.success.tickets_with_retries} tickets needed retries`}
						color="text-orange-600 dark:text-orange-400"
					/>
					<MetricCard
						label="Avg Retries (Failed)"
						value={analytics.success.avg_retries_on_failed.toFixed(1)}
						subtext="Average retries before failure"
						color="text-red-600 dark:text-red-400"
					/>
					<MetricCard
						label="Other Resolutions"
						value={analytics.success.other_resolutions.toString()}
						subtext="wont_do, duplicate, invalid, obsolete"
						color="text-gray-600 dark:text-gray-400"
					/>
				</div>
			</Section>

			{/* Human Interaction */}
			<Section title="Human Interaction" icon={<Users className="w-5 h-5" />}>
				<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
					<MetricCard
						label="Intervention Rate"
						value={`${analytics.human_interaction.human_intervention_rate.toFixed(1)}%`}
						subtext={`${analytics.human_interaction.human_interventions} / ${analytics.human_interaction.total_tickets} tickets`}
						color="text-purple-600 dark:text-purple-400"
					/>
					<MetricCard
						label="Avg Response Time"
						value={formatHours(analytics.human_interaction.avg_response_time_hours)}
						subtext="Time to respond to inbox"
						color="text-blue-600 dark:text-blue-400"
					/>
					<MetricCard
						label="Inbox Messages"
						value={analytics.human_interaction.total_inbox_messages.toString()}
						subtext={`${analytics.human_interaction.responded_messages} responded`}
						color="text-indigo-600 dark:text-indigo-400"
					/>
					<MetricCard
						label="Pending"
						value={(
							analytics.human_interaction.total_inbox_messages -
							analytics.human_interaction.responded_messages
						).toString()}
						subtext="Awaiting response"
						color="text-amber-600 dark:text-amber-400"
					/>
				</div>
			</Section>

			{/* Throughput */}
			<Section title="Throughput" icon={<BarChart3 className="w-5 h-5" />}>
				<div className="grid grid-cols-1 md:grid-cols-3 gap-4">
					<MetricCard
						label="Today"
						value={analytics.throughput.completed_today.toString()}
						subtext="Tickets completed"
						color="text-green-600 dark:text-green-400"
					/>
					<MetricCard
						label="This Week"
						value={analytics.throughput.completed_week.toString()}
						subtext="Tickets completed"
						color="text-blue-600 dark:text-blue-400"
					/>
					<MetricCard
						label="This Month"
						value={analytics.throughput.completed_month.toString()}
						subtext="Tickets completed"
						color="text-purple-600 dark:text-purple-400"
					/>
				</div>
			</Section>

			{/* Current WIP */}
			<Section title="Work in Progress" icon={<Clock className="w-5 h-5" />}>
				{analytics.wip.length > 0 ? (
					<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
						<table className="w-full">
							<thead>
								<tr className="border-b border-[var(--border)]">
									<th className="px-4 py-3 text-left text-sm font-medium text-[var(--muted-foreground)]">
										Status
									</th>
									<th className="px-4 py-3 text-right text-sm font-medium text-[var(--muted-foreground)]">
										Count
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-[var(--border)]">
								{analytics.wip.map((w) => (
									<tr key={w.status}>
										<td className="px-4 py-3">
											<span
												className={cn(
													"inline-flex px-2 py-0.5 text-sm rounded",
													getStatusColor(w.status),
												)}
											>
												{formatStatus(w.status)}
											</span>
										</td>
										<td className="px-4 py-3 text-right font-mono">{w.count}</td>
									</tr>
								))}
							</tbody>
							<tfoot>
								<tr className="border-t border-[var(--border)] bg-[var(--secondary)]">
									<td className="px-4 py-3 font-medium">Total WIP</td>
									<td className="px-4 py-3 text-right font-mono font-bold">
										{analytics.wip.reduce((sum, w) => sum + w.count, 0)}
									</td>
								</tr>
							</tfoot>
						</table>
					</div>
				) : (
					<p className="text-[var(--muted-foreground)]">No work in progress</p>
				)}
			</Section>

			{/* Cycle Time by Complexity */}
			<Section title="Cycle Time by Complexity" icon={<Clock className="w-5 h-5" />}>
				{analytics.cycle_time.length > 0 ? (
					<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
						<table className="w-full">
							<thead>
								<tr className="border-b border-[var(--border)]">
									<th className="px-4 py-3 text-left text-sm font-medium text-[var(--muted-foreground)]">
										Complexity
									</th>
									<th className="px-4 py-3 text-right text-sm font-medium text-[var(--muted-foreground)]">
										Tickets
									</th>
									<th className="px-4 py-3 text-right text-sm font-medium text-[var(--muted-foreground)]">
										Avg Cycle Time
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-[var(--border)]">
								{analytics.cycle_time.map((ct) => (
									<tr key={ct.complexity}>
										<td className="px-4 py-3 capitalize">{ct.complexity}</td>
										<td className="px-4 py-3 text-right font-mono">{ct.ticket_count}</td>
										<td className="px-4 py-3 text-right font-mono">
											{formatHours(ct.avg_cycle_hours)}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				) : (
					<p className="text-[var(--muted-foreground)]">
						No completed tickets with cycle time data
					</p>
				)}
			</Section>

			{/* Completion Trend */}
			<Section title="Completion Trend (Last 30 Days)" icon={<TrendingUp className="w-5 h-5" />}>
				{analytics.completion_trend && analytics.completion_trend.length > 0 ? (
					<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
						<div className="p-4">
							<SimpleTrendChart data={analytics.completion_trend} />
						</div>
					</div>
				) : (
					<p className="text-[var(--muted-foreground)]">No completion data for the period</p>
				)}
			</Section>
		</div>
	);
}

function Section({
	title,
	icon,
	children,
}: {
	title: string;
	icon: React.ReactNode;
	children: React.ReactNode;
}) {
	return (
		<section>
			<div className="flex items-center gap-2 mb-4">
				<span className="text-[var(--muted-foreground)]">{icon}</span>
				<h3 className="text-lg font-semibold">{title}</h3>
			</div>
			{children}
		</section>
	);
}

function MetricCard({
	label,
	value,
	subtext,
	color,
}: {
	label: string;
	value: string;
	subtext: string;
	color: string;
}) {
	return (
		<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4 stagger-item card-hover">
			<p className="text-sm text-[var(--muted-foreground)] mb-1">{label}</p>
			<p className={cn("text-3xl font-bold", color)}>{value}</p>
			<p className="text-xs text-[var(--muted-foreground)] mt-1">{subtext}</p>
		</div>
	);
}

function SimpleTrendChart({ data }: { data: Array<{ date: string; count: number }> }) {
	if (data.length === 0) return null;

	const total = data.reduce((sum, d) => sum + d.count, 0);

	// Format date for display (e.g., "Jan 15" from "2026-01-15")
	const chartData = data.map((d) => ({
		...d,
		displayDate: new Date(d.date).toLocaleDateString("en-US", { month: "short", day: "numeric" }),
	}));

	return (
		<div className="space-y-2">
			<ResponsiveContainer width="100%" height={160}>
				<BarChart data={chartData} margin={{ top: 8, right: 8, left: -16, bottom: 0 }}>
					<XAxis
						dataKey="displayDate"
						tick={{ fontSize: 10, fill: "var(--muted-foreground)" }}
						tickLine={false}
						axisLine={{ stroke: "var(--border)" }}
						interval="preserveStartEnd"
					/>
					<YAxis
						tick={{ fontSize: 10, fill: "var(--muted-foreground)" }}
						tickLine={false}
						axisLine={false}
						allowDecimals={false}
					/>
					<Tooltip
						content={({ active, payload }) => {
							if (!active || !payload?.length) return null;
							const item = payload[0].payload as { date: string; count: number };
							return (
								<div className="bg-[var(--card)] border border-[var(--border)] rounded px-2 py-1 text-sm shadow-lg">
									<p className="font-medium">{item.date}</p>
									<p className="text-[var(--muted-foreground)]">{item.count} completed</p>
								</div>
							);
						}}
					/>
					<Bar dataKey="count" fill="var(--primary)" radius={[2, 2, 0, 0]} />
				</BarChart>
			</ResponsiveContainer>
			<p className="text-sm text-center text-[var(--muted-foreground)]">Total: {total} completed</p>
		</div>
	);
}

function formatHours(hours: number): string {
	if (hours === 0) return "—";
	if (hours < 1) return `${Math.round(hours * 60)}m`;
	if (hours < 24) return `${hours.toFixed(1)}h`;
	const days = hours / 24;
	return `${days.toFixed(1)}d`;
}

function formatStatus(status: string): string {
	return status.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

function getStatusColor(status: string): string {
	switch (status) {
		case "in_progress":
			return "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300";
		case "ready":
			return "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300";
		case "review":
			return "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300";
		case "blocked":
			return "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300";
		case "human":
			return "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300";
		default:
			return "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300";
	}
}
