import { cn } from "@/lib/utils";

export interface MilestoneProgressProps {
	/** Completion percentage (0-100) */
	percentage: number;
	/** Completed ticket count */
	completed: number;
	/** Total ticket count */
	total: number;
	/** Show label text */
	showLabel?: boolean;
	/** Additional class names */
	className?: string;
	/** Size variant */
	size?: "sm" | "md" | "lg";
}

/**
 * MilestoneProgress displays a progress bar with ticket completion stats.
 */
export function MilestoneProgress({
	percentage,
	completed,
	total,
	showLabel = true,
	className,
	size = "md",
}: MilestoneProgressProps) {
	const sizeClasses = {
		sm: "h-1.5",
		md: "h-2",
		lg: "h-3",
	};

	// Determine color based on completion
	const getProgressColor = () => {
		if (percentage >= 100) return "bg-green-500";
		if (percentage >= 75) return "bg-blue-500";
		if (percentage >= 50) return "bg-yellow-500";
		if (percentage >= 25) return "bg-orange-500";
		return "bg-gray-400";
	};

	return (
		<div className={cn("space-y-1", className)}>
			{showLabel && (
				<div className="flex items-center justify-between text-sm">
					<span className="text-[var(--foreground-muted)]">Progress</span>
					<span className="font-medium">
						{completed}/{total} ({Math.round(percentage)}%)
					</span>
				</div>
			)}
			<div
				className={cn("w-full bg-[var(--border)] rounded-full overflow-hidden", sizeClasses[size])}
			>
				<div
					className={cn("h-full rounded-full transition-all duration-300", getProgressColor())}
					style={{ width: `${Math.min(percentage, 100)}%` }}
				/>
			</div>
		</div>
	);
}
