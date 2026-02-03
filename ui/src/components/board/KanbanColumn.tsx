import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface KanbanColumnProps {
	/** Column title */
	title: string;
	/** Number of items in column */
	count: number;
	/** Icon to show in header */
	icon: ReactNode;
	/** Border color class for the left tinted border */
	borderColor: string;
	/** Column content */
	children: ReactNode;
	/** Whether the column is empty */
	isEmpty?: boolean;
}

/**
 * KanbanColumn - Column container with header and scrollable content.
 * Uses a subtle tinted left border instead of saturated color stripes.
 */
export function KanbanColumn({
	title,
	count,
	icon,
	borderColor,
	children,
	isEmpty = false,
}: KanbanColumnProps) {
	return (
		<div
			className={cn(
				"flex-shrink-0 w-72 bg-[var(--card)] border border-[var(--border)] rounded-lg",
				"border-l-2",
				borderColor,
			)}
		>
			{/* Column header */}
			<div className="p-3 border-b border-[var(--border)] flex items-center justify-between">
				<div className="flex items-center gap-2 text-[var(--foreground-muted)]">
					{icon}
					<span className="font-medium text-[var(--foreground)]">{title}</span>
				</div>
				<span className="text-sm text-[var(--foreground-muted)] tabular-nums">{count}</span>
			</div>

			{/* Column content */}
			<div className="p-2 space-y-2 max-h-[calc(100vh-16rem)] overflow-y-auto">
				{isEmpty ? (
					<p className="text-sm text-[var(--foreground-subtle)] text-center py-6">(no tickets)</p>
				) : (
					children
				)}
			</div>
		</div>
	);
}
