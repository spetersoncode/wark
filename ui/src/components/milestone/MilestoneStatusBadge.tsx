import { CheckCircle2, Circle, XCircle } from "lucide-react";
import type { ComponentProps } from "react";
import { Badge } from "@/components/ui/badge";
import type { MilestoneStatus } from "@/lib/api";
import { cn } from "@/lib/utils";

const statusConfig: Record<
	MilestoneStatus,
	{
		icon: typeof Circle;
		label: string;
		textClass: string;
		bgClass: string;
		borderClass: string;
	}
> = {
	open: {
		icon: Circle,
		label: "Open",
		textClass: "text-blue-600 dark:text-blue-400",
		bgClass: "bg-blue-600/10 dark:bg-blue-400/10",
		borderClass: "border-blue-600 dark:border-blue-400",
	},
	achieved: {
		icon: CheckCircle2,
		label: "Achieved",
		textClass: "text-green-600 dark:text-green-400",
		bgClass: "bg-green-600/10 dark:bg-green-400/10",
		borderClass: "border-green-600 dark:border-green-400",
	},
	abandoned: {
		icon: XCircle,
		label: "Abandoned",
		textClass: "text-gray-500 dark:text-gray-400",
		bgClass: "bg-gray-500/10 dark:bg-gray-400/10",
		borderClass: "border-gray-500 dark:border-gray-400",
	},
};

export interface MilestoneStatusBadgeProps extends Omit<ComponentProps<typeof Badge>, "variant"> {
	status: MilestoneStatus;
}

/**
 * MilestoneStatusBadge displays a milestone status with an icon and label.
 */
export function MilestoneStatusBadge({ status, className, ...props }: MilestoneStatusBadgeProps) {
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
