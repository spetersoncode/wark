import { AlertTriangle, Calendar, Flag, ListTodo } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Markdown } from "../components/Markdown";
import { MilestoneProgress, MilestoneStatusBadge } from "../components/milestone";
import { MilestoneDetailSkeleton } from "../components/skeletons";
import { type Status, StatusBadge } from "../components/StatusBadge";
import {
	ApiError,
	getMilestone,
	getMilestoneTickets,
	type MilestoneWithStats,
	type Ticket,
} from "../lib/api";
import { useAutoRefresh } from "../lib/hooks";

export default function MilestoneDetail() {
	// The key includes path segments: PROJECT/MILESTONE
	const { "*": key } = useParams<{ "*": string }>();
	const [milestone, setMilestone] = useState<MilestoneWithStats | null>(null);
	const [tickets, setTickets] = useState<Ticket[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const fetchData = useCallback(async () => {
		if (!key) return;
		try {
			const [milestoneData, ticketData] = await Promise.all([
				getMilestone(key),
				getMilestoneTickets(key),
			]);
			setMilestone(milestoneData);
			setTickets(ticketData);
			setError(null);
		} catch (e) {
			if (e instanceof ApiError) {
				setError(e.message);
			} else {
				setError("Failed to fetch milestone");
			}
		} finally {
			setLoading(false);
		}
	}, [key]);

	// Initial fetch
	useEffect(() => {
		fetchData();
	}, [fetchData]);

	// Auto-refresh every 10 seconds when tab is visible
	useAutoRefresh(fetchData, [fetchData]);

	if (loading) {
		return <MilestoneDetailSkeleton />;
	}

	if (error || !milestone) {
		return (
			<div className="space-y-4 animate-in fade-in duration-200">
				<Link
					to="/milestones"
					className="inline-flex items-center gap-1 text-sm text-[var(--foreground-muted)] hover:text-[var(--foreground)] transition-colors"
				>
					← Milestones
				</Link>
				<div className="flex items-center gap-3 p-4 border border-error/20 bg-error/5 rounded-lg">
					<div className="w-10 h-10 rounded-full bg-error/10 flex items-center justify-center flex-shrink-0">
						<AlertTriangle className="w-5 h-5 text-error" />
					</div>
					<div className="flex-1">
						<p className="font-medium text-error">
							{error ? "Failed to load milestone" : "Milestone not found"}
						</p>
						<p className="text-sm text-error/80">
							{error || `The milestone "${key}" could not be found.`}
						</p>
					</div>
					{error && (
						<button
							type="button"
							onClick={() => fetchData()}
							className="px-3 py-1.5 text-sm rounded-md text-error hover:bg-error/10 transition-colors"
						>
							Retry
						</button>
					)}
				</div>
			</div>
		);
	}

	const milestoneKey = `${milestone.project_key}/${milestone.key}`;

	// Format target date if present
	const formattedDate = milestone.target_date
		? new Date(milestone.target_date).toLocaleDateString(undefined, {
				weekday: "long",
				month: "long",
				day: "numeric",
				year: "numeric",
			})
		: null;

	// Check if overdue
	const isOverdue =
		milestone.status === "open" &&
		milestone.target_date &&
		new Date(milestone.target_date) < new Date();

	// Group tickets by status for display
	const openTickets = tickets.filter((t) => t.status !== "closed");
	const closedTickets = tickets.filter((t) => t.status === "closed");

	return (
		<div className="space-y-6 max-w-5xl">
			{/* Breadcrumb back link */}
			<Link
				to="/milestones"
				className="inline-flex items-center gap-1 text-sm text-[var(--foreground-muted)] hover:text-[var(--foreground)] transition-colors"
			>
				← Milestones
			</Link>

			{/* Title block */}
			<div className="space-y-3">
				<div className="flex items-center gap-3 flex-wrap">
					<Flag className="w-5 h-5 text-[var(--primary)]" />
					<span className="font-mono text-sm text-[var(--foreground-muted)]">{milestoneKey}</span>
					<MilestoneStatusBadge status={milestone.status} />
				</div>
				<h1 className="text-2xl font-semibold text-[var(--foreground)]">{milestone.name}</h1>
			</div>

			{/* 2-column layout */}
			<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
				{/* Main content */}
				<div className="lg:col-span-2 space-y-6">
					{/* Goal section */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<h2 className="text-sm font-medium text-[var(--foreground-muted)] uppercase tracking-wide mb-3">
							Goal
						</h2>
						{milestone.goal ? (
							<div className="prose prose-sm dark:prose-invert max-w-none">
								<Markdown>{milestone.goal}</Markdown>
							</div>
						) : (
							<p className="text-sm text-[var(--foreground-subtle)] italic">No goal defined</p>
						)}
					</section>

					{/* Tickets section */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<h2 className="text-sm font-medium text-[var(--foreground-muted)] uppercase tracking-wide mb-3 flex items-center gap-2">
							<ListTodo className="w-4 h-4" />
							Tickets ({tickets.length})
						</h2>
						{tickets.length === 0 ? (
							<p className="text-sm text-[var(--foreground-subtle)]">
								No tickets linked to this milestone
							</p>
						) : (
							<div className="space-y-4">
								{/* Open tickets */}
								{openTickets.length > 0 && (
									<div>
										<h3 className="text-xs font-medium text-[var(--foreground-muted)] uppercase mb-2">
											Open ({openTickets.length})
										</h3>
										<ul className="space-y-1">
											{openTickets.map((ticket) => (
												<li key={ticket.id}>
													<Link
														to={`/tickets/${ticket.ticket_key}`}
														className="flex items-center justify-between p-2 rounded-md hover:bg-[var(--accent)] transition-colors text-sm"
													>
														<div className="flex items-center gap-2 min-w-0">
															<span className="font-mono text-xs text-[var(--foreground-muted)]">
																{ticket.ticket_key}
															</span>
															<span className="truncate">{ticket.title}</span>
														</div>
														<StatusBadge
															status={ticket.status as Status}
															className="scale-90 flex-shrink-0"
														/>
													</Link>
												</li>
											))}
										</ul>
									</div>
								)}

								{/* Closed tickets */}
								{closedTickets.length > 0 && (
									<div>
										<h3 className="text-xs font-medium text-[var(--foreground-muted)] uppercase mb-2">
											Closed ({closedTickets.length})
										</h3>
										<ul className="space-y-1">
											{closedTickets.map((ticket) => (
												<li key={ticket.id}>
													<Link
														to={`/tickets/${ticket.ticket_key}`}
														className="flex items-center justify-between p-2 rounded-md hover:bg-[var(--accent)] transition-colors text-sm opacity-60"
													>
														<div className="flex items-center gap-2 min-w-0">
															<span className="font-mono text-xs text-[var(--foreground-muted)]">
																{ticket.ticket_key}
															</span>
															<span className="truncate">{ticket.title}</span>
														</div>
														<StatusBadge
															status={ticket.status as Status}
															className="scale-90 flex-shrink-0"
														/>
													</Link>
												</li>
											))}
										</ul>
									</div>
								)}
							</div>
						)}
					</section>
				</div>

				{/* Sidebar */}
				<div className="space-y-4">
					{/* Progress section */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<h2 className="text-sm font-medium text-[var(--foreground-muted)] uppercase tracking-wide mb-3">
							Progress
						</h2>
						<MilestoneProgress
							percentage={milestone.completion_pct}
							completed={milestone.completed_count}
							total={milestone.ticket_count}
							className="mb-4"
						/>
					</section>

					{/* Details section */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<h2 className="text-sm font-medium text-[var(--foreground-muted)] uppercase tracking-wide mb-3">
							Details
						</h2>
						<dl className="space-y-2 text-sm">
							<div className="flex justify-between">
								<dt className="text-[var(--foreground-muted)]">Project</dt>
								<dd>
									<Link
										to={`/tickets?project=${milestone.project_key}`}
										className="font-mono text-xs hover:underline"
									>
										{milestone.project_key}
									</Link>
								</dd>
							</div>
							{formattedDate && (
								<div className="flex justify-between items-start">
									<dt className="text-[var(--foreground-muted)] flex items-center gap-1.5">
										<Calendar className="w-3.5 h-3.5" />
										Target
									</dt>
									<dd className={isOverdue ? "text-red-500 font-medium" : ""}>
										{formattedDate}
										{isOverdue && <span className="block text-xs">Overdue</span>}
									</dd>
								</div>
							)}
							<div className="flex justify-between">
								<dt className="text-[var(--foreground-muted)]">Created</dt>
								<dd className="text-xs">
									{new Date(milestone.created_at).toLocaleDateString()}
								</dd>
							</div>
						</dl>
					</section>
				</div>
			</div>
		</div>
	);
}
