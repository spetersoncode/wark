import { AlertTriangle, Flag } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { EmptyState } from "../components/EmptyState";
import { MilestoneCard } from "../components/milestone";
import { MilestonesSkeleton } from "../components/skeletons";
import { ApiError, listMilestones, type MilestoneWithStats } from "../lib/api";
import { useAutoRefresh } from "../lib/hooks";

export default function Milestones() {
	const [searchParams] = useSearchParams();
	const projectFilter = searchParams.get("project") || undefined;

	const [milestones, setMilestones] = useState<MilestoneWithStats[]>([]);
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);

	const fetchMilestones = useCallback(async () => {
		try {
			const data = await listMilestones(projectFilter);
			setMilestones(data);
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
	}, [projectFilter]);

	// Initial fetch
	useEffect(() => {
		fetchMilestones();
	}, [fetchMilestones]);

	// Auto-refresh every 10 seconds when tab is visible
	useAutoRefresh(fetchMilestones, [fetchMilestones]);

	if (loading) {
		return <MilestonesSkeleton />;
	}

	if (error) {
		return (
			<div className="flex items-center gap-3 p-4 border border-error/20 bg-error/5 rounded-lg animate-in fade-in duration-200">
				<div className="w-10 h-10 rounded-full bg-error/10 flex items-center justify-center flex-shrink-0">
					<AlertTriangle className="w-5 h-5 text-error" />
				</div>
				<div className="flex-1">
					<p className="font-medium text-error">Failed to load milestones</p>
					<p className="text-sm text-error/80">{error}</p>
				</div>
				<button
					type="button"
					onClick={() => fetchMilestones()}
					className="px-3 py-1.5 text-sm rounded-md text-error hover:bg-error/10 transition-colors"
				>
					Retry
				</button>
			</div>
		);
	}

	// Group milestones by status
	const openMilestones = milestones.filter((m) => m.status === "open");
	const achievedMilestones = milestones.filter((m) => m.status === "achieved");
	const abandonedMilestones = milestones.filter((m) => m.status === "abandoned");

	return (
		<div className="space-y-8">
			{/* Header */}
			<div className="flex items-center justify-between">
				<h2 className="text-2xl font-bold">
					Milestones
					{projectFilter && (
						<span className="text-[var(--muted-foreground)] font-normal ml-2">
							({projectFilter})
						</span>
					)}
				</h2>
			</div>

			{/* Milestones list */}
			{milestones.length === 0 ? (
				<EmptyState
					icon={Flag}
					title="No milestones yet"
					description="Milestones help organize work toward specific goals. Create one using the CLI to get started."
					variant="card"
				/>
			) : (
				<div className="space-y-8">
					{/* Open milestones */}
					{openMilestones.length > 0 && (
						<section>
							<h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
								<span className="w-2 h-2 rounded-full bg-blue-500" />
								Open ({openMilestones.length})
							</h3>
							<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
								{openMilestones.map((milestone) => (
									<MilestoneCard key={milestone.id} milestone={milestone} />
								))}
							</div>
						</section>
					)}

					{/* Achieved milestones */}
					{achievedMilestones.length > 0 && (
						<section>
							<h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
								<span className="w-2 h-2 rounded-full bg-green-500" />
								Achieved ({achievedMilestones.length})
							</h3>
							<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
								{achievedMilestones.map((milestone) => (
									<MilestoneCard key={milestone.id} milestone={milestone} />
								))}
							</div>
						</section>
					)}

					{/* Abandoned milestones */}
					{abandonedMilestones.length > 0 && (
						<section>
							<h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
								<span className="w-2 h-2 rounded-full bg-gray-400" />
								Abandoned ({abandonedMilestones.length})
							</h3>
							<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
								{abandonedMilestones.map((milestone) => (
									<MilestoneCard key={milestone.id} milestone={milestone} />
								))}
							</div>
						</section>
					)}
				</div>
			)}
		</div>
	);
}
