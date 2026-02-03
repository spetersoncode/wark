/**
 * Keyboard shortcut definitions and types
 */

export interface KeyboardShortcut {
	keys: string;
	description: string;
	category: "navigation" | "actions" | "lists";
}

export const KEYBOARD_SHORTCUTS: KeyboardShortcut[] = [
	// Search
	{ keys: "⌘K or /", description: "Focus search bar", category: "actions" },

	// Navigation (g then X sequences)
	{ keys: "g d", description: "Go to Dashboard", category: "navigation" },
	{ keys: "g p", description: "Go to Projects", category: "navigation" },
	{ keys: "g t", description: "Go to Tickets", category: "navigation" },
	{ keys: "g b", description: "Go to Board", category: "navigation" },
	{ keys: "g i", description: "Go to Inbox", category: "navigation" },
	{ keys: "g a", description: "Go to Analytics", category: "navigation" },

	// Actions
	{ keys: "r", description: "Refresh current view", category: "actions" },
	{ keys: "?", description: "Show keyboard shortcuts", category: "actions" },
	{ keys: "Escape", description: "Close modal / blur focus", category: "actions" },

	// List navigation
	{ keys: "j", description: "Next item in list", category: "lists" },
	{ keys: "k", description: "Previous item in list", category: "lists" },
	{ keys: "Enter", description: "Open selected item", category: "lists" },
];

export const CATEGORY_LABELS: Record<KeyboardShortcut["category"], string> = {
	navigation: "Navigation",
	actions: "Actions",
	lists: "List Navigation",
};
