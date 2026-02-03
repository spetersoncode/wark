import { AlertTriangle, CircleMinus, GitBranch } from "lucide-react";
import { Link } from "react-router-dom";
import { PriorityIndicator, type Priority } from "@/components/PriorityIndicator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { Ticket } from "@/lib/api";

interface KanbanCardProps {
	ticket: Ticket;
	/** Show blocked badge for blocked tickets in Ready column */
	showBlockedBadge?: boolean;
}

/**
 * KanbanCard - Minimal ticket card for the board view.
 * Shows: key, title (2 lines max), priority dot
 * Human flag: prominent warning icon + reason
 * Branch name: hidden by default, shown on hover
 */
export function KanbanCard({ ticket, showBlockedBadge = false }: KanbanCardProps) {
	const hasHumanFlag = !!ticket.human_flag_reason;
	const hasBranch = !!ticket.branch_name;

	return (
		<Link
			to={`/tickets/${ticket.ticket_key}`}
			className={cn(
				"block p-3 bg-[var(--background)] border rounded-md transition-colors group",
				"border-[var(--border)] hover:border-[var(--border-strong)]",
				hasHumanFlag && "border-l-2 border-l-[var(--status-human)]",
			)}
		>
			{/* Top row: Key + Priority dot + optional blocked badge */}
			<div className="flex items-center justify-between gap-2 mb-1.5">
				<span className="font-mono text-xs text-[var(--foreground-muted)]">
					{ticket.ticket_key}
				</span>
				<div className="flex items-center gap-1.5">
					{showBlockedBadge && (
						<Tooltip>
							<TooltipTrigger asChild>
								<span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs bg-[var(--background-muted)] text-[var(--foreground-muted)]">
									<CircleMinus className="size-3" />
									<span>Blocked</span>
								</span>
							</TooltipTrigger>
							<TooltipContent>Blocked by dependencies</TooltipContent>
						</Tooltip>
					)}
					<PriorityIndicator priority={ticket.priority as Priority} variant="dot" />
				</div>
			</div>

			{/* Title - max 2 lines */}
			<p className="text-sm font-medium leading-snug line-clamp-2">{ticket.title}</p>

			{/* Human flag reason - prominent warning */}
			{hasHumanFlag && (
				<div className="flex items-start gap-1.5 mt-2 p-2 rounded bg-[var(--status-human)]/10 text-[var(--status-human)]">
					<AlertTriangle className="size-3.5 mt-0.5 shrink-0" />
					<p className="text-xs leading-snug line-clamp-2">{ticket.human_flag_reason}</p>
				</div>
			)}

			{/* Branch name - shown on hover */}
			{hasBranch && (
				<Tooltip>
					<TooltipTrigger asChild>
						<div className="flex items-center gap-1 mt-2 text-xs text-[var(--foreground-subtle)] opacity-0 group-hover:opacity-100 transition-opacity">
							<GitBranch className="size-3" />
							<span className="truncate font-mono">{ticket.branch_name}</span>
						</div>
					</TooltipTrigger>
					<TooltipContent side="bottom" className="font-mono text-xs">
						{ticket.branch_name}
					</TooltipContent>
				</Tooltip>
			)}
		</Link>
	);
}
