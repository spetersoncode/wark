import {
	AlertTriangle,
	CheckCircle2,
	CircleDot,
	Clock,
	Inbox,
	TrendingDown,
	TrendingUp,
	UserRound,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { PriorityIndicator } from "@/components/PriorityIndicator";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError, getStatus, listTickets, type StatusResult, type Ticket } from "@/lib/api";
import { useAutoRefresh } from "@/lib/hooks";
import { cn } from "@/lib/utils";

interface NeedsAttentionData {
	expiringClaims: StatusResult["expiring_soon"];
	blockedTickets: Ticket[];
	humanFlaggedTickets: Ticket[];
}

export default function Dashboard() {
	const [status, setStatus] = useState<StatusResult | null>(null);
	const [attentionData, setAttentionData] = useState<NeedsAttentionData | null>(null);
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);
	const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

	const fetchData = useCallback(async () => {
		try {
			// Fetch all data in parallel
			const [statusData, blockedTickets, humanTickets] = await Promise.all([
				getStatus(),
				listTickets({ status: "blocked", limit: 5 }),
				listTickets({ status: "human", limit: 5 }),
			]);

			setStatus(statusData);
			setAttentionData({
				expiringClaims: statusData.expiring_soon,
				blockedTickets,
				humanFlaggedTickets: humanTickets,
			});
			setLastUpdated(new Date());
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
	}, []);

	// Initial fetch
	useEffect(() => {
		fetchData();
	}, [fetchData]);

	// Auto-refresh every 10 seconds when tab is visible (no manual refresh button needed)
	useAutoRefresh(fetchData, [fetchData]);

	if (loading) {
		return <DashboardSkeleton />;
	}

	if (error) {
		return (
			<div className="flex items-center gap-3 p-4 border border-error/20 bg-error/5 rounded-lg animate-in fade-in duration-200">
				<div className="w-10 h-10 rounded-full bg-error/10 flex items-center justify-center flex-shrink-0">
					<AlertTriangle className="w-5 h-5 text-error" />
				</div>
				<div className="flex-1">
					<p className="font-medium text-error">Failed to load dashboard</p>
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

	if (!status) return null;

	// Check if there are any items needing attention
	const hasAttentionItems =
		(attentionData?.expiringClaims.length ?? 0) > 0 ||
		(attentionData?.blockedTickets.length ?? 0) > 0 ||
		(attentionData?.humanFlaggedTickets.length ?? 0) > 0;

	return (
		<div className="space-y-6">
			{/* Header with last updated timestamp */}
			<div className="flex items-center justify-between">
				<h2 className="text-xl font-semibold">Dashboard</h2>
				{lastUpdated && (
					<span className="text-xs text-foreground-subtle">
						Updated {formatTimeAgo(lastUpdated)}
					</span>
				)}
			</div>

			{/* Compact stat cards */}
			<div className="grid grid-cols-2 md:grid-cols-4 gap-3">
				<CompactStatCard
					title="Ready"
					value={status.workable}
					icon={<CheckCircle2 className="size-4" />}
					colorClass="text-status-ready"
					href="/board?status=ready"
				/>
				<CompactStatCard
					title="Active"
					value={status.in_progress}
					icon={<Clock className="size-4" />}
					colorClass="text-status-in-progress"
					href="/board?status=in_progress"
				/>
				<CompactStatCard
					title="Blocked"
					value={status.blocked_deps + status.blocked_human}
					icon={<AlertTriangle className="size-4" />}
					colorClass="text-status-blocked"
					href="/board?status=blocked"
				/>
				<CompactStatCard
					title="Inbox"
					value={status.pending_inbox}
					icon={<Inbox className="size-4" />}
					colorClass="text-status-human"
					href="/inbox"
				/>
			</div>

			{/* Needs Attention section */}
			{hasAttentionItems && attentionData && (
				<section>
					<h3 className="text-sm font-medium text-foreground-muted mb-3 uppercase tracking-wide">
						Needs Attention
					</h3>
					<Card className="py-0 gap-0 overflow-hidden">
						<CardContent className="p-0">
							<ul className="divide-y divide-border">
								{/* Expiring claims - most urgent */}
								{attentionData.expiringClaims.map((claim) => (
									<AttentionItem
										key={`expiring-${claim.ticket_key}`}
										ticketKey={claim.ticket_key}
										icon={<Clock className="size-4 text-warning" />}
										type="expiring"
										description={`Claim expiring in ${claim.minutes_left}m`}
										meta={claim.worker_id}
									/>
								))}

								{/* Human-flagged tickets */}
								{attentionData.humanFlaggedTickets.map((ticket) => (
									<AttentionItem
										key={`human-${ticket.ticket_key}`}
										ticketKey={ticket.ticket_key}
										icon={<UserRound className="size-4 text-status-human" />}
										type="human"
										description={ticket.human_flag_reason || "Needs human decision"}
										meta={ticket.title}
										priority={ticket.priority}
									/>
								))}

								{/* Blocked tickets */}
								{attentionData.blockedTickets.map((ticket) => (
									<AttentionItem
										key={`blocked-${ticket.ticket_key}`}
										ticketKey={ticket.ticket_key}
										icon={<CircleDot className="size-4 text-status-blocked" />}
										type="blocked"
										description="Blocked by dependency"
										meta={ticket.title}
										priority={ticket.priority}
									/>
								))}
							</ul>
						</CardContent>
					</Card>
				</section>
			)}

			{/* Empty state for needs attention */}
			{!hasAttentionItems && (
				<section>
					<h3 className="text-sm font-medium text-foreground-muted mb-3 uppercase tracking-wide">
						Needs Attention
					</h3>
					<Card className="py-6">
						<CardContent className="flex flex-col items-center justify-center text-center">
							<CheckCircle2 className="size-8 text-status-ready mb-2" />
							<p className="text-sm text-foreground-muted">All clear</p>
							<p className="text-xs text-foreground-subtle">No items need immediate attention</p>
						</CardContent>
					</Card>
				</section>
			)}

			{/* Recent activity - streamlined */}
			{status.recent_activity.length > 0 && (
				<section>
					<h3 className="text-sm font-medium text-foreground-muted mb-3 uppercase tracking-wide">
						Recent Activity
					</h3>
					<Card className="py-0 gap-0 overflow-hidden">
						<CardContent className="p-0">
							<ul className="divide-y divide-border">
								{status.recent_activity.map((activity, i) => (
									<li
										key={`${activity.ticket_key}-${i}`}
										className="px-4 py-2.5 flex items-center gap-3 hover:bg-background-muted/50 transition-colors stagger-item"
									>
										<Link
											to={`/tickets/${activity.ticket_key}`}
											className="font-mono text-xs text-accent hover:underline shrink-0"
										>
											{activity.ticket_key}
										</Link>
										<ActionBadge action={activity.action} />
										{activity.summary && (
											<span className="text-xs text-foreground-muted truncate flex-1 min-w-0">
												{activity.summary}
											</span>
										)}
										<span className="text-xs text-foreground-subtle shrink-0">{activity.age}</span>
									</li>
								))}
							</ul>
						</CardContent>
					</Card>
				</section>
			)}
		</div>
	);
}

// =============================================================================
// Sub-components
// =============================================================================

interface CompactStatCardProps {
	title: string;
	value: number;
	icon: React.ReactNode;
	colorClass: string;
	href: string;
	trend?: { value: number; direction: "up" | "down" };
}

function CompactStatCard({ title, value, icon, colorClass, href, trend }: CompactStatCardProps) {
	return (
		<Link to={href} className="block bg-card border border-border rounded-lg p-3 card-hover">
			<div className="flex items-center justify-between mb-1">
				<span className={cn("shrink-0", colorClass)}>{icon}</span>
				{trend && (
					<span
						className={cn(
							"text-xs flex items-center gap-0.5",
							trend.direction === "up" ? "text-status-ready" : "text-error",
						)}
					>
						{trend.direction === "up" ? (
							<TrendingUp className="size-3" />
						) : (
							<TrendingDown className="size-3" />
						)}
						{trend.value}
					</span>
				)}
			</div>
			<p className="text-2xl font-bold leading-none mb-0.5">{value}</p>
			<p className="text-xs text-foreground-muted">{title}</p>
		</Link>
	);
}

interface AttentionItemProps {
	ticketKey: string;
	icon: React.ReactNode;
	type: "expiring" | "human" | "blocked";
	description: string;
	meta?: string;
	priority?: string;
}

function AttentionItem({ ticketKey, icon, type, description, meta, priority }: AttentionItemProps) {
	return (
		<li className="px-4 py-3 flex items-start gap-3 hover:bg-background-muted/50 transition-colors stagger-item">
			<span className="shrink-0 mt-0.5">{icon}</span>
			<div className="flex-1 min-w-0">
				<div className="flex items-center gap-2 mb-0.5">
					<Link
						to={`/tickets/${ticketKey}`}
						className="font-mono text-xs text-accent hover:underline"
					>
						{ticketKey}
					</Link>
					{priority && (
						<PriorityIndicator
							priority={priority as "highest" | "high" | "medium" | "low" | "lowest"}
							variant="dot"
						/>
					)}
					<span
						className={cn(
							"text-xs px-1.5 py-0.5 rounded",
							type === "expiring" && "bg-warning/10 text-warning",
							type === "human" && "bg-status-human/10 text-status-human",
							type === "blocked" && "bg-status-blocked/10 text-status-blocked",
						)}
					>
						{type === "expiring" ? "expiring" : type === "human" ? "human" : "blocked"}
					</span>
				</div>
				<p className="text-xs text-foreground-muted truncate">{description}</p>
				{meta && type !== "expiring" && (
					<p className="text-xs text-foreground-subtle truncate mt-0.5">{meta}</p>
				)}
			</div>
		</li>
	);
}

function ActionBadge({ action }: { action: string }) {
	const actionStyles: Record<string, string> = {
		completed: "bg-status-ready/10 text-status-ready",
		claimed: "bg-status-in-progress/10 text-status-in-progress",
		released: "bg-warning/10 text-warning",
		created: "bg-info/10 text-info",
		blocked: "bg-status-blocked/10 text-status-blocked",
		unblocked: "bg-status-ready/10 text-status-ready",
		review: "bg-status-review/10 text-status-review",
		accepted: "bg-status-ready/10 text-status-ready",
		rejected: "bg-error/10 text-error",
	};

	const style = actionStyles[action.toLowerCase()] || "bg-background-muted text-foreground-muted";

	return <span className={cn("text-xs px-1.5 py-0.5 rounded shrink-0", style)}>{action}</span>;
}

// =============================================================================
// Skeleton
// =============================================================================

function DashboardSkeleton() {
	return (
		<div className="space-y-6">
			{/* Header */}
			<div className="flex items-center justify-between">
				<Skeleton className="h-7 w-28" />
				<Skeleton className="h-4 w-24" />
			</div>

			{/* Stat cards */}
			<div className="grid grid-cols-2 md:grid-cols-4 gap-3">
				{["ready", "active", "blocked", "inbox"].map((name) => (
					<div key={name} className="bg-card border border-border rounded-lg p-3">
						<div className="flex items-center justify-between mb-1">
							<Skeleton className="size-4" />
						</div>
						<Skeleton className="h-8 w-12 mb-0.5" />
						<Skeleton className="h-3 w-16" />
					</div>
				))}
			</div>

			{/* Needs Attention section */}
			<section>
				<Skeleton className="h-4 w-32 mb-3" />
				<Card className="py-0 gap-0 overflow-hidden">
					<CardContent className="p-0">
						<ul className="divide-y divide-border">
							{["attention-1", "attention-2", "attention-3"].map((key) => (
								<li key={key} className="px-4 py-3 flex items-start gap-3">
									<Skeleton className="size-4 shrink-0 mt-0.5" />
									<div className="flex-1">
										<div className="flex items-center gap-2 mb-1">
											<Skeleton className="h-3 w-16" />
											<Skeleton className="h-4 w-14 rounded" />
										</div>
										<Skeleton className="h-3 w-48" />
									</div>
								</li>
							))}
						</ul>
					</CardContent>
				</Card>
			</section>

			{/* Recent Activity section */}
			<section>
				<Skeleton className="h-4 w-32 mb-3" />
				<Card className="py-0 gap-0 overflow-hidden">
					<CardContent className="p-0">
						<ul className="divide-y divide-border">
							{["activity-1", "activity-2", "activity-3", "activity-4", "activity-5"].map((key) => (
								<li key={key} className="px-4 py-2.5 flex items-center gap-3">
									<Skeleton className="h-3 w-16" />
									<Skeleton className="h-4 w-16 rounded" />
									<Skeleton className="h-3 w-48 flex-1" />
									<Skeleton className="h-3 w-8" />
								</li>
							))}
						</ul>
					</CardContent>
				</Card>
			</section>
		</div>
	);
}

// =============================================================================
// Helpers
// =============================================================================

function formatTimeAgo(date: Date): string {
	const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
	if (seconds < 60) return "just now";
	const minutes = Math.floor(seconds / 60);
	if (minutes < 60) return `${minutes}m ago`;
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `${hours}h ago`;
	return `${Math.floor(hours / 24)}d ago`;
}
