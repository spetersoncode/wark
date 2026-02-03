import {
	AlertTriangle,
	ArrowLeft,
	Check,
	CheckCircle,
	Clock,
	GitBranch,
	Hand,
	Play,
	RefreshCw,
	RotateCcw,
	User,
	X,
	XCircle,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { Markdown } from "../components/Markdown";
import {
	type ActivityLog,
	acceptTicket,
	type Claim,
	claimTicket,
	closeTicket,
	completeTicket,
	getTicket,
	rejectTicket,
	releaseTicket,
	reopenTicket,
	type Ticket,
} from "../lib/api";
import { useAutoRefresh } from "../lib/hooks";
import { cn, formatRelativeTime, getPriorityColor, getStatusColor } from "../lib/utils";

export default function TicketDetail() {
	const { key } = useParams<{ key: string }>();
	const navigate = useNavigate();
	const [ticket, setTicket] = useState<Ticket | null>(null);
	const [dependencies, setDependencies] = useState<Ticket[]>([]);
	const [dependents, setDependents] = useState<Ticket[]>([]);
	const [claim, setClaim] = useState<Claim | null>(null);
	const [history, setHistory] = useState<ActivityLog[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [actionLoading, setActionLoading] = useState<string | null>(null);

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

	async function handleAction(action: string, fn: () => Promise<unknown>) {
		setActionLoading(action);
		try {
			await fn();
			await fetchTicket();
		} catch (e) {
			setError(e instanceof Error ? e.message : `Failed to ${action}`);
		} finally {
			setActionLoading(null);
		}
	}

	if (loading) {
		return (
			<div className="flex items-center justify-center h-64">
				<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-[var(--primary)]" />
			</div>
		);
	}

	if (error || !ticket) {
		return (
			<div className="space-y-4">
				<button
					type="button"
					onClick={() => navigate(-1)}
					className="flex items-center gap-2 text-sm text-[var(--muted-foreground)] hover:text-[var(--foreground)]"
				>
					<ArrowLeft className="w-4 h-4" />
					Back
				</button>
				<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 text-red-700 dark:text-red-300">
					{error || "Ticket not found"}
				</div>
			</div>
		);
	}

	const isActive = claim?.status === "active";
	const canClaim = ticket.status === "ready" && !isActive;
	const canRelease = isActive;
	const canComplete = ticket.status === "in_progress" && isActive;
	const canAccept = ticket.status === "review";
	const canReject = ticket.status === "review";
	const canClose = ticket.status !== "closed";
	const canReopen = ticket.status === "closed";

	return (
		<div className="space-y-6">
			{/* Header */}
			<div className="flex items-center gap-4">
				<button
					type="button"
					onClick={() => navigate(-1)}
					className="p-2 rounded-md hover:bg-[var(--accent)] transition-colors"
				>
					<ArrowLeft className="w-5 h-5" />
				</button>
				<div className="flex-1">
					<div className="flex items-center gap-3 mb-1">
						<span className="font-mono text-sm text-[var(--muted-foreground)]">
							{ticket.ticket_key}
						</span>
						<span
							className={cn(
								"text-sm px-2 py-0.5 rounded-md font-medium",
								getStatusColor(ticket.status),
								"bg-[var(--secondary)]",
							)}
						>
							{ticket.status.replace("_", " ")}
						</span>
						<span
							className={cn(
								"text-sm px-2 py-0.5 rounded-md font-medium",
								getPriorityColor(ticket.priority),
							)}
						>
							{ticket.priority}
						</span>
					</div>
					<h1 className="text-2xl font-bold">{ticket.title}</h1>
				</div>
				<button
					type="button"
					onClick={fetchTicket}
					className="p-2 rounded-md hover:bg-[var(--accent)] transition-colors"
				>
					<RefreshCw className="w-5 h-5" />
				</button>
			</div>

			{/* Actions */}
			<div className="flex flex-wrap gap-2">
				{canClaim && (
					<ActionButton
						onClick={() => handleAction("claim", () => claimTicket(ticket.ticket_key))}
						loading={actionLoading === "claim"}
						icon={<Hand className="w-4 h-4" />}
					>
						Claim
					</ActionButton>
				)}
				{canRelease && (
					<ActionButton
						onClick={() => handleAction("release", () => releaseTicket(ticket.ticket_key))}
						loading={actionLoading === "release"}
						icon={<X className="w-4 h-4" />}
						variant="secondary"
					>
						Release
					</ActionButton>
				)}
				{canComplete && (
					<ActionButton
						onClick={() => handleAction("complete", () => completeTicket(ticket.ticket_key))}
						loading={actionLoading === "complete"}
						icon={<CheckCircle className="w-4 h-4" />}
						variant="success"
					>
						Complete
					</ActionButton>
				)}
				{canAccept && (
					<ActionButton
						onClick={() => handleAction("accept", () => acceptTicket(ticket.ticket_key))}
						loading={actionLoading === "accept"}
						icon={<Check className="w-4 h-4" />}
						variant="success"
					>
						Accept
					</ActionButton>
				)}
				{canReject && (
					<ActionButton
						onClick={() => {
							const reason = window.prompt("Rejection reason:");
							if (reason) {
								handleAction("reject", () => rejectTicket(ticket.ticket_key, reason));
							}
						}}
						loading={actionLoading === "reject"}
						icon={<XCircle className="w-4 h-4" />}
						variant="danger"
					>
						Reject
					</ActionButton>
				)}
				{canClose && (
					<ActionButton
						onClick={() => {
							const reason = window.prompt("Close reason (optional):");
							handleAction("close", () =>
								closeTicket(ticket.ticket_key, "completed", reason || undefined),
							);
						}}
						loading={actionLoading === "close"}
						icon={<XCircle className="w-4 h-4" />}
						variant="secondary"
					>
						Close
					</ActionButton>
				)}
				{canReopen && (
					<ActionButton
						onClick={() => handleAction("reopen", () => reopenTicket(ticket.ticket_key))}
						loading={actionLoading === "reopen"}
						icon={<RotateCcw className="w-4 h-4" />}
					>
						Reopen
					</ActionButton>
				)}
			</div>

			{/* Info grid */}
			<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
				{/* Main content */}
				<div className="lg:col-span-2 space-y-6">
					{/* Description */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<h2 className="text-lg font-semibold mb-3">Description</h2>
						{ticket.description ? (
							<Markdown>{ticket.description}</Markdown>
						) : (
							<p className="text-sm text-[var(--muted-foreground)] italic">No description</p>
						)}
					</section>

					{/* Activity history */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<h2 className="text-lg font-semibold mb-3">Activity</h2>
						{history.length === 0 ? (
							<p className="text-sm text-[var(--muted-foreground)]">No activity yet</p>
						) : (
							<ul className="space-y-3">
								{history.map((log) => (
									<li key={log.id} className="flex items-start gap-3 text-sm">
										<div className="w-8 h-8 rounded-full bg-[var(--secondary)] flex items-center justify-center flex-shrink-0">
											{log.actor_type === "human" ? (
												<User className="w-4 h-4" />
											) : log.actor_type === "agent" ? (
												<Play className="w-4 h-4" />
											) : (
												<Clock className="w-4 h-4" />
											)}
										</div>
										<div className="flex-1 min-w-0">
											<div className="flex items-center gap-2">
												<span className="font-medium">{log.action}</span>
												{log.actor_id && (
													<span className="text-[var(--muted-foreground)]">by {log.actor_id}</span>
												)}
												<span className="text-[var(--muted-foreground)]">
													{formatRelativeTime(log.created_at)}
												</span>
											</div>
											{log.summary && (
												<p className="text-[var(--muted-foreground)] mt-0.5">{log.summary}</p>
											)}
										</div>
									</li>
								))}
							</ul>
						)}
					</section>
				</div>

				{/* Sidebar */}
				<div className="space-y-6">
					{/* Current claim */}
					{claim && claim.status === "active" && (
						<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
							<h2 className="text-lg font-semibold mb-3 flex items-center gap-2">
								<Hand className="w-5 h-5" />
								Claimed
							</h2>
							<dl className="space-y-2 text-sm">
								<div className="flex justify-between">
									<dt className="text-[var(--muted-foreground)]">Worker</dt>
									<dd className="font-mono">{claim.worker_id}</dd>
								</div>
								<div className="flex justify-between">
									<dt className="text-[var(--muted-foreground)]">Expires</dt>
									<dd>{formatRelativeTime(claim.expires_at)}</dd>
								</div>
							</dl>
						</section>
					)}

					{/* Metadata */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<h2 className="text-lg font-semibold mb-3">Details</h2>
						<dl className="space-y-2 text-sm">
							<div className="flex justify-between">
								<dt className="text-[var(--muted-foreground)]">Complexity</dt>
								<dd>{ticket.complexity}</dd>
							</div>
							<div className="flex justify-between">
								<dt className="text-[var(--muted-foreground)]">Retries</dt>
								<dd>
									{ticket.retry_count} / {ticket.max_retries}
								</dd>
							</div>
							{ticket.resolution && (
								<div className="flex justify-between">
									<dt className="text-[var(--muted-foreground)]">Resolution</dt>
									<dd>{ticket.resolution}</dd>
								</div>
							)}
							{ticket.branch_name && (
								<div className="flex items-center gap-2 pt-2 border-t border-[var(--border)]">
									<GitBranch className="w-4 h-4 text-[var(--muted-foreground)]" />
									<span className="font-mono text-xs truncate">{ticket.branch_name}</span>
								</div>
							)}
							{ticket.human_flag_reason && (
								<div className="flex items-start gap-2 pt-2 border-t border-[var(--border)]">
									<AlertTriangle className="w-4 h-4 text-purple-500 flex-shrink-0 mt-0.5" />
									<span className="text-purple-600 dark:text-purple-400 text-xs">
										{ticket.human_flag_reason}
									</span>
								</div>
							)}
						</dl>
					</section>

					{/* Dependencies */}
					{dependencies.length > 0 && (
						<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
							<h2 className="text-lg font-semibold mb-3">Dependencies ({dependencies.length})</h2>
							<ul className="space-y-2">
								{dependencies.map((dep) => (
									<li key={dep.id}>
										<Link
											to={`/tickets/${dep.ticket_key}`}
											className="flex items-center justify-between p-2 rounded-md hover:bg-[var(--secondary)] transition-colors"
										>
											<span className="font-mono text-sm">{dep.ticket_key}</span>
											<span
												className={cn("text-xs px-1.5 py-0.5 rounded", getStatusColor(dep.status))}
											>
												{dep.status}
											</span>
										</Link>
									</li>
								))}
							</ul>
						</section>
					)}

					{/* Dependents */}
					{dependents.length > 0 && (
						<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
							<h2 className="text-lg font-semibold mb-3">Dependents ({dependents.length})</h2>
							<ul className="space-y-2">
								{dependents.map((dep) => (
									<li key={dep.id}>
										<Link
											to={`/tickets/${dep.ticket_key}`}
											className="flex items-center justify-between p-2 rounded-md hover:bg-[var(--secondary)] transition-colors"
										>
											<span className="font-mono text-sm">{dep.ticket_key}</span>
											<span
												className={cn("text-xs px-1.5 py-0.5 rounded", getStatusColor(dep.status))}
											>
												{dep.status}
											</span>
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

function ActionButton({
	children,
	onClick,
	loading,
	icon,
	variant = "primary",
}: {
	children: React.ReactNode;
	onClick: () => void;
	loading: boolean;
	icon: React.ReactNode;
	variant?: "primary" | "secondary" | "success" | "danger";
}) {
	const variantClasses = {
		primary: "bg-[var(--primary)] text-[var(--primary-foreground)]",
		secondary: "bg-[var(--secondary)] text-[var(--secondary-foreground)]",
		success: "bg-green-600 text-white",
		danger: "bg-red-600 text-white",
	};

	return (
		<button
			type="button"
			onClick={onClick}
			disabled={loading}
			className={cn(
				"flex items-center gap-2 px-4 py-2 text-sm rounded-md transition-opacity disabled:opacity-50",
				variantClasses[variant],
			)}
		>
			{loading ? <RefreshCw className="w-4 h-4 animate-spin" /> : icon}
			{children}
		</button>
	);
}
