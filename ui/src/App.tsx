import { AlertTriangle, CheckCircle2, CircleDot, Clock, Settings } from "lucide-react";
import { useEffect, useState } from "react";
import { ApiError, getStatus, type StatusResult } from "./lib/api";
import { cn } from "./lib/utils";

function App() {
	const [status, setStatus] = useState<StatusResult | null>(null);
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);

	useEffect(() => {
		async function fetchStatus() {
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
			}
		}

		fetchStatus();
		// Poll every 30 seconds
		const interval = setInterval(fetchStatus, 30000);
		return () => clearInterval(interval);
	}, []);

	return (
		<div className="min-h-screen bg-[var(--background)]">
			{/* Header */}
			<header className="border-b border-[var(--border)] bg-[var(--card)]">
				<div className="container mx-auto px-4 h-14 flex items-center justify-between">
					<h1 className="text-xl font-bold">wark</h1>
					<button
						type="button"
						className="p-2 rounded-md hover:bg-[var(--accent)] transition-colors"
						aria-label="Settings"
					>
						<Settings className="w-5 h-5" />
					</button>
				</div>
			</header>

			{/* Main content */}
			<main className="container mx-auto px-4 py-8">
				{loading && (
					<div className="flex items-center justify-center h-64">
						<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-[var(--primary)]" />
					</div>
				)}

				{error && (
					<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 text-red-700 dark:text-red-300">
						{error}
					</div>
				)}

				{status && !error && (
					<div className="space-y-8">
						{/* Status cards */}
						<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
							<StatusCard
								title="Workable"
								value={status.workable}
								icon={<CheckCircle2 className="w-5 h-5" />}
								color="text-green-600 dark:text-green-400"
							/>
							<StatusCard
								title="In Progress"
								value={status.in_progress}
								icon={<Clock className="w-5 h-5" />}
								color="text-blue-600 dark:text-blue-400"
							/>
							<StatusCard
								title="Blocked"
								value={status.blocked_deps + status.blocked_human}
								icon={<AlertTriangle className="w-5 h-5" />}
								color="text-orange-600 dark:text-orange-400"
							/>
							<StatusCard
								title="Pending Inbox"
								value={status.pending_inbox}
								icon={<CircleDot className="w-5 h-5" />}
								color="text-purple-600 dark:text-purple-400"
							/>
						</div>

						{/* Expiring soon */}
						{status.expiring_soon.length > 0 && (
							<section>
								<h2 className="text-lg font-semibold mb-4">Claims Expiring Soon</h2>
								<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
									<ul className="divide-y divide-[var(--border)]">
										{status.expiring_soon.map((claim) => (
											<li
												key={claim.ticket_key}
												className="px-4 py-3 flex items-center justify-between"
											>
												<span className="font-mono">{claim.ticket_key}</span>
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
								<h2 className="text-lg font-semibold mb-4">Recent Activity</h2>
								<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
									<ul className="divide-y divide-[var(--border)]">
										{status.recent_activity.map((activity, i) => (
											<li
												key={`${activity.ticket_key}-${i}`}
												className="px-4 py-3 flex items-center justify-between"
											>
												<div className="flex items-center gap-3">
													<span className="font-mono text-sm">{activity.ticket_key}</span>
													<span className="text-sm px-2 py-0.5 rounded bg-[var(--secondary)]">
														{activity.action}
													</span>
												</div>
												<span className="text-sm text-[var(--muted-foreground)]">
													{activity.age}
												</span>
											</li>
										))}
									</ul>
								</div>
							</section>
						)}
					</div>
				)}
			</main>
		</div>
	);
}

function StatusCard({
	title,
	value,
	icon,
	color,
}: {
	title: string;
	value: number;
	icon: React.ReactNode;
	color: string;
}) {
	return (
		<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
			<div className="flex items-center gap-3 mb-2">
				<span className={cn(color)}>{icon}</span>
				<span className="text-sm text-[var(--muted-foreground)]">{title}</span>
			</div>
			<p className="text-3xl font-bold">{value}</p>
		</div>
	);
}

export default App;
