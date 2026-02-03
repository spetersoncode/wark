import type { LucideIcon } from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/lib/utils";

export interface EmptyStateProps extends ComponentProps<"div"> {
	/** Optional Lucide icon to display above the title */
	icon?: LucideIcon;
	/** Main message to display */
	title: string;
	/** Optional secondary description text */
	description?: string;
	/** Optional action button or link */
	action?: ReactNode;
	/** Variant controls the visual style */
	variant?: "default" | "compact" | "card";
}

/**
 * EmptyState displays a zero-state message for empty lists, tables, or sections.
 *
 * Designed with personality - not just "no data" but helpful, friendly messages
 * that guide users or reassure them.
 *
 * @example
 * // Inbox empty state - positive/reassuring
 * <EmptyState
 *   icon={CheckCircle}
 *   title="All clear"
 *   description="No pending messages. Your agents are working independently."
 * />
 *
 * @example
 * // Compact variant for columns
 * <EmptyState title="No tickets" variant="compact" />
 *
 * @example
 * // With action
 * <EmptyState
 *   icon={Search}
 *   title="No results"
 *   description="Try adjusting your filters"
 *   action={<button onClick={clearFilters}>Clear filters</button>}
 * />
 */
export function EmptyState({
	icon: Icon,
	title,
	description,
	action,
	variant = "default",
	className,
	...props
}: EmptyStateProps) {
	if (variant === "compact") {
		return (
			<div className={cn("text-center py-6 animate-in fade-in duration-200", className)} {...props}>
				<p className="text-sm text-muted-foreground/70">{title}</p>
			</div>
		);
	}

	if (variant === "card") {
		return (
			<div
				className={cn(
					"bg-[var(--card)] border border-[var(--border)] rounded-lg",
					"flex flex-col items-center justify-center py-12 px-4 text-center",
					"animate-in fade-in duration-200",
					className,
				)}
				{...props}
			>
				{Icon && (
					<div className="w-14 h-14 rounded-full bg-[var(--background-muted)] flex items-center justify-center mb-4">
						<Icon className="size-7 text-muted-foreground/60" strokeWidth={1.5} />
					</div>
				)}
				<p className="text-base font-medium text-foreground mb-1">{title}</p>
				{description && <p className="text-sm text-muted-foreground max-w-xs">{description}</p>}
				{action && <div className="mt-4">{action}</div>}
			</div>
		);
	}

	// Default variant
	return (
		<div
			className={cn(
				"flex flex-col items-center justify-center py-8 text-center",
				"animate-in fade-in duration-200",
				className,
			)}
			{...props}
		>
			{Icon && (
				<div className="w-12 h-12 rounded-full bg-[var(--background-muted)] flex items-center justify-center mb-3">
					<Icon className="size-6 text-muted-foreground/60" strokeWidth={1.5} />
				</div>
			)}
			<p className="text-sm font-medium text-muted-foreground">{title}</p>
			{description && (
				<p className="text-xs text-muted-foreground/70 mt-1 max-w-[240px]">{description}</p>
			)}
			{action && <div className="mt-3">{action}</div>}
		</div>
	);
}
