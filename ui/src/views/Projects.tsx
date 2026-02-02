import { FolderKanban, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ApiError, listProjects, type ProjectWithStats } from "../lib/api";
import { cn } from "../lib/utils";

export default function Projects() {
	const [projects, setProjects] = useState<ProjectWithStats[]>([]);
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);
	const [refreshing, setRefreshing] = useState(false);

	const fetchProjects = useCallback(async () => {
		try {
			const data = await listProjects();
			setProjects(data);
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
		fetchProjects();
	}, [fetchProjects]);

	function handleRefresh() {
		setRefreshing(true);
		fetchProjects();
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
				<h2 className="text-2xl font-bold">Projects</h2>
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

			{/* Projects list */}
			{projects.length === 0 ? (
				<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-8 text-center">
					<FolderKanban className="w-12 h-12 mx-auto mb-4 text-[var(--muted-foreground)]" />
					<p className="text-[var(--muted-foreground)]">No projects found</p>
				</div>
			) : (
				<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
					{projects.map((project) => (
						<ProjectCard key={project.id} project={project} />
					))}
				</div>
			)}
		</div>
	);
}

function ProjectCard({ project }: { project: ProjectWithStats }) {
	const totalOpen =
		project.stats.blocked_count +
		project.stats.ready_count +
		project.stats.in_progress_count +
		project.stats.human_count +
		project.stats.review_count;

	return (
		<Link
			to={`/board?project=${project.key}`}
			className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4 hover:border-[var(--primary)] transition-colors block"
		>
			<div className="flex items-start gap-3 mb-3">
				<FolderKanban className="w-5 h-5 text-[var(--primary)] mt-0.5" />
				<div className="min-w-0 flex-1">
					<div className="flex items-center gap-2">
						<span className="font-mono text-sm text-[var(--muted-foreground)]">
							{project.key}
						</span>
					</div>
					<h3 className="font-semibold truncate">{project.name}</h3>
				</div>
			</div>

			{project.description && (
				<p className="text-sm text-[var(--muted-foreground)] mb-4 line-clamp-2">
					{project.description}
				</p>
			)}

			{/* Stats row */}
			<div className="flex items-center gap-4 text-sm">
				<div className="flex items-center gap-1.5">
					<span className="text-[var(--muted-foreground)]">Open:</span>
					<span className="font-medium">{totalOpen}</span>
				</div>
				<div className="flex items-center gap-1.5">
					<span className="text-green-600 dark:text-green-400">Ready:</span>
					<span className="font-medium">{project.stats.ready_count}</span>
				</div>
				<div className="flex items-center gap-1.5">
					<span className="text-blue-600 dark:text-blue-400">Active:</span>
					<span className="font-medium">{project.stats.in_progress_count}</span>
				</div>
			</div>
		</Link>
	);
}
