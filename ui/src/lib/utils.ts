import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * Utility function to merge Tailwind CSS classes with conflict resolution.
 * Used by shadcn/ui components.
 */
export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

/**
 * Format a relative time string (e.g., "5 minutes ago")
 */
export function formatRelativeTime(date: Date | string): string {
	const d = typeof date === "string" ? new Date(date) : date;
	const now = new Date();
	const diffMs = now.getTime() - d.getTime();
	const diffMins = Math.floor(diffMs / 60000);
	const diffHours = Math.floor(diffMs / 3600000);
	const diffDays = Math.floor(diffMs / 86400000);

	if (diffMins < 1) return "just now";
	if (diffMins < 60) return `${diffMins}m ago`;
	if (diffHours < 24) return `${diffHours}h ago`;
	return `${diffDays}d ago`;
}

/**
 * Format a duration in minutes to a human-readable string
 */
export function formatDuration(minutes: number): string {
	if (minutes < 60) return `${minutes}m`;
	const hours = Math.floor(minutes / 60);
	const mins = minutes % 60;
	return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
}

/**
 * Get a color class for ticket status
 */
export function getStatusColor(status: string): string {
	switch (status) {
		case "ready":
			return "text-green-600 dark:text-green-400";
		case "in_progress":
			return "text-blue-600 dark:text-blue-400";
		case "blocked":
			return "text-orange-600 dark:text-orange-400";
		case "human":
			return "text-purple-600 dark:text-purple-400";
		case "review":
			return "text-yellow-600 dark:text-yellow-400";
		case "closed":
			return "text-gray-500 dark:text-gray-400";
		default:
			return "text-gray-600 dark:text-gray-400";
	}
}

/**
 * Get a color class for ticket priority
 */
export function getPriorityColor(priority: string): string {
	switch (priority) {
		case "highest":
			return "text-red-600 dark:text-red-400";
		case "high":
			return "text-orange-600 dark:text-orange-400";
		case "medium":
			return "text-yellow-600 dark:text-yellow-400";
		case "low":
			return "text-blue-600 dark:text-blue-400";
		case "lowest":
			return "text-gray-500 dark:text-gray-400";
		default:
			return "text-gray-600 dark:text-gray-400";
	}
}
