import type { ComponentProps } from "react";
import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

export interface EmptyStateProps extends ComponentProps<"div"> {
	/** Optional Lucide icon to display above the title */
	icon?: LucideIcon;
	/** Main message to display */
	title: string;
	/** Optional secondary description text */
	description?: string;
}

/**
 * EmptyState displays a zero-state message for empty lists, tables, or sections.
 *
 * Visually understated but clear, with optional icon and description.
 *
 * @example
 * // Inbox empty state
 * <EmptyState
 *   icon={CheckCircle}
 *   title="All clear"
 *   description="No pending messages"
 * />
 *
 * @example
 * // Minimal board column empty state
 * <EmptyState title="(no tickets)" />
 *
 * @example
 * // Empty search results
 * <EmptyState
 *   icon={Search}
 *   title="No tickets found"
 *   description="Try adjusting your filters"
 * />
 */
export function EmptyState({
	icon: Icon,
	title,
	description,
	className,
	...props
}: EmptyStateProps) {
	return (
		<div
			className={cn("flex flex-col items-center justify-center py-8 text-center", className)}
			{...props}
		>
			{Icon && <Icon className="size-12 text-muted-foreground/50 mb-3" strokeWidth={1.5} />}
			<p className="text-sm font-medium text-muted-foreground">{title}</p>
			{description && (
				<p className="text-xs text-muted-foreground/70 mt-1 max-w-[200px]">{description}</p>
			)}
		</div>
	);
}
