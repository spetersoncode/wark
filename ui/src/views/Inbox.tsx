import {
	AlertCircle,
	CheckCircle,
	HelpCircle,
	Info,
	MessageSquare,
	RefreshCw,
	Star,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Markdown } from "../components/Markdown";
import { type InboxMessage, listInbox, type MessageType } from "../lib/api";
import { useAutoRefresh } from "../lib/hooks";
import { cn, formatRelativeTime } from "../lib/utils";

const MESSAGE_TYPE_CONFIG: Record<
	MessageType,
	{ label: string; icon: React.ReactNode; color: string }
> = {
	question: {
		label: "Question",
		icon: <HelpCircle className="w-4 h-4" />,
		color: "text-blue-600 dark:text-blue-400",
	},
	decision: {
		label: "Decision",
		icon: <Star className="w-4 h-4" />,
		color: "text-yellow-600 dark:text-yellow-400",
	},
	review: {
		label: "Review",
		icon: <MessageSquare className="w-4 h-4" />,
		color: "text-purple-600 dark:text-purple-400",
	},
	escalation: {
		label: "Escalation",
		icon: <AlertCircle className="w-4 h-4" />,
		color: "text-red-600 dark:text-red-400",
	},
	info: {
		label: "Info",
		icon: <Info className="w-4 h-4" />,
		color: "text-gray-600 dark:text-gray-400",
	},
};

export default function Inbox() {
	const [messages, setMessages] = useState<InboxMessage[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const fetchMessages = useCallback(async () => {
		try {
			// Always show only pending messages
			const data = await listInbox({ pending: true });
			setMessages(data);
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

	if (loading) {
		return (
			<div className="flex items-center justify-center h-64">
				<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-[var(--primary)]" />
			</div>
		);
	}

	if (error) {
		return (
			<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 text-red-700 dark:text-red-300">
				{error}
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
						<span className="text-sm px-2 py-1 rounded-full bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300">
							{pendingCount} pending
						</span>
					)}
				</div>
				<button
					type="button"
					onClick={handleRefresh}
					disabled={refreshing}
					className="flex items-center gap-2 px-3 py-2 text-sm rounded-md bg-[var(--secondary)] hover:bg-[var(--accent)] transition-colors disabled:opacity-50"
				>
					<RefreshCw className={cn("w-4 h-4", refreshing && "animate-spin")} />
					Refresh
				</button>
			</div>

			{/* Messages list */}
			{messages.length === 0 ? (
				<div className="text-center py-12 text-[var(--muted-foreground)]">No pending messages</div>
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

function InboxCard({
	message,
}: {
	message: InboxMessage;
}) {
	const [expanded, setExpanded] = useState(!message.responded_at);

	const typeConfig = MESSAGE_TYPE_CONFIG[message.message_type];

	return (
		<div
			className={cn(
				"bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden",
				!message.responded_at && "border-l-4 border-l-purple-500",
			)}
		>
			{/* Header */}
			<div className="p-4 border-b border-[var(--border)]">
				<div className="flex items-start justify-between gap-4">
					<div className="flex-1 min-w-0">
						<div className="flex items-center gap-2 mb-1">
							<span className={cn("flex items-center gap-1", typeConfig.color)}>
								{typeConfig.icon}
								<span className="text-sm font-medium">{typeConfig.label}</span>
							</span>
							{message.responded_at && (
								<span className="flex items-center gap-1 text-green-600 dark:text-green-400 text-sm">
									<CheckCircle className="w-3 h-3" />
									Responded
								</span>
							)}
						</div>
						<Link
							to={`/tickets/${message.ticket_key}`}
							className="font-mono text-sm text-[var(--muted-foreground)] hover:text-[var(--foreground)]"
						>
							{message.ticket_key}
						</Link>
						<span className="text-[var(--muted-foreground)] mx-2">•</span>
						<span className="text-sm">{message.ticket_title}</span>
					</div>
					<span className="text-sm text-[var(--muted-foreground)] whitespace-nowrap">
						{formatRelativeTime(message.created_at)}
					</span>
				</div>
			</div>

			{/* Content */}
			<div className="p-4">
				{message.from_agent && (
					<p className="text-xs text-[var(--muted-foreground)] mb-2">From: {message.from_agent}</p>
				)}
				<Markdown>{message.content}</Markdown>
			</div>

			{/* Response section (only shown if already responded) */}
			{message.responded_at && (
				<div className="px-4 pb-4">
					<button
						type="button"
						onClick={() => setExpanded(!expanded)}
						className="text-sm text-[var(--muted-foreground)] hover:text-[var(--foreground)]"
					>
						{expanded ? "Hide response" : "Show response"}
					</button>
					{expanded && (
						<div className="mt-2 p-3 bg-[var(--secondary)] rounded-md">
							<p className="text-xs text-[var(--muted-foreground)] mb-1">
								Responded {formatRelativeTime(message.responded_at)}
							</p>
							<p className="text-sm whitespace-pre-wrap">{message.response}</p>
						</div>
					)}
				</div>
			)}
		</div>
	);
}
