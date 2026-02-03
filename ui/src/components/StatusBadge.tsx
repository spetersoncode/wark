import { CircleCheck, CircleDot, CircleMinus, CircleX, Eye, UserRound } from "lucide-react";
import type { ComponentProps } from "react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export type Status = "ready" | "in_progress" | "human" | "review" | "blocked" | "closed";

const statusConfig: Record<
	Status,
	{
		icon: typeof CircleCheck;
		label: string;
		textClass: string;
		bgClass: string;
		borderClass: string;
	}
> = {
	ready: {
		icon: CircleCheck,
		label: "Ready",
		textClass: "text-status-ready",
		bgClass: "bg-status-ready/10",
		borderClass: "border-status-ready",
	},
	in_progress: {
		icon: CircleDot,
		label: "In Progress",
		textClass: "text-status-in-progress",
		bgClass: "bg-status-in-progress/10",
		borderClass: "border-status-in-progress",
	},
	human: {
		icon: UserRound,
		label: "Human",
		textClass: "text-status-human",
		bgClass: "bg-status-human/10",
		borderClass: "border-status-human",
	},
	review: {
		icon: Eye,
		label: "Review",
		textClass: "text-status-review",
		bgClass: "bg-status-review/10",
		borderClass: "border-status-review",
	},
	blocked: {
		icon: CircleMinus,
		label: "Blocked",
		textClass: "text-status-blocked",
		bgClass: "bg-status-blocked/10",
		borderClass: "border-status-blocked",
	},
	closed: {
		icon: CircleX,
		label: "Closed",
		textClass: "text-status-closed",
		bgClass: "bg-status-closed/10",
		borderClass: "border-status-closed",
	},
};

export interface StatusBadgeProps extends Omit<ComponentProps<typeof Badge>, "variant"> {
	status: Status;
}

/**
 * StatusBadge displays a ticket/work item status with an icon and label.
 * Uses the design system status colors for visual consistency.
 */
export function StatusBadge({ status, className, ...props }: StatusBadgeProps) {
	const config = statusConfig[status];
	const Icon = config.icon;

	return (
		<Badge
			variant="outline"
			className={cn(config.textClass, config.bgClass, config.borderClass, className)}
			{...props}
		>
			<Icon className="size-3" />
			{config.label}
		</Badge>
	);
}
