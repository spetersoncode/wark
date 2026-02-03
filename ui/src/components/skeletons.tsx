import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

/**
 * Skeleton for dashboard stat cards (StatusCard in Dashboard)
 * Shows icon area + label on top, large number below
 */
export function StatCardSkeleton() {
	return (
		<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
			<div className="flex items-center gap-3 mb-2">
				<Skeleton className="h-5 w-5 rounded" />
				<Skeleton className="h-4 w-20" />
			</div>
			<Skeleton className="h-9 w-16" />
		</div>
	);
}

/**
 * Skeleton for the dashboard stats grid (4 cards)
 */
export function DashboardStatsSkeleton() {
	return (
		<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
			<StatCardSkeleton />
			<StatCardSkeleton />
			<StatCardSkeleton />
			<StatCardSkeleton />
		</div>
	);
}

/**
 * Skeleton for activity list items (Recent Activity in Dashboard, Activity in TicketDetail)
 * Shows avatar circle + text lines
 */
export function ActivityItemSkeleton() {
	return (
		<li className="px-4 py-3 flex items-center justify-between">
			<div className="flex items-center gap-3">
				<Skeleton className="h-4 w-16 font-mono" />
				<Skeleton className="h-5 w-20 rounded" />
				<Skeleton className="h-4 w-48" />
			</div>
			<Skeleton className="h-4 w-12" />
		</li>
	);
}

/**
 * Skeleton for activity timeline items with avatar (TicketDetail style)
 */
export function ActivityTimelineSkeleton() {
	return (
		<li className="flex items-start gap-3">
			<Skeleton className="w-8 h-8 rounded-full flex-shrink-0" />
			<div className="flex-1 min-w-0 space-y-1.5">
				<div className="flex items-center gap-2">
					<Skeleton className="h-4 w-24" />
					<Skeleton className="h-4 w-16" />
					<Skeleton className="h-4 w-12" />
				</div>
				<Skeleton className="h-3 w-3/4" />
			</div>
		</li>
	);
}

/**
 * Skeleton for expiring claims list items
 */
export function ExpiringClaimSkeleton() {
	return (
		<li className="px-4 py-3 flex items-center justify-between">
			<Skeleton className="h-4 w-20 font-mono" />
			<Skeleton className="h-4 w-36" />
		</li>
	);
}

/**
 * Full dashboard skeleton layout
 */
export function DashboardSkeleton() {
	return (
		<div className="space-y-8">
			{/* Header */}
			<div className="flex items-center justify-between">
				<Skeleton className="h-8 w-32" />
				<Skeleton className="h-9 w-24 rounded-md" />
			</div>

			{/* Stats grid */}
			<DashboardStatsSkeleton />

			{/* Expiring claims section */}
			<section>
				<Skeleton className="h-6 w-48 mb-4" />
				<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
					<ul className="divide-y divide-[var(--border)]">
						<ExpiringClaimSkeleton />
						<ExpiringClaimSkeleton />
					</ul>
				</div>
			</section>

			{/* Recent activity section */}
			<section>
				<Skeleton className="h-6 w-36 mb-4" />
				<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
					<ul className="divide-y divide-[var(--border)]">
						<ActivityItemSkeleton />
						<ActivityItemSkeleton />
						<ActivityItemSkeleton />
						<ActivityItemSkeleton />
						<ActivityItemSkeleton />
					</ul>
				</div>
			</section>
		</div>
	);
}

/**
 * Skeleton for a single ticket table row
 */
export function TicketRowSkeleton() {
	return (
		<tr>
			<td className="px-4 py-4">
				<Skeleton className="h-4 w-16 font-mono" />
			</td>
			<td className="px-4 py-4">
				<Skeleton className="h-4 w-64" />
			</td>
			<td className="px-4 py-4">
				<Skeleton className="h-4 w-20" />
			</td>
			<td className="px-4 py-4">
				<Skeleton className="h-5 w-24 rounded-md" />
			</td>
			<td className="px-4 py-4">
				<Skeleton className="h-4 w-16" />
			</td>
			<td className="px-4 py-4">
				<Skeleton className="h-4 w-14" />
			</td>
			<td className="px-4 py-4">
				<Skeleton className="h-4 w-16" />
			</td>
		</tr>
	);
}

/**
 * Full tickets list skeleton with table structure
 */
export function TicketsListSkeleton() {
	return (
		<div className="space-y-4">
			{/* Header */}
			<div className="flex items-center justify-between">
				<Skeleton className="h-8 w-24" />
				<Skeleton className="h-9 w-24 rounded-md" />
			</div>

			{/* Filter bar */}
			<div className="flex items-center gap-4 flex-wrap p-3 bg-[var(--card)] border border-[var(--border)] rounded-lg">
				<Skeleton className="h-4 w-16" />
				<Skeleton className="h-8 w-32 rounded-md" />
				<Skeleton className="h-8 w-32 rounded-md" />
				<Skeleton className="h-8 w-32 rounded-md" />
				<Skeleton className="h-8 w-32 rounded-md" />
			</div>

			{/* Table */}
			<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
				<div className="overflow-x-auto">
					<table className="w-full">
						<thead className="bg-[var(--secondary)] border-b border-[var(--border)]">
							<tr>
								<th className="px-4 py-3 text-left text-sm font-medium text-[var(--muted-foreground)] w-28">
									Key
								</th>
								<th className="px-4 py-3 text-left text-sm font-medium text-[var(--muted-foreground)]">
									Title
								</th>
								<th className="px-4 py-3 text-left text-sm font-medium text-[var(--muted-foreground)] w-32">
									Project
								</th>
								<th className="px-4 py-3 text-left text-sm font-medium text-[var(--muted-foreground)] w-32">
									Status
								</th>
								<th className="px-4 py-3 text-left text-sm font-medium text-[var(--muted-foreground)] w-28">
									Priority
								</th>
								<th className="px-4 py-3 text-left text-sm font-medium text-[var(--muted-foreground)] w-24">
									Complexity
								</th>
								<th className="px-4 py-3 text-left text-sm font-medium text-[var(--muted-foreground)] w-28">
									Created
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-[var(--border)]">
							<TicketRowSkeleton />
							<TicketRowSkeleton />
							<TicketRowSkeleton />
							<TicketRowSkeleton />
							<TicketRowSkeleton />
							<TicketRowSkeleton />
							<TicketRowSkeleton />
							<TicketRowSkeleton />
						</tbody>
					</table>
				</div>
			</div>
		</div>
	);
}

/**
 * Skeleton for a single kanban card
 */
export function KanbanCardSkeleton() {
	return (
		<div className="p-3 bg-[var(--background)] border border-[var(--border)] rounded-md">
			<div className="flex items-start justify-between gap-2 mb-2">
				<Skeleton className="h-3 w-14 font-mono" />
				<Skeleton className="h-4 w-12 rounded" />
			</div>
			<Skeleton className="h-4 w-full mb-1" />
			<Skeleton className="h-4 w-3/4" />
		</div>
	);
}

/**
 * Skeleton for a single kanban column - uses subtle left border
 */
export function KanbanColumnSkeleton({
	cardCount = 3,
	borderColor = "border-l-gray-500",
}: {
	cardCount?: number;
	borderColor?: string;
}) {
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
				<div className="flex items-center gap-2">
					<Skeleton className="h-4 w-4" />
					<Skeleton className="h-4 w-20" />
				</div>
				<Skeleton className="h-4 w-4" />
			</div>

			{/* Column content */}
			<div className="p-2 space-y-2">
				{Array.from({ length: cardCount }, (_, i) => `skeleton-card-${i}`).map((key) => (
					<KanbanCardSkeleton key={key} />
				))}
			</div>
		</div>
	);
}

/**
 * Skeleton for the compact closed column
 */
export function ClosedColumnSkeleton({ itemCount = 5 }: { itemCount?: number }) {
	return (
		<div
			className={cn(
				"flex-shrink-0 w-48 bg-[var(--card)] border border-[var(--border)] rounded-lg",
				"border-l-2 border-l-[var(--status-closed)]",
			)}
		>
			{/* Column header */}
			<div className="p-3 border-b border-[var(--border)] flex items-center justify-between">
				<div className="flex items-center gap-2">
					<Skeleton className="h-4 w-4" />
					<Skeleton className="h-4 w-16" />
				</div>
				<Skeleton className="h-4 w-4" />
			</div>

			{/* Compact list */}
			<div className="p-2 space-y-1">
				{Array.from({ length: itemCount }, (_, i) => `skeleton-closed-${i}`).map((key) => (
					<div key={key} className="flex items-center gap-2 px-2 py-1.5">
						<Skeleton className="h-2.5 w-2.5 rounded-full" />
						<Skeleton className="h-3 w-16 font-mono" />
					</div>
				))}
			</div>
		</div>
	);
}

/**
 * Full board skeleton with multiple columns
 * Matches the redesigned board: 4 main columns + compact closed column
 */
export function BoardSkeleton() {
	const columns = [
		{ borderColor: "border-l-[var(--status-ready)]", cards: 3 },
		{ borderColor: "border-l-[var(--status-in-progress)]", cards: 2 },
		{ borderColor: "border-l-[var(--status-human)]", cards: 1 },
		{ borderColor: "border-l-[var(--status-review)]", cards: 2 },
	];

	return (
		<div className="space-y-4">
			{/* Header */}
			<div className="flex items-center justify-between">
				<Skeleton className="h-8 w-20" />
				<Skeleton className="h-9 w-24 rounded-md" />
			</div>

			{/* Filter bar */}
			<div className="flex items-center gap-4 flex-wrap p-3 bg-[var(--card)] border border-[var(--border)] rounded-lg">
				<Skeleton className="h-4 w-16" />
				<Skeleton className="h-8 w-32 rounded-md" />
				<Skeleton className="h-8 w-32 rounded-md" />
				<Skeleton className="h-8 w-32 rounded-md" />
			</div>

			{/* Kanban columns */}
			<div className="flex gap-4 overflow-x-auto pb-4">
				{columns.map((col) => (
					<KanbanColumnSkeleton key={col.borderColor} borderColor={col.borderColor} cardCount={col.cards} />
				))}
				{/* Compact closed column */}
				<ClosedColumnSkeleton itemCount={6} />
			</div>
		</div>
	);
}

/**
 * Skeleton for an inbox message card
 */
export function InboxMessageSkeleton() {
	return (
		<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden border-l-4 border-l-purple-500">
			{/* Header */}
			<div className="p-4 border-b border-[var(--border)]">
				<div className="flex items-start justify-between gap-4">
					<div className="flex-1 min-w-0">
						<div className="flex items-center gap-2 mb-1">
							<Skeleton className="h-4 w-4" />
							<Skeleton className="h-4 w-16" />
						</div>
						<div className="flex items-center gap-2">
							<Skeleton className="h-4 w-20 font-mono" />
							<Skeleton className="h-4 w-4 rounded-full" />
							<Skeleton className="h-4 w-48" />
						</div>
					</div>
					<Skeleton className="h-4 w-16" />
				</div>
			</div>

			{/* Content */}
			<div className="p-4 space-y-2">
				<Skeleton className="h-3 w-24" />
				<Skeleton className="h-4 w-full" />
				<Skeleton className="h-4 w-full" />
				<Skeleton className="h-4 w-3/4" />
			</div>
		</div>
	);
}

/**
 * Full inbox skeleton layout
 */
export function InboxSkeleton() {
	return (
		<div className="space-y-4">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-4">
					<Skeleton className="h-8 w-16" />
					<Skeleton className="h-6 w-20 rounded-full" />
				</div>
				<Skeleton className="h-9 w-24 rounded-md" />
			</div>

			{/* Messages */}
			<div className="space-y-4">
				<InboxMessageSkeleton />
				<InboxMessageSkeleton />
				<InboxMessageSkeleton />
			</div>
		</div>
	);
}

/**
 * Skeleton for ticket detail page
 */
export function TicketDetailSkeleton() {
	return (
		<div className="space-y-6">
			{/* Header */}
			<div className="flex items-center gap-4">
				<Skeleton className="h-9 w-9 rounded-md" />
				<div className="flex-1">
					<div className="flex items-center gap-3 mb-1">
						<Skeleton className="h-4 w-20 font-mono" />
						<Skeleton className="h-5 w-20 rounded-md" />
						<Skeleton className="h-5 w-16 rounded-md" />
					</div>
					<Skeleton className="h-8 w-96" />
				</div>
				<Skeleton className="h-9 w-9 rounded-md" />
			</div>

			{/* Info grid */}
			<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
				{/* Main content */}
				<div className="lg:col-span-2 space-y-6">
					{/* Description */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<Skeleton className="h-6 w-28 mb-3" />
						<div className="space-y-2">
							<Skeleton className="h-4 w-full" />
							<Skeleton className="h-4 w-full" />
							<Skeleton className="h-4 w-full" />
							<Skeleton className="h-4 w-3/4" />
						</div>
					</section>

					{/* Activity history */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<Skeleton className="h-6 w-20 mb-3" />
						<ul className="space-y-3">
							<ActivityTimelineSkeleton />
							<ActivityTimelineSkeleton />
							<ActivityTimelineSkeleton />
							<ActivityTimelineSkeleton />
						</ul>
					</section>
				</div>

				{/* Sidebar */}
				<div className="space-y-6">
					{/* Details */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<Skeleton className="h-6 w-16 mb-3" />
						<div className="space-y-2">
							<div className="flex justify-between">
								<Skeleton className="h-4 w-20" />
								<Skeleton className="h-4 w-16" />
							</div>
							<div className="flex justify-between">
								<Skeleton className="h-4 w-14" />
								<Skeleton className="h-4 w-12" />
							</div>
						</div>
					</section>

					{/* Dependencies */}
					<section className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
						<Skeleton className="h-6 w-32 mb-3" />
						<div className="space-y-2">
							<Skeleton className="h-10 w-full rounded-md" />
							<Skeleton className="h-10 w-full rounded-md" />
						</div>
					</section>
				</div>
			</div>
		</div>
	);
}

/**
 * Skeleton for projects grid
 */
export function ProjectCardSkeleton() {
	return (
		<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
			<div className="flex items-start gap-3 mb-3">
				<Skeleton className="h-5 w-5 mt-0.5" />
				<div className="min-w-0 flex-1">
					<Skeleton className="h-4 w-12 mb-1" />
					<Skeleton className="h-5 w-32" />
				</div>
			</div>
			<Skeleton className="h-4 w-full mb-1" />
			<Skeleton className="h-4 w-3/4 mb-4" />
			<div className="flex items-center gap-4">
				<Skeleton className="h-4 w-16" />
				<Skeleton className="h-4 w-16" />
				<Skeleton className="h-4 w-16" />
			</div>
		</div>
	);
}

/**
 * Full projects page skeleton
 */
export function ProjectsSkeleton() {
	return (
		<div className="space-y-8">
			{/* Header */}
			<div className="flex items-center justify-between">
				<Skeleton className="h-8 w-24" />
				<Skeleton className="h-9 w-24 rounded-md" />
			</div>

			{/* Projects grid */}
			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
				<ProjectCardSkeleton />
				<ProjectCardSkeleton />
				<ProjectCardSkeleton />
				<ProjectCardSkeleton />
				<ProjectCardSkeleton />
				<ProjectCardSkeleton />
			</div>
		</div>
	);
}

/**
 * Skeleton for analytics metric cards
 */
export function MetricCardSkeleton() {
	return (
		<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-4">
			<Skeleton className="h-4 w-24 mb-1" />
			<Skeleton className="h-9 w-20 mb-1" />
			<Skeleton className="h-3 w-32" />
		</div>
	);
}

/**
 * Skeleton for analytics sections
 */
export function AnalyticsSectionSkeleton({ cardCount = 4 }: { cardCount?: number }) {
	return (
		<section>
			<div className="flex items-center gap-2 mb-4">
				<Skeleton className="h-5 w-5" />
				<Skeleton className="h-5 w-32" />
			</div>
			<div
				className={cn(
					"grid gap-4",
					cardCount === 3
						? "grid-cols-1 md:grid-cols-3"
						: "grid-cols-1 md:grid-cols-2 lg:grid-cols-4",
				)}
			>
				{Array.from({ length: cardCount }, (_, i) => `skeleton-metric-${i}`).map((key) => (
					<MetricCardSkeleton key={key} />
				))}
			</div>
		</section>
	);
}

/**
 * Full analytics page skeleton
 */
export function AnalyticsSkeleton() {
	return (
		<div className="space-y-8">
			{/* Header */}
			<div className="flex items-center justify-between">
				<Skeleton className="h-8 w-24" />
				<div className="flex items-center gap-4">
					<Skeleton className="h-9 w-36 rounded-md" />
					<Skeleton className="h-9 w-24 rounded-md" />
				</div>
			</div>

			{/* Success Metrics */}
			<AnalyticsSectionSkeleton cardCount={4} />

			{/* Human Interaction */}
			<AnalyticsSectionSkeleton cardCount={4} />

			{/* Throughput */}
			<AnalyticsSectionSkeleton cardCount={3} />

			{/* WIP Table */}
			<section>
				<div className="flex items-center gap-2 mb-4">
					<Skeleton className="h-5 w-5" />
					<Skeleton className="h-5 w-36" />
				</div>
				<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden">
					<div className="divide-y divide-[var(--border)]">
						{Array.from({ length: 4 }, (_, i) => `skeleton-wip-${i}`).map((key) => (
							<div key={key} className="px-4 py-3 flex justify-between">
								<Skeleton className="h-5 w-24" />
								<Skeleton className="h-5 w-8" />
							</div>
						))}
					</div>
				</div>
			</section>

			{/* Chart */}
			<section>
				<div className="flex items-center gap-2 mb-4">
					<Skeleton className="h-5 w-5" />
					<Skeleton className="h-5 w-56" />
				</div>
				<div className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden p-4">
					<Skeleton className="h-40 w-full" />
				</div>
			</section>
		</div>
	);
}
