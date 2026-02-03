import { ArrowDown, ArrowUp, ArrowUpDown, Filter, ListTodo, RefreshCw, X } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
	ApiError,
	listProjects,
	listTickets,
	type ProjectWithStats,
	type Ticket,
	type TicketComplexity,
	type TicketPriority,
	type TicketStatus,
} from "../lib/api";
import { useAutoRefresh } from "../lib/hooks";
import { cn } from "../lib/utils";

type SortField = "ticket_key" | "title" | "status" | "priority" | "complexity" | "created_at";
type SortDirection = "asc" | "desc";

const STATUS_ORDER: Record<TicketStatus, number> = {
	in_progress: 0,
	review: 1,
	human: 2,
	ready: 3,
	blocked: 4,
	closed: 5,
};

const PRIORITY_ORDER: Record<TicketPriority, number> = {
	highest: 0,
	high: 1,
	medium: 2,
	low: 3,
	lowest: 4,
};

const COMPLEXITY_ORDER: Record<TicketComplexity, number> = {
	trivial: 0,
	small: 1,
	medium: 2,
	large: 3,
	xlarge: 4,
};

const STATUS_STYLES: Record<TicketStatus, string> = {
	blocked: "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
	ready: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
	in_progress: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
	human: "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400",
	review: "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400",
	closed: "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-500",
};

const STATUSES: { value: TicketStatus; label: string }[] = [
	{ value: "blocked", label: "Blocked" },
	{ value: "ready", label: "Ready" },
	{ value: "in_progress", label: "In Progress" },
	{ value: "human", label: "Human" },
	{ value: "review", label: "Review" },
	{ value: "closed", label: "Closed" },
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

const PRIORITY_STYLES: Record<TicketPriority, string> = {
	highest: "text-red-600 dark:text-red-400",
	high: "text-orange-600 dark:text-orange-400",
	medium: "text-yellow-600 dark:text-yellow-400",
	low: "text-blue-600 dark:text-blue-400",
	lowest: "text-gray-600 dark:text-gray-400",
};

export default function Tickets() {
	const [searchParams, setSearchParams] = useSearchParams();

	// Read filters from URL
	const filterProject = searchParams.get("project");
	const filterStatus = searchParams.get("status") as TicketStatus | null;
	const filterPriority = searchParams.get("priority") as TicketPriority | null;
	const filterComplexity = searchParams.get("complexity") as TicketComplexity | null;

	const activeFilterCount = [filterProject, filterStatus, filterPriority, filterComplexity].filter(
		Boolean,
	).length;
	const hasActiveFilters = activeFilterCount > 0;

	const [tickets, setTickets] = useState<Ticket[]>([]);
	const [projects, setProjects] = useState<ProjectWithStats[]>([]);
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);
	const [sortField, setSortField] = useState<SortField>("created_at");
	const [sortDirection, setSortDirection] = useState<SortDirection>("desc");

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
		setSearchParams(new URLSearchParams());
	}, [setSearchParams]);

	const fetchData = useCallback(async () => {
		try {
			const params: {
				project?: string;
				status?: TicketStatus;
				priority?: TicketPriority;
				complexity?: TicketComplexity;
			} = {};
			if (filterProject) params.project = filterProject;
			if (filterStatus) params.status = filterStatus;
			if (filterPriority) params.priority = filterPriority;
			if (filterComplexity) params.complexity = filterComplexity;

			const [ticketData, projectData] = await Promise.all([
				listTickets(Object.keys(params).length > 0 ? params : undefined),
				listProjects(),
			]);
			setTickets(ticketData);
			setProjects(projectData);
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
	}, [filterProject, filterStatus, filterPriority, filterComplexity]);

	// Initial fetch
	useEffect(() => {
		fetchData();
	}, [fetchData]);

	// Auto-refresh every 10 seconds when tab is visible
	const { refreshing, refresh: handleRefresh } = useAutoRefresh(fetchData, [fetchData]);

	function handleSort(field: SortField) {
		if (sortField === field) {
			setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
		} else {
			setSortField(field);
			setSortDirection("asc");
		}
	}

	function compareTickets(a: Ticket, b: Ticket): number {
		let comparison = 0;

		switch (sortField) {
			case "ticket_key":
				// Sort by project key first, then by number
				if (a.project_key !== b.project_key) {
					comparison = a.project_key.localeCompare(b.project_key);
				} else {
					comparison = a.number - b.number;
				}
				break;
			case "title":
				comparison = a.title.localeCompare(b.title);
				break;
			case "status":
				comparison = STATUS_ORDER[a.status] - STATUS_ORDER[b.status];
				break;
			case "priority":
				comparison = PRIORITY_ORDER[a.priority] - PRIORITY_ORDER[b.priority];
				break;
			case "complexity":
				comparison = COMPLEXITY_ORDER[a.complexity] - COMPLEXITY_ORDER[b.complexity];
				break;
			case "created_at":
				comparison = new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
				break;
		}

		return sortDirection === "asc" ? comparison : -comparison;
	}

	const sortedTickets = [...tickets].sort(compareTickets);

	function SortHeader({
		field,
		children,
		className,
	}: {
		field: SortField;
		children: React.ReactNode;
		className?: string;
	}) {
		const isActive = sortField === field;
		return (
			<th
				className={cn(
					"px-4 py-3 text-left text-sm font-medium text-[var(--muted-foreground)] cursor-pointer select-none hover:text-[var(--foreground)] transition-colors",
					className,
				)}
				onClick={() => handleSort(field)}
			>
				<div className="flex items-center gap-1">
					{children}
					{isActive ? (
						sortDirection === "asc" ? (
							<ArrowUp className="w-4 h-4" />
						) : (
							<ArrowDown className="w-4 h-4" />
						)
					) : (
						<ArrowUpDown className="w-4 h-4 opacity-40" />
					)}
				</div>
			</th>
		);
	}

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString("en-US", {
			month: "short",
			day: "numeric",
			year: date.getFullYear() !== new Date().getFullYear() ? "numeric" : undefined,
		});
	}

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
			{/* Header with refresh */}
			<div className="flex items-center justify-between">
				<h2 className="text-2xl font-bold">Tickets</h2>
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

				{/* Status Filter */}
				<select
					value={filterStatus || ""}
					onChange={(e) => setFilter("status", e.target.value || null)}
					className="px-3 py-1.5 text-sm rounded-md bg-[var(--background)] border border-[var(--border)] focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
				>
					<option value="">All Statuses</option>
					{STATUSES.map((s) => (
						<option key={s.value} value={s.value}>
							{s.label}
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
						Clear filters ({activeFilterCount})
					</button>
				)}

				{/* Active filter count / ticket count */}
				{hasActiveFilters && (
					<span className="text-xs text-[var(--muted-foreground)]">
						{tickets.length} ticket{tickets.length !== 1 ? "s" : ""} shown
					</span>
				)}
			</div>

			{/* Tickets table */}
			{tickets.length === 0 ? (
				<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-8 text-center">
					<ListTodo className="w-12 h-12 mx-auto mb-4 text-[var(--muted-foreground)]" />
					<p className="text-[var(--muted-foreground)]">No tickets found</p>
				</div>
			) : (
				<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
					<div className="overflow-x-auto">
						<table className="w-full">
							<thead className="bg-[var(--secondary)] border-b border-[var(--border)]">
								<tr>
									<SortHeader field="ticket_key" className="w-28">
										Key
									</SortHeader>
									<SortHeader field="title">Title</SortHeader>
									<SortHeader field="status" className="w-28">
										Status
									</SortHeader>
									<SortHeader field="priority" className="w-24">
										Priority
									</SortHeader>
									<SortHeader field="complexity" className="w-24">
										Complexity
									</SortHeader>
									<SortHeader field="created_at" className="w-28">
										Created
									</SortHeader>
								</tr>
							</thead>
							<tbody className="divide-y divide-[var(--border)]">
								{sortedTickets.map((ticket) => (
									<tr key={ticket.id} className="hover:bg-[var(--accent)] transition-colors">
										<td className="px-4 py-3">
											<Link
												to={`/tickets/${ticket.ticket_key}`}
												className="font-mono text-sm text-[var(--primary)] hover:underline"
											>
												{ticket.ticket_key}
											</Link>
										</td>
										<td className="px-4 py-3">
											<Link
												to={`/tickets/${ticket.ticket_key}`}
												className="hover:text-[var(--primary)] transition-colors"
											>
												{ticket.title}
											</Link>
										</td>
										<td className="px-4 py-3">
											<span
												className={cn(
													"inline-flex px-2 py-0.5 text-xs font-medium rounded-full",
													STATUS_STYLES[ticket.status],
												)}
											>
												{ticket.status.replace("_", " ")}
											</span>
										</td>
										<td className="px-4 py-3">
											<span
												className={cn(
													"text-sm font-medium capitalize",
													PRIORITY_STYLES[ticket.priority],
												)}
											>
												{ticket.priority}
											</span>
										</td>
										<td className="px-4 py-3 text-sm text-[var(--muted-foreground)] capitalize">
											{ticket.complexity === "xlarge" ? "X-Large" : ticket.complexity}
										</td>
										<td className="px-4 py-3 text-sm text-[var(--muted-foreground)]">
											{formatDate(ticket.created_at)}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				</div>
			)}
		</div>
	);
}
