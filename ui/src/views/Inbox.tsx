import {
	AlertTriangle,
	CheckCircle,
	FileSearch,
	HelpCircle,
	Info,
	RefreshCw,
	Scale,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { EmptyState } from "../components/EmptyState";
import { useRefreshShortcut } from "../components/KeyboardShortcutsProvider";
import { Markdown } from "../components/Markdown";
import { InboxSkeleton } from "../components/skeletons";
import { type InboxMessage, listInbox, type MessageType } from "../lib/api";
import { useAutoRefresh } from "../lib/hooks";
import { cn, formatRelativeTime } from "../lib/utils";

/** Priority order for message types (lower = more urgent) */
const MESSAGE_TYPE_PRIORITY: Record<MessageType, number> = {
	escalation: 0,
	question: 1,
	decision: 2,
	review: 3,
	info: 4,
};

const MESSAGE_TYPE_CONFIG: Record<
	MessageType,
	{ label: string; icon: React.ReactNode; color: string }
> = {
	escalation: {
		label: "Escalation",
		icon: <AlertTriangle className="w-4 h-4" />,
		color: "text-red-600 dark:text-red-400",
	},
	question: {
		label: "Question",
		icon: <HelpCircle className="w-4 h-4" />,
		color: "text-blue-600 dark:text-blue-400",
	},
	decision: {
		label: "Decision",
		icon: <Scale className="w-4 h-4" />,
		color: "text-amber-600 dark:text-amber-400",
	},
	review: {
		label: "Review",
		icon: <FileSearch className="w-4 h-4" />,
		color: "text-purple-600 dark:text-purple-400",
	},
	info: {
		label: "Info",
		icon: <Info className="w-4 h-4" />,
		color: "text-gray-600 dark:text-gray-400",
	},
};

/**
 * Sort messages by:
 * 1. Pending (not responded) first
 * 2. Within pending: by message type priority (escalation > question > decision > review > info)
 * 3. Within same type: by created_at (newest first)
 */
function sortMessages(messages: InboxMessage[]): InboxMessage[] {
	return [...messages].sort((a, b) => {
		// Pending messages first
		const aPending = !a.responded_at;
		const bPending = !b.responded_at;
		if (aPending !== bPending) return aPending ? -1 : 1;

		// Sort by message type priority
		const aPriority = MESSAGE_TYPE_PRIORITY[a.message_type];
		const bPriority = MESSAGE_TYPE_PRIORITY[b.message_type];
		if (aPriority !== bPriority) return aPriority - bPriority;

		// Sort by created_at (newest first)
		return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
	});
}

export default function Inbox() {
	const [messages, setMessages] = useState<InboxMessage[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const fetchMessages = useCallback(async () => {
		try {
			// Always show only pending messages
			const data = await listInbox({ pending: true });
			setMessages(sortMessages(data));
			setError(null);
		} catch (e) {
			setError(e instanceof Error ? e.message : "Failed to fetch inbox");
		} finally {
			setLoading(false);
		}
	}, []);

	// Initial fetch
	useEffect(() => {
		fetchMessages();
	}, [fetchMessages]);

	// Auto-refresh every 10 seconds when tab is visible
	const { refreshing, refresh: handleRefresh } = useAutoRefresh(fetchMessages, [fetchMessages]);

	// Register "r" keyboard shortcut for refresh
	useRefreshShortcut(handleRefresh);

	if (loading) {
		return <InboxSkeleton />;
	}

	if (error) {
		return (
			<div className="flex items-center gap-3 p-4 border border-error/20 bg-error/5 rounded-lg animate-in fade-in duration-200">
				<div className="w-10 h-10 rounded-full bg-error/10 flex items-center justify-center flex-shrink-0">
					<AlertTriangle className="w-5 h-5 text-error" />
				</div>
				<div className="flex-1">
					<p className="font-medium text-error">Failed to load inbox</p>
					<p className="text-sm text-error/80">{error}</p>
				</div>
				<button
					type="button"
					onClick={() => fetchMessages()}
					className="px-3 py-1.5 text-sm rounded-md text-error hover:bg-error/10 transition-colors"
				>
					Retry
				</button>
			</div>
		);
	}

	const pendingCount = messages.filter((m) => !m.responded_at).length;

	return (
		<div className="space-y-4">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-4">
					<h2 className="text-2xl font-bold">Inbox</h2>
					{pendingCount > 0 && (
						<span className="text-sm px-2 py-1 rounded-full bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300">
							{pendingCount} pending
						</span>
					)}
				</div>
				<button
					type="button"
					onClick={handleRefresh}
					disabled={refreshing}
					className="flex items-center gap-2 px-3 py-2 text-sm rounded-md bg-[var(--secondary)] hover:bg-[var(--accent-muted)] transition-colors disabled:opacity-50 press-effect"
				>
					<RefreshCw className={cn("w-4 h-4", refreshing && "animate-spin")} />
					Refresh
				</button>
			</div>

			{/* Messages list */}
			{messages.length === 0 ? (
				<EmptyState
					icon={CheckCircle}
					title="All clear! ✨"
					description="No pending messages. Your agents are working independently and haven't needed your input."
					variant="card"
				/>
			) : (
				<div className="space-y-4">
					{messages.map((message) => (
						<InboxCard key={message.id} message={message} />
					))}
				</div>
			)}
		</div>
	);
}

function InboxCard({ message }: { message: InboxMessage }) {
	const [expanded, setExpanded] = useState(!message.responded_at);
	const isResponded = !!message.responded_at;
	const isEscalation = message.message_type === "escalation";

	const typeConfig = MESSAGE_TYPE_CONFIG[message.message_type];

	// Determine border color based on message type and status
	const getBorderClass = () => {
		if (isResponded) return "border-l-gray-300 dark:border-l-gray-600";
		if (isEscalation) return "border-l-red-500 dark:border-l-red-400";
		return "border-l-amber-500 dark:border-l-amber-400";
	};

	return (
		<div
			className={cn(
				"bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden",
				"border-l-4 stagger-item",
				getBorderClass(),
				isResponded && "opacity-70",
			)}
		>
			{/* Header - clickable for responded messages */}
			{/* biome-ignore lint/a11y/noStaticElementInteractions: role and keyboard handler provided conditionally */}
			<div
				className={cn("p-4", !expanded && "cursor-pointer hover:bg-[var(--secondary)]")}
				onClick={() => isResponded && setExpanded(!expanded)}
				onKeyDown={(e) => {
					if (isResponded && (e.key === "Enter" || e.key === " ")) {
						e.preventDefault();
						setExpanded(!expanded);
					}
				}}
				role={isResponded ? "button" : undefined}
				tabIndex={isResponded ? 0 : undefined}
			>
				<div className="flex items-start justify-between gap-4">
					<div className="flex-1 min-w-0">
						<div className="flex items-center gap-2 mb-1">
							<span className={cn("flex items-center gap-1.5", typeConfig.color)}>
								{typeConfig.icon}
								<span className="text-sm font-medium">{typeConfig.label}</span>
							</span>
							{isResponded && (
								<span className="flex items-center gap-1 text-green-600 dark:text-green-400 text-sm">
									<CheckCircle className="w-3 h-3" />
									Responded
								</span>
							)}
						</div>
						<div className="flex items-center flex-wrap gap-x-2">
							<Link
								to={`/tickets/${message.ticket_key}`}
								className="font-mono text-sm text-[var(--muted-foreground)] hover:text-[var(--foreground)]"
								onClick={(e) => e.stopPropagation()}
							>
								{message.ticket_key}
							</Link>
							<span className="text-[var(--muted-foreground)]">·</span>
							<span className="text-sm text-[var(--muted-foreground)] truncate">
								{message.ticket_title}
							</span>
						</div>
					</div>
					<span className="text-sm text-[var(--muted-foreground)] whitespace-nowrap">
						{formatRelativeTime(message.created_at)}
					</span>
				</div>
			</div>

			{/* Content - collapsible for responded messages */}
			{expanded && (
				<div className={cn("px-4 pb-4", isResponded && "pt-0 border-t border-[var(--border)]")}>
					{message.from_agent && (
						<p className="text-xs text-[var(--muted-foreground)] mb-2 pt-3">
							From: {message.from_agent}
						</p>
					)}
					<div className={cn(isResponded && "text-[var(--muted-foreground)]")}>
						<Markdown>{message.content}</Markdown>
					</div>

					{/* Response section (only shown if already responded) */}
					{message.responded_at && message.response && (
						<div className="mt-3 p-3 bg-[var(--secondary)] rounded-md border border-[var(--border)]">
							<p className="text-xs text-[var(--muted-foreground)] mb-1">
								Responded {formatRelativeTime(message.responded_at)}
							</p>
							<p className="text-sm whitespace-pre-wrap">{message.response}</p>
						</div>
					)}
				</div>
			)}

			{/* Collapsed preview for responded messages */}
			{!expanded && message.responded_at && (
				<div className="px-4 pb-3">
					<p className="text-xs text-[var(--muted-foreground)]">
						Click to expand · Responded {formatRelativeTime(message.responded_at)}
					</p>
				</div>
			)}
		</div>
	);
}
