import { ChevronDown, ChevronRight, CircleCheck } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { PriorityIndicator, type Priority } from "@/components/PriorityIndicator";
import type { Ticket } from "@/lib/api";
import { cn } from "@/lib/utils";

interface ClosedColumnProps {
	tickets: Ticket[];
	/** Max items to show before collapsing */
	maxVisible?: number;
}

/**
 * ClosedColumn - Compact, collapsible list view for closed tickets.
 * Shows ticket key + priority dot in a minimal list format.
 */
export function ClosedColumn({ tickets, maxVisible = 8 }: ClosedColumnProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const hasMore = tickets.length > maxVisible;
	const visibleTickets = isExpanded ? tickets : tickets.slice(0, maxVisible);

	return (
		<div
			className={cn(
				"flex-shrink-0 w-48 bg-[var(--card)] border border-[var(--border)] rounded-lg",
				"border-l-2 border-l-[var(--status-closed)]",
			)}
		>
			{/* Column header */}
			<div className="p-3 border-b border-[var(--border)] flex items-center justify-between">
				<div className="flex items-center gap-2 text-[var(--foreground-muted)]">
					<CircleCheck className="size-4" />
					<span className="font-medium text-[var(--foreground)]">Closed</span>
				</div>
				<span className="text-sm text-[var(--foreground-muted)] tabular-nums">
					{tickets.length}
				</span>
			</div>

			{/* Compact list */}
			<div className="p-2 max-h-[calc(100vh-16rem)] overflow-y-auto">
				{tickets.length === 0 ? (
					<p className="text-sm text-[var(--foreground-subtle)] text-center py-6">(no tickets)</p>
				) : (
					<>
						<ul className="space-y-1">
							{visibleTickets.map((ticket) => (
								<li key={ticket.id}>
									<Link
										to={`/tickets/${ticket.ticket_key}`}
										className="flex items-center gap-2 px-2 py-1.5 rounded text-sm hover:bg-[var(--background-muted)] transition-colors"
									>
										<PriorityIndicator priority={ticket.priority as Priority} variant="dot" />
										<span className="font-mono text-xs text-[var(--foreground-muted)] truncate">
											{ticket.ticket_key}
										</span>
									</Link>
								</li>
							))}
						</ul>

						{/* Expand/Collapse or View All */}
						{hasMore && (
							<button
								type="button"
								onClick={() => setIsExpanded(!isExpanded)}
								className="flex items-center gap-1 w-full px-2 py-2 mt-1 text-xs text-[var(--accent)] hover:text-[var(--accent-hover)] hover:bg-[var(--background-muted)] rounded transition-colors"
							>
								{isExpanded ? (
									<>
										<ChevronDown className="size-3" />
										<span>Show less</span>
									</>
								) : (
									<>
										<ChevronRight className="size-3" />
										<span>+{tickets.length - maxVisible} more</span>
									</>
								)}
							</button>
						)}

						{/* Link to full list when expanded and many items */}
						{isExpanded && tickets.length > 20 && (
							<Link
								to="/tickets?status=closed"
								className="block text-center py-2 text-xs text-[var(--accent)] hover:underline"
							>
								View all in Tickets →
							</Link>
						)}
					</>
				)}
			</div>
		</div>
	);
}
