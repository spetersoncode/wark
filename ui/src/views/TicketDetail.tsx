import { AlertTriangle, Copy, GitBranch, Hand } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Markdown } from "../components/Markdown";
import { type Priority, PriorityIndicator } from "../components/PriorityIndicator";
import { type Status, StatusBadge } from "../components/StatusBadge";
import { TicketDetailSkeleton } from "../components/skeletons";
import { type ActivityLog, type Claim, getTicket, type Ticket } from "../lib/api";
import { useAutoRefresh } from "../lib/hooks";
import { cn, formatRelativeTime } from "../lib/utils";

// Map action types to colors for activity timeline dots
const actionColors: Record<string, string> = {
	created: "bg-[var(--foreground-subtle)]",
	claimed: "bg-status-in-progress",
	released: "bg-status-blocked",
	completed: "bg-status-ready",
	accepted: "bg-status-closed",
	rejected: "bg-priority-high",
	blocked: "bg-status-blocked",
	unblocked: "bg-status-ready",
	human_flagged: "bg-status-human",
	retried: "bg-priority-medium",
	transitioned: "bg-status-review",
};

function getActionColor(action: string): string {
	// Check for exact match first
	if (actionColors[action]) {
		return actionColors[action];
	}
	// Check for partial matches
	for (const [key, color] of Object.entries(actionColors)) {
		if (action.toLowerCase().includes(key)) {
			return color;
		}
	}
	return "bg-[var(--foreground-subtle)]";
}

export default function TicketDetail() {
	const { key } = useParams<{ key: string }>();
	const [ticket, setTicket] = useState<Ticket | null>(null);
	const [dependencies, setDependencies] = useState<Ticket[]>([]);
	const [dependents, setDependents] = useState<Ticket[]>([]);
	const [claim, setClaim] = useState<Claim | null>(null);
	const [history, setHistory] = useState<ActivityLog[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [copiedBranch, setCopiedBranch] = useState(false);

	const fetchTicket = useCallback(async () => {
		if (!key) return;
		try {
			const data = await getTicket(key);
			setTicket(data.ticket);
			setDependencies(data.dependencies);
			setDependents(data.dependents);
			setClaim(data.claim || null);
			setHistory(data.history);
			setError(null);
		} catch (e) {
			setError(e instanceof Error ? e.message : "Failed to fetch ticket");
		} finally {
			setLoading(false);
		}
	}, [key]);

	// Initial fetch
	useEffect(() => {
		fetchTicket();
	}, [fetchTicket]);

	// Auto-refresh every 10 seconds when tab is visible
	useAutoRefresh(fetchTicket, [fetchTicket]);

	const copyBranchName = async () => {
		if (!ticket?.branch_name) return;
		try {
			await navigator.clipboard.writeText(ticket.branch_name);
			setCopiedBranch(true);
			setTimeout(() => setCopiedBranch(false), 2000);
		} catch {
			// Ignore clipboard errors
		}
	};

	if (loading) {
		return <TicketDetailSkeleton />;
	}

	if (error || !ticket) {
		return (
			<div className="space-y-4">
				<Link
					to="/tickets"
					className="inline-flex items-center gap-1 text-sm text-[var(--foreground-muted)] hover:text-[var(--foreground)] transition-colors"
				>
					← Tickets
				</Link>
				<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 text-red-700 dark:text-red-300">
					{error || "Ticket not found"}
				</div>
			</div>
		);
	}

	return (
		<div className="space-y-6 max-w-5xl">
			{/* Breadcrumb back link */}
			<Link
				to="/tickets"
				className="inline-flex items-center gap-1 text-sm text-[var(--foreground-muted)] hover:text-[var(--foreground)] transition-colors"
			>
				← Tickets
			</Link>

			{/* Title block */}
			<div className="space-y-3">
				<div className="flex items-center gap-3 flex-wrap">
					<span className="font-mono text-sm text-[var(--foreground-muted)]">
						{ticket.ticket_key}
					</span>
					<StatusBadge status={ticket.status as Status} />
					<PriorityIndicator priority={ticket.priority as Priority} />
				</div>
				<h1 className="text-2xl font-semibold text-[var(--foreground)]">{ticket.title}</h1>

				{/* Human flag banner - prominent amber banner below title */}
				{ticket.human_flag_reason && (
					<div className="flex items-start gap-3 p-3 bg-status-human/10 border border-status-human/30 rounded-md">
						<AlertTriangle className="w-5 h-5 text-status-human flex-shrink-0 mt-0.5" />
						<div>
							<p className="text-sm font-medium text-status-human">Needs Human Attention</p>
							<p className="text-sm text-[var(--foreground-muted)] mt-0.5">
								{ticket.human_flag_reason}
							</p>
						</div>
					</div>
				)}
			</div>

			{/* 2-column layout: Description (wide left) + Details sidebar (narrow right) */}
			<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
				{/* Main content - Description (wider) */}
				<div className="lg:col-span-2 space-y-6">
					{/* Description */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<h2 className="text-sm font-medium text-[var(--foreground-muted)] uppercase tracking-wide mb-3">
							Description
						</h2>
						{ticket.description ? (
							<div className="prose prose-sm dark:prose-invert max-w-none">
								<Markdown>{ticket.description}</Markdown>
							</div>
						) : (
							<p className="text-sm text-[var(--foreground-subtle)] italic">No description</p>
						)}
					</section>

					{/* Activity timeline - simplified with colored dots */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<h2 className="text-sm font-medium text-[var(--foreground-muted)] uppercase tracking-wide mb-3">
							Activity
						</h2>
						{history.length === 0 ? (
							<p className="text-sm text-[var(--foreground-subtle)]">No activity yet</p>
						) : (
							<ul className="space-y-2">
								{history.map((log) => (
									<li key={log.id} className="flex items-start gap-3 text-sm py-1">
										{/* Colored dot based on action type */}
										<span
											className={cn(
												"w-2.5 h-2.5 rounded-full mt-1.5 flex-shrink-0",
												getActionColor(log.action),
											)}
										/>
										<div className="flex-1 min-w-0 flex items-baseline gap-2 flex-wrap">
											<span className="font-medium text-[var(--foreground)]">{log.action}</span>
											{log.actor_id && (
												<span className="text-[var(--foreground-muted)]">{log.actor_id}</span>
											)}
											{log.summary && (
												<span className="text-[var(--foreground-subtle)] truncate">
													{log.summary}
												</span>
											)}
										</div>
										<span className="text-xs text-[var(--foreground-subtle)] flex-shrink-0">
											{formatRelativeTime(log.created_at)}
										</span>
									</li>
								))}
							</ul>
						)}
					</section>
				</div>

				{/* Sidebar - Details (narrower) */}
				<div className="space-y-4">
					{/* Current claim */}
					{claim && claim.status === "active" && (
						<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
							<h2 className="text-sm font-medium text-[var(--foreground-muted)] uppercase tracking-wide mb-3 flex items-center gap-2">
								<Hand className="w-4 h-4" />
								Claimed
							</h2>
							<dl className="space-y-2 text-sm">
								<div className="flex justify-between">
									<dt className="text-[var(--foreground-muted)]">Worker</dt>
									<dd className="font-mono text-xs">{claim.worker_id}</dd>
								</div>
								<div className="flex justify-between">
									<dt className="text-[var(--foreground-muted)]">Expires</dt>
									<dd className="text-xs">{formatRelativeTime(claim.expires_at)}</dd>
								</div>
							</dl>
						</section>
					)}

					{/* Details metadata */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<h2 className="text-sm font-medium text-[var(--foreground-muted)] uppercase tracking-wide mb-3">
							Details
						</h2>
						<dl className="space-y-2 text-sm">
							<div className="flex justify-between">
								<dt className="text-[var(--foreground-muted)]">Complexity</dt>
								<dd>{ticket.complexity}</dd>
							</div>
							<div className="flex justify-between">
								<dt className="text-[var(--foreground-muted)]">Retries</dt>
								<dd>
									{ticket.retry_count} / {ticket.max_retries}
								</dd>
							</div>
							{ticket.resolution && (
								<div className="flex justify-between">
									<dt className="text-[var(--foreground-muted)]">Resolution</dt>
									<dd>{ticket.resolution}</dd>
								</div>
							)}
						</dl>

						{/* Branch name with copy button */}
						{ticket.branch_name && (
							<div className="mt-3 pt-3 border-t border-[var(--border)]">
								<div className="flex items-center gap-2 text-[var(--foreground-muted)] mb-1.5">
									<GitBranch className="w-3.5 h-3.5" />
									<span className="text-xs">Branch</span>
								</div>
								<div className="flex items-center gap-2">
									<code className="font-mono text-xs text-[var(--foreground)] truncate flex-1">
										{ticket.branch_name}
									</code>
									<button
										type="button"
										onClick={copyBranchName}
										className="p-1 rounded hover:bg-[var(--accent)] transition-colors flex-shrink-0"
										title={copiedBranch ? "Copied!" : "Copy branch name"}
									>
										<Copy
											className={cn(
												"w-3.5 h-3.5",
												copiedBranch ? "text-status-ready" : "text-[var(--foreground-muted)]",
											)}
										/>
									</button>
								</div>
							</div>
						)}
					</section>

					{/* Dependencies */}
					{(dependencies.length > 0 || dependents.length === 0) && (
						<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
							<h2 className="text-sm font-medium text-[var(--foreground-muted)] uppercase tracking-wide mb-3">
								Dependencies
							</h2>
							{dependencies.length === 0 ? (
								<p className="text-sm text-[var(--foreground-subtle)]">None</p>
							) : (
								<ul className="space-y-1.5">
									{dependencies.map((dep) => (
										<li key={dep.id}>
											<Link
												to={`/tickets/${dep.ticket_key}`}
												className="flex items-center justify-between p-2 rounded-md hover:bg-[var(--accent)] transition-colors text-sm"
											>
												<span className="font-mono text-xs">{dep.ticket_key}</span>
												<StatusBadge status={dep.status as Status} className="scale-90" />
											</Link>
										</li>
									))}
								</ul>
							)}
						</section>
					)}

					{/* Dependents */}
					{dependents.length > 0 && (
						<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
							<h2 className="text-sm font-medium text-[var(--foreground-muted)] uppercase tracking-wide mb-3">
								Dependents
							</h2>
							<ul className="space-y-1.5">
								{dependents.map((dep) => (
									<li key={dep.id}>
										<Link
											to={`/tickets/${dep.ticket_key}`}
											className="flex items-center justify-between p-2 rounded-md hover:bg-[var(--accent)] transition-colors text-sm"
										>
											<span className="font-mono text-xs">{dep.ticket_key}</span>
											<StatusBadge status={dep.status as Status} className="scale-90" />
										</Link>
									</li>
								))}
							</ul>
						</section>
					)}
				</div>
			</div>
		</div>
	);
}
