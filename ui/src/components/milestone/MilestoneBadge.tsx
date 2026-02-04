import { Flag } from "lucide-react";
import { Link } from "react-router-dom";
import { cn } from "@/lib/utils";

export interface MilestoneBadgeProps {
	milestoneKey: string;
	className?: string;
	/** Whether to render as a link to the milestone detail */
	link?: boolean;
}

/**
 * MilestoneBadge displays a compact milestone identifier badge.
 * Can optionally link to the milestone detail page.
 */
export function MilestoneBadge({ milestoneKey, className, link = true }: MilestoneBadgeProps) {
	const content = (
		<span
			className={cn(
				"inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium",
				"bg-blue-600/10 dark:bg-blue-400/10 text-blue-600 dark:text-blue-400",
				"border border-blue-600/20 dark:border-blue-400/20",
				link && "hover:bg-blue-600/20 dark:hover:bg-blue-400/20 transition-colors",
				className,
			)}
		>
			<Flag className="size-3" />
			<span>{milestoneKey}</span>
		</span>
	);

	if (link) {
		return (
			<Link to={`/milestones/${milestoneKey}`} onClick={(e) => e.stopPropagation()}>
				{content}
			</Link>
		);
	}

	return content;
}
