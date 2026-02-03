import { AlertTriangle, FolderKanban } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { EmptyState } from "../components/EmptyState";
import { ProjectsSkeleton } from "../components/skeletons";
import { ApiError, listProjects, type ProjectWithStats } from "../lib/api";
import { useAutoRefresh } from "../lib/hooks";

export default function Projects() {
	const [projects, setProjects] = useState<ProjectWithStats[]>([]);
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);

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
		}
	}, []);

	// Initial fetch
	useEffect(() => {
		fetchProjects();
	}, [fetchProjects]);

	// Auto-refresh every 10 seconds when tab is visible
	useAutoRefresh(fetchProjects, [fetchProjects]);

	if (loading) {
		return <ProjectsSkeleton />;
	}

	if (error) {
		return (
			<div className="flex items-center gap-3 p-4 border border-error/20 bg-error/5 rounded-lg animate-in fade-in duration-200">
				<div className="w-10 h-10 rounded-full bg-error/10 flex items-center justify-center flex-shrink-0">
					<AlertTriangle className="w-5 h-5 text-error" />
				</div>
				<div className="flex-1">
					<p className="font-medium text-error">Failed to load projects</p>
					<p className="text-sm text-error/80">{error}</p>
				</div>
				<button
					type="button"
					onClick={() => fetchProjects()}
					className="px-3 py-1.5 text-sm rounded-md text-error hover:bg-error/10 transition-colors"
				>
					Retry
				</button>
			</div>
		);
	}

	return (
		<div className="space-y-8">
			{/* Header */}
			<h2 className="text-2xl font-bold">Projects</h2>

			{/* Projects list */}
			{projects.length === 0 ? (
				<EmptyState
					icon={FolderKanban}
					title="No projects yet"
					description="Projects organize your tickets. Create one using the CLI to get started."
					variant="card"
				/>
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
			to={`/tickets?project=${project.key}`}
			className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4 block card-hover stagger-item"
		>
			<div className="flex items-start gap-3 mb-3">
				<FolderKanban className="w-5 h-5 text-[var(--primary)] mt-0.5" />
				<div className="min-w-0 flex-1">
					<div className="flex items-center gap-2">
						<span className="font-mono text-sm text-[var(--muted-foreground)]">{project.key}</span>
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
