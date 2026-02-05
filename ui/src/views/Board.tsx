import { CircleCheck, CircleDot, Eye, Filter, UserRound, X } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { KanbanCard, KanbanColumn } from "@/components/board";
import { BoardSkeleton } from "@/components/skeletons";
import {
	listMilestones,
	listProjects,
	listTickets,
	type MilestoneWithStats,
	type ProjectWithStats,
	type Ticket,
	type TicketComplexity,
	type TicketPriority,
	type TicketStatus,
} from "@/lib/api";
import { useAutoRefresh } from "@/lib/hooks";

/**
 * Column configuration - no longer includes "blocked" as a separate column.
 * Blocked tickets appear in Ready with a blocked badge.
 */
const COLUMNS: {
	key: TicketStatus;
	label: string;
	icon: React.ReactNode;
	borderColor: string;
}[] = [
	{
		key: "ready",
		label: "Ready",
		icon: <CircleCheck className="size-4" />,
		borderColor: "border-l-[var(--status-ready)]",
	},
	{
		key: "working",
		label: "Active",
		icon: <CircleDot className="size-4" />,
		borderColor: "border-l-[var(--status-in-progress)]",
	},
	{
		key: "human",
		label: "Human",
		icon: <UserRound className="size-4" />,
		borderColor: "border-l-[var(--status-human)]",
	},
	{
		key: "review",
		label: "Review",
		icon: <Eye className="size-4" />,
		borderColor: "border-l-[var(--status-review)]",
	},
];

const PRIORITIES: { value: TicketPriority; label: string }[] = [
	{ value: "highest", label: "Highest" },
	{ value: "high", label: "High" },
	{ value: "medium", label: "Medium" },
	{ value: "low", label: "Low" },
	{ value: "lowest", label: "Lowest" },
];

const COMPLEXITIES: { value: TicketComplexity; label: string }[] = [
	{ value: "trivial", label: "Trivial" },
	{ value: "small", label: "Small" },
	{ value: "medium", label: "Medium" },
	{ value: "large", label: "Large" },
	{ value: "xlarge", label: "X-Large" },
];

export default function Board() {
	const [tickets, setTickets] = useState<Ticket[]>([]);
	const [projects, setProjects] = useState<ProjectWithStats[]>([]);
	const [milestones, setMilestones] = useState<MilestoneWithStats[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [searchParams, setSearchParams] = useSearchParams();

	// Read filters from URL
	const filterStatus = searchParams.get("status") as TicketStatus | null;
	const filterProject = searchParams.get("project");
	const filterPriority = searchParams.get("priority") as TicketPriority | null;
	const filterComplexity = searchParams.get("complexity") as TicketComplexity | null;
	const filterMilestone = searchParams.get("milestone");

	const hasActiveFilters = filterProject || filterPriority || filterComplexity || filterMilestone;

	// Update a single filter in URL params
	const setFilter = useCallback(
		(key: string, value: string | null) => {
			setSearchParams((prev) => {
				const next = new URLSearchParams(prev);
				if (value) {
					next.set(key, value);
				} else {
					next.delete(key);
				}
				return next;
			});
		},
		[setSearchParams],
	);

	// Clear all filters
	const clearFilters = useCallback(() => {
		setSearchParams((prev) => {
			const next = new URLSearchParams(prev);
			next.delete("project");
			next.delete("priority");
			next.delete("complexity");
			next.delete("milestone");
			// Keep status filter if present (column filter)
			return next;
		});
	}, [setSearchParams]);

	const fetchData = useCallback(async () => {
		try {
			// Fetch tickets with API-supported filters
			const ticketParams: { project?: string; priority?: TicketPriority; milestone?: string; limit: number } = {
				limit: 200,
			};
			if (filterProject) ticketParams.project = filterProject;
			if (filterPriority) ticketParams.priority = filterPriority;
			if (filterMilestone) ticketParams.milestone = filterMilestone;

			const [ticketData, projectData, milestoneData] = await Promise.all([
				listTickets(ticketParams),
				listProjects(),
				listMilestones(),
			]);

			setTickets(ticketData);
			setProjects(projectData);
			setMilestones(milestoneData);
			setError(null);
		} catch (e) {
			setError(e instanceof Error ? e.message : "Failed to fetch data");
		} finally {
			setLoading(false);
		}
	}, [filterProject, filterPriority, filterMilestone]);

	// Initial fetch
	useEffect(() => {
		fetchData();
	}, [fetchData]);

	// Auto-refresh every 10 seconds when tab is visible
	useAutoRefresh(fetchData, [fetchData]);

	// Apply client-side complexity filter (API doesn't support it)
	const filteredTickets = filterComplexity
		? tickets.filter((t) => t.complexity === filterComplexity)
		: tickets;

	// Group tickets by status, with blocked tickets going to ready column
	const ticketsByStatus = COLUMNS.reduce(
		(acc, { key }) => {
			if (key === "ready") {
				// Ready column includes both ready and blocked tickets
				acc[key] = filteredTickets.filter((t) => t.status === "ready" || t.status === "blocked");
			} else {
				acc[key] = filteredTickets.filter((t) => t.status === key);
			}
			return acc;
		},
		{} as Record<TicketStatus, Ticket[]>,
	);

	const visibleColumns = filterStatus ? COLUMNS.filter((c) => c.key === filterStatus) : COLUMNS;

	if (loading) {
		return <BoardSkeleton />;
	}

	if (error) {
		return (
			<div className="flex items-center gap-3 p-4 border border-error/20 bg-error/5 rounded-lg animate-in fade-in duration-200">
				<div className="w-10 h-10 rounded-full bg-error/10 flex items-center justify-center flex-shrink-0">
					<Filter className="w-5 h-5 text-error" />
				</div>
				<div className="flex-1">
					<p className="font-medium text-error">Failed to load board</p>
					<p className="text-sm text-error/80">{error}</p>
				</div>
				<button
					type="button"
					onClick={() => fetchData()}
					className="px-3 py-1.5 text-sm rounded-md text-error hover:bg-error/10 transition-colors"
				>
					Retry
				</button>
			</div>
		);
	}

	return (
		<div className="space-y-4">
			{/* Header */}
			<div className="flex items-center gap-4">
				<h2 className="text-2xl font-bold">Board</h2>
				{filterStatus && (
					<Link
						to="/board"
						className="text-sm text-[var(--foreground-muted)] hover:text-[var(--foreground)]"
					>
						Clear status filter
					</Link>
				)}
			</div>

			{/* Filter Controls */}
			<div className="flex items-center gap-4 flex-wrap p-3 bg-[var(--card)] border border-[var(--border)] rounded-lg">
				<div className="flex items-center gap-2 text-sm text-[var(--foreground-muted)]">
					<Filter className="size-4" />
					<span>Filters:</span>
				</div>

				{/* Project Filter */}
				<select
					value={filterProject || ""}
					onChange={(e) => setFilter("project", e.target.value || null)}
					className="px-3 py-1.5 text-sm rounded-md bg-[var(--background)] border border-[var(--border)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:border-transparent"
				>
					<option value="">All Projects</option>
					{projects.map((p) => (
						<option key={p.key} value={p.key}>
							{p.name}
						</option>
					))}
				</select>

				{/* Priority Filter */}
				<select
					value={filterPriority || ""}
					onChange={(e) => setFilter("priority", e.target.value || null)}
					className="px-3 py-1.5 text-sm rounded-md bg-[var(--background)] border border-[var(--border)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:border-transparent"
				>
					<option value="">All Priorities</option>
					{PRIORITIES.map((p) => (
						<option key={p.value} value={p.value}>
							{p.label}
						</option>
					))}
				</select>

				{/* Complexity Filter */}
				<select
					value={filterComplexity || ""}
					onChange={(e) => setFilter("complexity", e.target.value || null)}
					className="px-3 py-1.5 text-sm rounded-md bg-[var(--background)] border border-[var(--border)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:border-transparent"
				>
					<option value="">All Complexities</option>
					{COMPLEXITIES.map((c) => (
						<option key={c.value} value={c.value}>
							{c.label}
						</option>
					))}
				</select>

				{/* Milestone Filter */}
				<select
					value={filterMilestone || ""}
					onChange={(e) => setFilter("milestone", e.target.value || null)}
					className="px-3 py-1.5 text-sm rounded-md bg-[var(--background)] border border-[var(--border)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:border-transparent"
				>
					<option value="">All Milestones</option>
					{milestones.map((m) => (
						<option key={`${m.project_key}/${m.key}`} value={`${m.project_key}/${m.key}`}>
							{m.project_key}/{m.key} - {m.name}
						</option>
					))}
				</select>

				{/* Clear Filters */}
				{hasActiveFilters && (
					<button
						type="button"
						onClick={clearFilters}
						className="flex items-center gap-1 px-2 py-1.5 text-sm text-[var(--foreground-muted)] hover:text-[var(--foreground)] hover:bg-[var(--accent-muted)] rounded-md transition-colors"
					>
						<X className="size-3" />
						Clear filters
					</button>
				)}

				{/* Active filter count */}
				{hasActiveFilters && (
					<span className="text-xs text-[var(--foreground-muted)]">
						{filteredTickets.length} ticket{filteredTickets.length !== 1 ? "s" : ""} shown
					</span>
				)}
			</div>

			{/* Kanban columns */}
			<div className="flex gap-4 overflow-x-auto pb-4">
				{visibleColumns.map(({ key, label, icon, borderColor }) => (
					<KanbanColumn
						key={key}
						title={label}
						count={ticketsByStatus[key]?.length || 0}
						icon={icon}
						borderColor={borderColor}
						isEmpty={!ticketsByStatus[key]?.length}
					>
						{ticketsByStatus[key]?.map((ticket) => (
							<KanbanCard
								key={ticket.id}
								ticket={ticket}
								showBlockedBadge={ticket.status === "blocked"}
							/>
						))}
					</KanbanColumn>
				))}

			</div>
		</div>
	);
}
