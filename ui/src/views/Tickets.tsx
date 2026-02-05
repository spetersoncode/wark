import { ArrowDown, ArrowUp, ArrowUpDown, Filter, ListTodo, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { EmptyState } from "../components/EmptyState";
import {
	useListNavigationShortcuts,
	useRefreshShortcut,
} from "../components/KeyboardShortcutsProvider";
import { PriorityIndicator } from "../components/PriorityIndicator";
import { StatusBadge } from "../components/StatusBadge";
import { TicketsListSkeleton } from "../components/skeletons";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "../components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "../components/ui/tooltip";
import {
	ApiError,
	listMilestones,
	listProjects,
	listTickets,
	type MilestoneWithStats,
	type ProjectWithStats,
	type Ticket,
	type TicketComplexity,
	type TicketPriority,
	type TicketStatus,
} from "../lib/api";
import { MilestoneBadge } from "../components/milestone";
import { useAutoRefresh } from "../lib/hooks";
import { cn } from "../lib/utils";

type SortField =
	| "ticket_key"
	| "title"
	| "status"
	| "priority"
	| "complexity"
	| "project"
	| "milestone"
	| "created_at";
type SortDirection = "asc" | "desc";

const STATUS_ORDER: Record<TicketStatus, number> = {
	working: 0,
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

const STATUSES: { value: TicketStatus; label: string }[] = [
	{ value: "blocked", label: "Blocked" },
	{ value: "ready", label: "Ready" },
	{ value: "working", label: "In Progress" },
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

export default function Tickets() {
	const [searchParams, setSearchParams] = useSearchParams();
	const navigate = useNavigate();

	// Read filters from URL
	const filterProject = searchParams.get("project");
	const filterStatus = searchParams.get("status") as TicketStatus | null;
	const filterPriority = searchParams.get("priority") as TicketPriority | null;
	const filterComplexity = searchParams.get("complexity") as TicketComplexity | null;
	const filterMilestone = searchParams.get("milestone");

	const activeFilterCount = [filterProject, filterStatus, filterPriority, filterComplexity, filterMilestone].filter(
		Boolean,
	).length;
	const hasActiveFilters = activeFilterCount > 0;

	const [tickets, setTickets] = useState<Ticket[]>([]);
	const [projects, setProjects] = useState<ProjectWithStats[]>([]);
	const [milestones, setMilestones] = useState<MilestoneWithStats[]>([]);
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);
	const [sortField, setSortField] = useState<SortField>("created_at");
	const [sortDirection, setSortDirection] = useState<SortDirection>("desc");
	const [selectedIndex, setSelectedIndex] = useState(-1);

	// Build a map of project keys to names for display
	const projectMap = new Map(projects.map((p) => [p.key, p.name]));

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
				milestone?: string;
			} = {};
			if (filterProject) params.project = filterProject;
			if (filterStatus) params.status = filterStatus;
			if (filterPriority) params.priority = filterPriority;
			if (filterComplexity) params.complexity = filterComplexity;
			if (filterMilestone) params.milestone = filterMilestone;

			const [ticketData, projectData, milestoneData] = await Promise.all([
				listTickets(Object.keys(params).length > 0 ? params : undefined),
				listProjects(),
				listMilestones(),
			]);
			setTickets(ticketData);
			setProjects(projectData);
			setMilestones(milestoneData);
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
	}, [filterProject, filterStatus, filterPriority, filterComplexity, filterMilestone]);

	// Initial fetch
	useEffect(() => {
		fetchData();
	}, [fetchData]);

	// Auto-refresh every 10 seconds when tab is visible
	const { refresh } = useAutoRefresh(fetchData, [fetchData]);

	// Register "r" keyboard shortcut for refresh
	useRefreshShortcut(refresh);

	// Stable compare function for sorting
	const compareTickets = useCallback(
		(a: Ticket, b: Ticket): number => {
			let comparison = 0;

			switch (sortField) {
				case "ticket_key":
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
				case "project":
					comparison = a.project_key.localeCompare(b.project_key);
					break;
				case "milestone":
					// Sort tickets with milestones first, then alphabetically by milestone key
					if (a.milestone_key && b.milestone_key) {
						comparison = a.milestone_key.localeCompare(b.milestone_key);
					} else if (a.milestone_key) {
						comparison = -1;
					} else if (b.milestone_key) {
						comparison = 1;
					} else {
						comparison = 0;
					}
					break;
				case "created_at":
					comparison = new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
					break;
			}

			return sortDirection === "asc" ? comparison : -comparison;
		},
		[sortField, sortDirection],
	);

	// Memoize sorted tickets to avoid recalculating on every render
	const sortedTicketsMemo = useMemo(() => {
		return [...tickets].sort(compareTickets);
	}, [tickets, compareTickets]);

	// List navigation handlers for j/k/Enter shortcuts
	const listNavigationHandlers = useMemo(
		() => ({
			onNext: () => {
				setSelectedIndex((prev) => (prev < sortedTicketsMemo.length - 1 ? prev + 1 : prev));
			},
			onPrevious: () => {
				setSelectedIndex((prev) => (prev > 0 ? prev - 1 : 0));
			},
			onSelect: () => {
				if (selectedIndex >= 0 && selectedIndex < sortedTicketsMemo.length) {
					navigate(`/tickets/${sortedTicketsMemo[selectedIndex].ticket_key}`);
				}
			},
			itemCount: sortedTicketsMemo.length,
		}),
		[sortedTicketsMemo, selectedIndex, navigate],
	);

	// Register j/k/Enter shortcuts for list navigation
	useListNavigationShortcuts(listNavigationHandlers);

	// Reset selected index when tickets change
	// biome-ignore lint/correctness/useExhaustiveDependencies: we intentionally want to reset on tickets change
	useEffect(() => {
		setSelectedIndex(-1);
	}, [tickets]);

	function handleSort(field: SortField) {
		if (sortField === field) {
			setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
		} else {
			setSortField(field);
			setSortDirection("asc");
		}
		// Reset selection when sort changes
		setSelectedIndex(-1);
	}

	// Use the memoized sorted tickets
	const sortedTickets = sortedTicketsMemo;

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
			<TableHead
				className={cn(
					"cursor-pointer select-none hover:text-foreground transition-colors",
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
			</TableHead>
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
		return <TicketsListSkeleton />;
	}

	if (error) {
		return (
			<div className="flex items-center gap-3 p-4 border border-error/20 bg-error/5 rounded-lg animate-in fade-in duration-200">
				<div className="w-10 h-10 rounded-full bg-error/10 flex items-center justify-center flex-shrink-0">
					<Filter className="w-5 h-5 text-error" />
				</div>
				<div className="flex-1">
					<p className="font-medium text-error">Failed to load tickets</p>
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
			<h2 className="text-2xl font-bold">Tickets</h2>

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

				{/* Milestone Filter */}
				<select
					value={filterMilestone || ""}
					onChange={(e) => setFilter("milestone", e.target.value || null)}
					className="px-3 py-1.5 text-sm rounded-md bg-[var(--background)] border border-[var(--border)] focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
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
				<EmptyState
					icon={ListTodo}
					title={hasActiveFilters ? "No tickets match filters" : "No tickets yet"}
					description={
						hasActiveFilters
							? "Try adjusting your filters or clear them to see all tickets."
							: "Tickets are created via the CLI. Run `wark ticket create` to add your first one."
					}
					variant="card"
					action={
						hasActiveFilters ? (
							<button
								type="button"
								onClick={clearFilters}
								className="px-4 py-2 text-sm bg-[var(--secondary)] hover:bg-[var(--accent-muted)] rounded-md transition-colors"
							>
								Clear filters
							</button>
						) : undefined
					}
				/>
			) : (
				<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
					<div className="overflow-x-auto max-h-[calc(100vh-280px)]">
						<Table>
							<TableHeader className="bg-[var(--secondary)] sticky top-0 z-10">
								<TableRow className="hover:bg-transparent">
									<SortHeader field="ticket_key" className="w-28">
										Key
									</SortHeader>
									<SortHeader field="title">Title</SortHeader>
									<SortHeader field="project" className="w-32">
										Project
									</SortHeader>
									<SortHeader field="milestone" className="w-36">
										Milestone
									</SortHeader>
									<SortHeader field="status" className="w-32">
										Status
									</SortHeader>
									<SortHeader field="priority" className="w-28">
										Priority
									</SortHeader>
									<SortHeader field="complexity" className="w-24">
										Complexity
									</SortHeader>
									<SortHeader field="created_at" className="w-28">
										Created
									</SortHeader>
								</TableRow>
							</TableHeader>
							<TableBody>
								{sortedTickets.map((ticket, index) => (
									<TableRow
										key={ticket.id}
										className={cn(
											"hover:bg-muted/50 transition-colors cursor-pointer",
											selectedIndex === index && "bg-[var(--accent)] ring-1 ring-[var(--primary)]",
										)}
										onClick={() => setSelectedIndex(index)}
										onDoubleClick={() => navigate(`/tickets/${ticket.ticket_key}`)}
									>
										<TableCell className="py-4">
											<Link
												to={`/tickets/${ticket.ticket_key}`}
												className="font-mono text-sm text-[var(--primary)] hover:underline"
											>
												{ticket.ticket_key}
											</Link>
										</TableCell>
										<TableCell className="py-4 max-w-[300px]">
											<Tooltip delayDuration={300}>
												<TooltipTrigger asChild>
													<Link
														to={`/tickets/${ticket.ticket_key}`}
														className="block truncate hover:text-[var(--primary)] transition-colors"
													>
														{ticket.title}
													</Link>
												</TooltipTrigger>
												<TooltipContent side="top" className="max-w-md">
													{ticket.title}
												</TooltipContent>
											</Tooltip>
										</TableCell>
										<TableCell className="py-4">
											<span className="text-sm text-muted-foreground">
												{projectMap.get(ticket.project_key) || ticket.project_key}
											</span>
										</TableCell>
										<TableCell className="py-4">
											{ticket.milestone_key ? (
												<MilestoneBadge milestoneKey={ticket.milestone_key} />
											) : (
												<span className="text-sm text-muted-foreground">—</span>
											)}
										</TableCell>
										<TableCell className="py-4">
											<StatusBadge status={ticket.status} />
										</TableCell>
										<TableCell className="py-4">
											<PriorityIndicator priority={ticket.priority} variant="full" />
										</TableCell>
										<TableCell className="py-4 text-sm text-muted-foreground capitalize">
											{ticket.complexity === "xlarge" ? "X-Large" : ticket.complexity}
										</TableCell>
										<TableCell className="py-4 text-sm text-muted-foreground">
											{formatDate(ticket.created_at)}
										</TableCell>
									</TableRow>
								))}
							</TableBody>
						</Table>
					</div>
					{/* Footer with count */}
					<div className="px-4 py-3 border-t border-[var(--border)] bg-[var(--secondary)]">
						<span className="text-sm text-muted-foreground">
							Showing {tickets.length} ticket{tickets.length !== 1 ? "s" : ""}
						</span>
					</div>
				</div>
			)}
		</div>
	);
}
