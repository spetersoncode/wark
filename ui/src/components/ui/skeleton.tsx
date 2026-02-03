import { cn } from "@/lib/utils";

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
	return (
		<div
			data-slot="skeleton"
			className={cn("rounded-md", "bg-[var(--background-muted)] skeleton-shimmer", className)}
			{...props}
		/>
	);
}

export { Skeleton };
