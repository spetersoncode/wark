import {
	ArrowDown,
	ArrowUp,
	ArrowUpDown,
	ListTodo,
	RefreshCw,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
	ApiError,
	listTickets,
	type Ticket,
	type TicketComplexity,
	type TicketPriority,
	type TicketStatus,
} from "../lib/api";
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

const PRIORITY_STYLES: Record<TicketPriority, string> = {
	highest: "text-red-600 dark:text-red-400",
	high: "text-orange-600 dark:text-orange-400",
	medium: "text-yellow-600 dark:text-yellow-400",
	low: "text-blue-600 dark:text-blue-400",
	lowest: "text-gray-600 dark:text-gray-400",
};

export default function Tickets() {
	const [searchParams] = useSearchParams();
	const projectFilter = searchParams.get("project");

	const [tickets, setTickets] = useState<Ticket[]>([]);
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);
	const [refreshing, setRefreshing] = useState(false);
	const [sortField, setSortField] = useState<SortField>("created_at");
	const [sortDirection, setSortDirection] = useState<SortDirection>("desc");

	const fetchTickets = useCallback(async () => {
		try {
			const data = await listTickets(projectFilter ? { project: projectFilter } : undefined);
			setTickets(data);
			setError(null);
		} catch (e) {
			if (e instanceof ApiError) {
				setError(`API Error: ${e.message}`);
			} else {
				setError("Failed to connect to wark server");
			}
		} finally {
			setLoading(false);
			setRefreshing(false);
		}
	}, [projectFilter]);

	useEffect(() => {
		fetchTickets();
	}, [fetchTickets]);

	function handleRefresh() {
		setRefreshing(true);
		fetchTickets();
	}

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
		<div className="space-y-8">
			{/* Header with refresh */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-3">
					<h2 className="text-2xl font-bold">Tickets</h2>
					{projectFilter && (
						<span className="px-2 py-1 text-sm font-mono bg-[var(--secondary)] rounded">
							{projectFilter}
						</span>
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
									<tr
										key={ticket.id}
										className="hover:bg-[var(--accent)] transition-colors"
									>
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
