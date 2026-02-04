import { Calendar, Flag } from "lucide-react";
import { Link } from "react-router-dom";
import type { MilestoneWithStats } from "@/lib/api";
import { MilestoneProgress } from "./MilestoneProgress";
import { MilestoneStatusBadge } from "./MilestoneStatusBadge";

export interface MilestoneCardProps {
	milestone: MilestoneWithStats;
}

/**
 * MilestoneCard displays a milestone in a card format for list views.
 */
export function MilestoneCard({ milestone }: MilestoneCardProps) {
	const milestoneKey = `${milestone.project_key}/${milestone.key}`;

	// Format target date if present
	const formattedDate = milestone.target_date
		? new Date(milestone.target_date).toLocaleDateString(undefined, {
				month: "short",
				day: "numeric",
				year: "numeric",
			})
		: null;

	// Check if overdue (open milestone with past target date)
	const isOverdue =
		milestone.status === "open" &&
		milestone.target_date &&
		new Date(milestone.target_date) < new Date();

	return (
		<Link
			to={`/milestones/${milestoneKey}`}
			className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4 block card-hover stagger-item"
		>
			<div className="flex items-start gap-3 mb-3">
				<Flag className="w-5 h-5 text-[var(--primary)] mt-0.5 flex-shrink-0" />
				<div className="min-w-0 flex-1">
					<div className="flex items-center gap-2 mb-1">
						<span className="font-mono text-sm text-[var(--muted-foreground)]">
							{milestoneKey}
						</span>
						<MilestoneStatusBadge status={milestone.status} className="scale-90" />
					</div>
					<h3 className="font-semibold truncate">{milestone.name}</h3>
				</div>
			</div>

			{/* Goal excerpt if present */}
			{milestone.goal && (
				<p className="text-sm text-[var(--muted-foreground)] mb-3 line-clamp-2">
					{milestone.goal}
				</p>
			)}

			{/* Progress bar */}
			<MilestoneProgress
				percentage={milestone.completion_pct}
				completed={milestone.completed_count}
				total={milestone.ticket_count}
				size="sm"
				className="mb-3"
			/>

			{/* Stats row */}
			<div className="flex items-center gap-4 text-sm">
				<div className="flex items-center gap-1.5">
					<span className="text-[var(--muted-foreground)]">Tickets:</span>
					<span className="font-medium">{milestone.ticket_count}</span>
				</div>
				{formattedDate && (
					<div className="flex items-center gap-1.5">
						<Calendar className={`w-3.5 h-3.5 ${isOverdue ? "text-red-500" : "text-[var(--muted-foreground)]"}`} />
						<span className={isOverdue ? "text-red-500 font-medium" : "text-[var(--muted-foreground)]"}>
							{formattedDate}
						</span>
					</div>
				)}
			</div>
		</Link>
	);
}
