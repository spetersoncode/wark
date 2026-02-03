import { CheckCircle, Search } from "lucide-react";
import { EmptyState } from "@/components/EmptyState";
import { type Priority, PriorityIndicator } from "@/components/PriorityIndicator";
import { type Status, StatusBadge } from "@/components/StatusBadge";

const statuses: Status[] = ["ready", "in_progress", "human", "review", "blocked", "closed"];
const priorities: Priority[] = ["highest", "high", "medium", "low", "lowest"];

export function ComponentDemo() {
	return (
		<div className="space-y-8 max-w-2xl">
			<section className="space-y-4">
				<h2 className="text-lg font-semibold">StatusBadge</h2>
				<div className="flex flex-wrap gap-2">
					{statuses.map((status) => (
						<StatusBadge key={status} status={status} />
					))}
				</div>
			</section>

			<section className="space-y-4">
				<h2 className="text-lg font-semibold">PriorityIndicator</h2>

				<div className="space-y-3">
					<div>
						<h3 className="text-sm text-[var(--foreground-muted)] mb-2">Full variant (default)</h3>
						<div className="flex flex-wrap gap-4">
							{priorities.map((priority) => (
								<PriorityIndicator key={priority} priority={priority} variant="full" />
							))}
						</div>
					</div>

					<div>
						<h3 className="text-sm text-[var(--foreground-muted)] mb-2">Dot variant</h3>
						<div className="flex flex-wrap gap-4 items-center">
							{priorities.map((priority) => (
								<PriorityIndicator key={priority} priority={priority} variant="dot" />
							))}
						</div>
					</div>

					<div>
						<h3 className="text-sm text-[var(--foreground-muted)] mb-2">Text variant</h3>
						<div className="flex flex-wrap gap-4">
							{priorities.map((priority) => (
								<PriorityIndicator key={priority} priority={priority} variant="text" />
							))}
						</div>
					</div>
				</div>
			</section>

			<section className="space-y-4">
				<h2 className="text-lg font-semibold">EmptyState</h2>
				<div className="grid grid-cols-3 gap-4">
					<div className="border border-[var(--border)] rounded-md">
						<EmptyState icon={CheckCircle} title="All clear" description="No pending messages" />
					</div>
					<div className="border border-[var(--border)] rounded-md">
						<EmptyState title="(no tickets)" className="py-4" />
					</div>
					<div className="border border-[var(--border)] rounded-md">
						<EmptyState
							icon={Search}
							title="No tickets found"
							description="Try adjusting your filters"
						/>
					</div>
				</div>
			</section>

			<section className="space-y-4">
				<h2 className="text-lg font-semibold">Combined Example</h2>
				<div className="border border-[var(--border)] rounded-md p-4 space-y-3">
					<div className="flex items-center justify-between">
						<span className="font-medium">WARK-123: Fix authentication bug</span>
						<div className="flex items-center gap-3">
							<PriorityIndicator priority="high" variant="dot" />
							<StatusBadge status="in_progress" />
						</div>
					</div>
					<div className="flex items-center justify-between">
						<span className="font-medium">WARK-124: Add dark mode support</span>
						<div className="flex items-center gap-3">
							<PriorityIndicator priority="medium" variant="dot" />
							<StatusBadge status="review" />
						</div>
					</div>
					<div className="flex items-center justify-between">
						<span className="font-medium">WARK-125: Database migration</span>
						<div className="flex items-center gap-3">
							<PriorityIndicator priority="highest" variant="dot" />
							<StatusBadge status="blocked" />
						</div>
					</div>
				</div>
			</section>
		</div>
	);
}
