import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

export type Priority = "highest" | "high" | "medium" | "low" | "lowest";
export type PriorityVariant = "dot" | "text" | "full";

const priorityConfig: Record<
	Priority,
	{
		label: string;
		textClass: string;
		bgClass: string;
	}
> = {
	highest: {
		label: "Highest",
		textClass: "text-priority-highest",
		bgClass: "bg-priority-highest",
	},
	high: {
		label: "High",
		textClass: "text-priority-high",
		bgClass: "bg-priority-high",
	},
	medium: {
		label: "Medium",
		textClass: "text-priority-medium",
		bgClass: "bg-priority-medium",
	},
	low: {
		label: "Low",
		textClass: "text-priority-low",
		bgClass: "bg-priority-low",
	},
	lowest: {
		label: "Lowest",
		textClass: "text-priority-lowest",
		bgClass: "bg-priority-lowest",
	},
};

export interface PriorityIndicatorProps extends ComponentProps<"span"> {
	priority: Priority;
	variant?: PriorityVariant;
}

/**
 * PriorityIndicator displays a ticket/work item priority level.
 *
 * Variants:
 * - `dot`: Colored circle only
 * - `text`: Priority label only
 * - `full`: Colored dot + label (default)
 */
export function PriorityIndicator({
	priority,
	variant = "full",
	className,
	...props
}: PriorityIndicatorProps) {
	const config = priorityConfig[priority];

	if (variant === "dot") {
		return (
			<span
				className={cn("inline-block size-2.5 rounded-full", config.bgClass, className)}
				title={config.label}
				{...props}
			/>
		);
	}

	if (variant === "text") {
		return (
			<span className={cn("text-xs font-medium", config.textClass, className)} {...props}>
				{config.label}
			</span>
		);
	}

	// full variant: dot + text
	return (
		<span className={cn("inline-flex items-center gap-1.5", className)} {...props}>
			<span className={cn("inline-block size-2.5 rounded-full shrink-0", config.bgClass)} />
			<span className={cn("text-xs font-medium", config.textClass)}>{config.label}</span>
		</span>
	);
}
