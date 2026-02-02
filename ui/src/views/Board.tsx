import { AlertTriangle, CheckCircle, CircleDot, Clock, Eye, Filter, RefreshCw, X } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
	listProjects,
	listTickets,
	type ProjectWithStats,
	type Ticket,
	type TicketComplexity,
	type TicketPriority,
	type TicketStatus,
} from "../lib/api";
import { cn, getPriorityColor } from "../lib/utils";

const STATUSES: { key: TicketStatus; label: string; icon: React.ReactNode; color: string }[] = [
	{
		key: "ready",
		label: "Ready",
		icon: <CheckCircle className="w-4 h-4" />,
		color: "border-green-500",
	},
	{
		key: "in_progress",
		label: "In Progress",
		icon: <Clock className="w-4 h-4" />,
		color: "border-blue-500",
	},
	{
		key: "human",
		label: "Human",
		icon: <AlertTriangle className="w-4 h-4" />,
		color: "border-purple-500",
	},
	{
		key: "review",
		label: "Review",
		icon: <Eye className="w-4 h-4" />,
		color: "border-yellow-500",
	},
	{
		key: "closed",
		label: "Closed",
		icon: <CircleDot className="w-4 h-4" />,
		color: "border-gray-500",
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
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [refreshing, setRefreshing] = useState(false);
	const [searchParams, setSearchParams] = useSearchParams();

	// Read filters from URL
	const filterStatus = searchParams.get("status") as TicketStatus | null;
	const filterProject = searchParams.get("project");
	const filterPriority = searchParams.get("priority") as TicketPriority | null;
	const filterComplexity = searchParams.get("complexity") as TicketComplexity | null;

	const hasActiveFilters = filterProject || filterPriority || filterComplexity;

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
			// Keep status filter if present (column filter)
			return next;
		});
	}, [setSearchParams]);

	const fetchData = useCallback(async () => {
		try {
			// Fetch tickets with API-supported filters
			const ticketParams: { project?: string; priority?: TicketPriority; limit: number } = {
				limit: 200,
			};
			if (filterProject) ticketParams.project = filterProject;
			if (filterPriority) ticketParams.priority = filterPriority;

			const [ticketData, projectData] = await Promise.all([
				listTickets(ticketParams),
				listProjects(),
			]);

			setTickets(ticketData);
			setProjects(projectData);
			setError(null);
		} catch (e) {
			setError(e instanceof Error ? e.message : "Failed to fetch data");
		} finally {
			setLoading(false);
			setRefreshing(false);
		}
	}, [filterProject, filterPriority]);

	useEffect(() => {
		fetchData();
	}, [fetchData]);

	function handleRefresh() {
		setRefreshing(true);
		fetchData();
	}

	// Apply client-side complexity filter (API doesn't support it)
	const filteredTickets = filterComplexity
		? tickets.filter((t) => t.complexity === filterComplexity)
		: tickets;

	const ticketsByStatus = STATUSES.reduce(
		(acc, { key }) => {
			acc[key] = filteredTickets.filter((t) => t.status === key);
			return acc;
		},
		{} as Record<TicketStatus, Ticket[]>,
	);

	const visibleStatuses = filterStatus ? STATUSES.filter((s) => s.key === filterStatus) : STATUSES;

	if (loading) {
		return (
			<div className="flex items-center justify-center h-64">
				<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-[var(--primary)]" />
			</div>
		);
	}

	if (error) {
		return (
			<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 text-red-700 dark:text-red-300">
				{error}
			</div>
		);
	}

	return (
		<div className="space-y-4">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-4">
					<h2 className="text-2xl font-bold">Board</h2>
					{filterStatus && (
						<Link
							to="/board"
							className="text-sm text-[var(--muted-foreground)] hover:text-[var(--foreground)]"
						>
							Clear status filter
						</Link>
					)}
				</div>
				<button
					type="button"
					onClick={handleRefresh}
					disabled={refreshing}
					className="flex items-center gap-2 px-3 py-2 text-sm rounded-md bg-[var(--secondary)] hover:bg-[var(--accent)] transition-colors disabled:opacity-50"
				>
					<RefreshCw className={cn("w-4 h-4", refreshing && "animate-spin")} />
					Refresh
				</button>
			</div>

			{/* Filter Controls */}
			<div className="flex items-center gap-4 flex-wrap p-3 bg-[var(--card)] border border-[var(--border)] rounded-lg">
				<div className="flex items-center gap-2 text-sm text-[var(--muted-foreground)]">
					<Filter className="w-4 h-4" />
					<span>Filters:</span>
				</div>

				{/* Project Filter */}
				<select
					value={filterProject || ""}
					onChange={(e) => setFilter("project", e.target.value || null)}
					className="px-3 py-1.5 text-sm rounded-md bg-[var(--background)] border border-[var(--border)] focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
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
					className="px-3 py-1.5 text-sm rounded-md bg-[var(--background)] border border-[var(--border)] focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
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
					className="px-3 py-1.5 text-sm rounded-md bg-[var(--background)] border border-[var(--border)] focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
				>
					<option value="">All Complexities</option>
					{COMPLEXITIES.map((c) => (
						<option key={c.value} value={c.value}>
							{c.label}
						</option>
					))}
				</select>

				{/* Clear Filters */}
				{hasActiveFilters && (
					<button
						type="button"
						onClick={clearFilters}
						className="flex items-center gap-1 px-2 py-1.5 text-sm text-[var(--muted-foreground)] hover:text-[var(--foreground)] hover:bg-[var(--accent)] rounded-md transition-colors"
					>
						<X className="w-3 h-3" />
						Clear filters
					</button>
				)}

				{/* Active filter count */}
				{hasActiveFilters && (
					<span className="text-xs text-[var(--muted-foreground)]">
						{filteredTickets.length} ticket{filteredTickets.length !== 1 ? "s" : ""} shown
					</span>
				)}
			</div>

			{/* Kanban columns */}
			<div className="flex gap-4 overflow-x-auto pb-4">
				{visibleStatuses.map(({ key, label, icon, color }) => (
					<div
						key={key}
						className={cn(
							"flex-shrink-0 w-72 bg-[var(--card)] border border-[var(--border)] rounded-lg",
							"border-t-4",
							color,
						)}
					>
						{/* Column header */}
						<div className="p-3 border-b border-[var(--border)] flex items-center justify-between">
							<div className="flex items-center gap-2">
								{icon}
								<span className="font-medium">{label}</span>
							</div>
							<span className="text-sm text-[var(--muted-foreground)]">
								{ticketsByStatus[key]?.length || 0}
							</span>
						</div>

						{/* Column content */}
						<div className="p-2 space-y-2 max-h-[calc(100vh-16rem)] overflow-y-auto">
							{ticketsByStatus[key]?.length === 0 ? (
								<p className="text-sm text-[var(--muted-foreground)] text-center py-4">
									No tickets
								</p>
							) : (
								ticketsByStatus[key]?.map((ticket) => (
									<TicketCard key={ticket.id} ticket={ticket} />
								))
							)}
						</div>
					</div>
				))}
			</div>
		</div>
	);
}

function TicketCard({ ticket }: { ticket: Ticket }) {
	return (
		<Link
			to={`/tickets/${ticket.ticket_key}`}
			className="block p-3 bg-[var(--background)] border border-[var(--border)] rounded-md hover:border-[var(--primary)] transition-colors"
		>
			<div className="flex items-start justify-between gap-2 mb-2">
				<span className="font-mono text-xs text-[var(--muted-foreground)]">
					{ticket.ticket_key}
				</span>
				<span
					className={cn(
						"text-xs px-1.5 py-0.5 rounded font-medium",
						getPriorityColor(ticket.priority),
					)}
				>
					{ticket.priority}
				</span>
			</div>
			<p className="text-sm font-medium line-clamp-2">{ticket.title}</p>
			{ticket.human_flag_reason && (
				<p className="text-xs text-purple-600 dark:text-purple-400 mt-1 truncate">
					⚠ {ticket.human_flag_reason}
				</p>
			)}
			{ticket.branch_name && (
				<p className="text-xs text-[var(--muted-foreground)] mt-1 truncate font-mono">
					{ticket.branch_name}
				</p>
			)}
		</Link>
	);
}
