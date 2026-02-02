import { AlertTriangle, CheckCircle2, CircleDot, Clock, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ApiError, getStatus, type StatusResult } from "../lib/api";
import { cn } from "../lib/utils";

export default function Dashboard() {
	const [status, setStatus] = useState<StatusResult | null>(null);
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);
	const [refreshing, setRefreshing] = useState(false);

	const fetchStatus = useCallback(async () => {
		try {
			const data = await getStatus();
			setStatus(data);
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
	}, []);

	useEffect(() => {
		fetchStatus();
		const interval = setInterval(fetchStatus, 30000);
		return () => clearInterval(interval);
	}, [fetchStatus]);

	function handleRefresh() {
		setRefreshing(true);
		fetchStatus();
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

	if (!status) return null;

	return (
		<div className="space-y-8">
			{/* Header with refresh */}
			<div className="flex items-center justify-between">
				<h2 className="text-2xl font-bold">Dashboard</h2>
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

			{/* Status cards */}
			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
				<StatusCard
					title="Workable"
					value={status.workable}
					icon={<CheckCircle2 className="w-5 h-5" />}
					color="text-green-600 dark:text-green-400"
					href="/board?status=ready"
				/>
				<StatusCard
					title="In Progress"
					value={status.in_progress}
					icon={<Clock className="w-5 h-5" />}
					color="text-blue-600 dark:text-blue-400"
					href="/board?status=in_progress"
				/>
				<StatusCard
					title="Blocked"
					value={status.blocked_deps + status.blocked_human}
					icon={<AlertTriangle className="w-5 h-5" />}
					color="text-orange-600 dark:text-orange-400"
					href="/board?status=blocked"
				/>
				<StatusCard
					title="Pending Inbox"
					value={status.pending_inbox}
					icon={<CircleDot className="w-5 h-5" />}
					color="text-purple-600 dark:text-purple-400"
					href="/inbox?pending=true"
				/>
			</div>

			{/* Expiring soon */}
			{status.expiring_soon.length > 0 && (
				<section>
					<h3 className="text-lg font-semibold mb-4">Claims Expiring Soon</h3>
					<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
						<ul className="divide-y divide-[var(--border)]">
							{status.expiring_soon.map((claim) => (
								<li key={claim.ticket_key} className="px-4 py-3 flex items-center justify-between">
									<Link
										to={`/tickets/${claim.ticket_key}`}
										className="font-mono hover:text-[var(--primary)] transition-colors"
									>
										{claim.ticket_key}
									</Link>
									<span className="text-sm text-[var(--muted-foreground)]">
										{claim.minutes_left}m remaining • {claim.worker_id}
									</span>
								</li>
							))}
						</ul>
					</div>
				</section>
			)}

			{/* Recent activity */}
			{status.recent_activity.length > 0 && (
				<section>
					<h3 className="text-lg font-semibold mb-4">Recent Activity</h3>
					<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
						<ul className="divide-y divide-[var(--border)]">
							{status.recent_activity.map((activity, i) => (
								<li
									key={`${activity.ticket_key}-${i}`}
									className="px-4 py-3 flex items-center justify-between"
								>
									<div className="flex items-center gap-3">
										<Link
											to={`/tickets/${activity.ticket_key}`}
											className="font-mono text-sm hover:text-[var(--primary)] transition-colors"
										>
											{activity.ticket_key}
										</Link>
										<span className="text-sm px-2 py-0.5 rounded bg-[var(--secondary)]">
											{activity.action}
										</span>
										{activity.summary && (
											<span className="text-sm text-[var(--muted-foreground)] truncate max-w-md">
												{activity.summary}
											</span>
										)}
									</div>
									<span className="text-sm text-[var(--muted-foreground)]">{activity.age}</span>
								</li>
							))}
						</ul>
					</div>
				</section>
			)}
		</div>
	);
}

function StatusCard({
	title,
	value,
	icon,
	color,
	href,
}: {
	title: string;
	value: number;
	icon: React.ReactNode;
	color: string;
	href: string;
}) {
	return (
		<Link
			to={href}
			className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4 hover:border-[var(--primary)] transition-colors"
		>
			<div className="flex items-center gap-3 mb-2">
				<span className={cn(color)}>{icon}</span>
				<span className="text-sm text-[var(--muted-foreground)]">{title}</span>
			</div>
			<p className="text-3xl font-bold">{value}</p>
		</Link>
	);
}
