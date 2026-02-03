/**
 * Wark API Client
 *
 * This module provides typed functions for interacting with the wark server's REST API.
 * All endpoints are relative to /api and proxied through Vite in development.
 */

const API_BASE = "/api";

/** HTTP response error with status and message */
export class ApiError extends Error {
	status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = "ApiError";
		this.status = status;
	}
}

/** Generic fetch wrapper with error handling */
async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
	const res = await fetch(`${API_BASE}${endpoint}`, {
		...options,
		headers: {
			"Content-Type": "application/json",
			...options?.headers,
		},
	});

	if (!res.ok) {
		const error = await res.json().catch(() => ({ error: res.statusText }));
		throw new ApiError(res.status, error.error || error.message || "Unknown error");
	}

	return res.json();
}

// =============================================================================
// Types
// =============================================================================

export interface Project {
	id: number;
	key: string;
	name: string;
	description?: string;
	created_at: string;
	updated_at: string;
}

export interface ProjectStats {
	total_tickets: number;
	blocked_count: number;
	ready_count: number;
	in_progress_count: number;
	human_count: number;
	review_count: number;
	closed_completed_count: number;
	closed_other_count: number;
}

export interface ProjectWithStats extends Project {
	stats: ProjectStats;
}

export type TicketStatus = "blocked" | "ready" | "in_progress" | "human" | "review" | "closed";

export type TicketPriority = "highest" | "high" | "medium" | "low" | "lowest";

export type TicketComplexity = "trivial" | "small" | "medium" | "large" | "xlarge";

export type Resolution = "completed" | "wont_do" | "duplicate" | "invalid" | "obsolete";

export interface Ticket {
	id: number;
	project_id: number;
	project_key: string;
	ticket_key: string;
	number: number;
	title: string;
	description?: string;
	status: TicketStatus;
	priority: TicketPriority;
	complexity: TicketComplexity;
	resolution?: Resolution;
	branch_name?: string;
	human_flag_reason?: string;
	retry_count: number;
	max_retries: number;
	parent_ticket_id?: number;
	created_at: string;
	updated_at: string;
	completed_at?: string;
}

export interface Claim {
	id: number;
	ticket_id: number;
	worker_id: string;
	claimed_at: string;
	expires_at: string;
	released_at?: string;
	status: "active" | "completed" | "expired" | "released";
}

export type MessageType = "question" | "decision" | "review" | "escalation" | "info";

export interface InboxMessage {
	id: number;
	ticket_id: number;
	ticket_key: string;
	ticket_title: string;
	message_type: MessageType;
	content: string;
	from_agent?: string;
	response?: string;
	created_at: string;
	responded_at?: string;
}

export interface ActivityLog {
	id: number;
	ticket_id: number;
	action: string;
	actor_type: "human" | "agent" | "system";
	actor_id?: string;
	summary?: string;
	details?: string;
	created_at: string;
}

export interface StatusResult {
	workable: number;
	in_progress: number;
	blocked_deps: number;
	blocked_human: number;
	pending_inbox: number;
	expiring_soon: Array<{
		ticket_key: string;
		worker_id: string;
		minutes_left: number;
	}>;
	recent_activity: Array<{
		ticket_key: string;
		action: string;
		age: string;
		summary: string;
	}>;
	project?: string;
}

// =============================================================================
// API Functions
// =============================================================================

// Health check
export const health = () => fetchApi<{ status: string; database: string }>("/health");

// Status
export const getStatus = (project?: string) =>
	fetchApi<StatusResult>(`/status${project ? `?project=${project}` : ""}`);

// Projects
export const listProjects = () => fetchApi<ProjectWithStats[]>("/projects");

export const getProject = (key: string) => fetchApi<ProjectWithStats>(`/projects/${key}`);

export const createProject = (data: { key: string; name: string; description?: string }) =>
	fetchApi<Project>("/projects", {
		method: "POST",
		body: JSON.stringify(data),
	});

// Tickets
export interface TicketListParams {
	project?: string;
	status?: TicketStatus;
	priority?: TicketPriority;
	complexity?: TicketComplexity;
	workable?: boolean;
	limit?: number;
}

export const listTickets = (params?: TicketListParams) => {
	const query = new URLSearchParams();
	if (params?.project) query.set("project", params.project);
	if (params?.status) query.set("status", params.status);
	if (params?.priority) query.set("priority", params.priority);
	if (params?.complexity) query.set("complexity", params.complexity);
	if (params?.workable) query.set("workable", "true");
	if (params?.limit) query.set("limit", params.limit.toString());
	const queryStr = query.toString();
	return fetchApi<Ticket[]>(`/tickets${queryStr ? `?${queryStr}` : ""}`);
};

export const getTicket = (key: string) =>
	fetchApi<{
		ticket: Ticket;
		dependencies: Ticket[];
		dependents: Ticket[];
		claim?: Claim;
		history: ActivityLog[];
	}>(`/tickets/${key}`);

export const createTicket = (
	projectKey: string,
	data: {
		title: string;
		description?: string;
		priority?: TicketPriority;
		complexity?: TicketComplexity;
		depends_on?: string[];
	},
) =>
	fetchApi<Ticket>(`/projects/${projectKey}/tickets`, {
		method: "POST",
		body: JSON.stringify(data),
	});

export const updateTicket = (
	key: string,
	data: {
		title?: string;
		description?: string;
		priority?: TicketPriority;
		complexity?: TicketComplexity;
	},
) =>
	fetchApi<Ticket>(`/tickets/${key}`, {
		method: "PATCH",
		body: JSON.stringify(data),
	});

// Inbox
export interface InboxListParams {
	pending?: boolean;
	project?: string;
	type?: MessageType;
}

export const listInbox = (params?: InboxListParams) => {
	const query = new URLSearchParams();
	if (params?.pending !== undefined) query.set("pending", params.pending.toString());
	if (params?.project) query.set("project", params.project);
	if (params?.type) query.set("type", params.type);
	const queryStr = query.toString();
	return fetchApi<InboxMessage[]>(`/inbox${queryStr ? `?${queryStr}` : ""}`);
};

export const getInboxMessage = (id: number) => fetchApi<InboxMessage>(`/inbox/${id}`);

export const respondToInbox = (id: number, response: string) =>
	fetchApi<InboxMessage>(`/inbox/${id}/respond`, {
		method: "POST",
		body: JSON.stringify({ response }),
	});

// Claims
export const listClaims = (active?: boolean) =>
	fetchApi<Array<Claim & { ticket_key: string; ticket_title: string }>>(
		`/claims${active ? "?active=true" : ""}`,
	);

// Search
export const searchTickets = (query: string, limit?: number) => {
	const params = new URLSearchParams({ q: query });
	if (limit) params.set("limit", limit.toString());
	return fetchApi<Ticket[]>(`/tickets/search?${params.toString()}`);
};

// Analytics
export interface SuccessMetrics {
	total_closed: number;
	completed_count: number;
	other_resolutions: number;
	success_rate: number;
	tickets_with_retries: number;
	total_tickets: number;
	retry_rate: number;
	avg_retries_on_failed: number;
}

export interface HumanInteractionMetrics {
	total_tickets: number;
	human_interventions: number;
	human_intervention_rate: number;
	total_inbox_messages: number;
	responded_messages: number;
	avg_response_time_hours: number;
}

export interface CycleTimeByComplexity {
	complexity: TicketComplexity;
	ticket_count: number;
	avg_cycle_hours: number;
}

export interface ThroughputMetrics {
	completed_today: number;
	completed_week: number;
	completed_month: number;
}

export interface WIPByStatus {
	status: string;
	count: number;
}

export interface TrendDataPoint {
	date: string;
	count: number;
}

export interface AnalyticsFilter {
	project?: string;
	since?: string;
	until?: string;
	trend_days?: number;
}

export interface AnalyticsResult {
	success: SuccessMetrics;
	human_interaction: HumanInteractionMetrics;
	cycle_time: CycleTimeByComplexity[];
	throughput: ThroughputMetrics;
	wip: WIPByStatus[];
	completion_trend: TrendDataPoint[];
	filter: {
		project?: string;
		since?: string;
		until?: string;
		trend_days: number;
	};
}

export const getAnalytics = (params?: AnalyticsFilter) => {
	const query = new URLSearchParams();
	if (params?.project) query.set("project", params.project);
	if (params?.since) query.set("since", params.since);
	if (params?.until) query.set("until", params.until);
	if (params?.trend_days) query.set("trend_days", params.trend_days.toString());
	const queryStr = query.toString();
	return fetchApi<AnalyticsResult>(`/analytics${queryStr ? `?${queryStr}` : ""}`);
};
