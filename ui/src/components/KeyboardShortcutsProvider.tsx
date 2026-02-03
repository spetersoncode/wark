import {
	createContext,
	type ReactNode,
	useCallback,
	useContext,
	useEffect,
	useRef,
	useState,
} from "react";
import { useNavigate } from "react-router-dom";
import { KeyboardShortcutsModal } from "./KeyboardShortcutsModal";

interface KeyboardShortcutsContextValue {
	/** Open the keyboard shortcuts modal */
	showShortcutsModal: () => void;
	/** Register a refresh handler for the current view */
	registerRefresh: (handler: () => void) => void;
	/** Unregister the refresh handler */
	unregisterRefresh: () => void;
	/** Register list navigation handlers */
	registerListNavigation: (handlers: ListNavigationHandlers) => void;
	/** Unregister list navigation handlers */
	unregisterListNavigation: () => void;
}

interface ListNavigationHandlers {
	/** Move to next item */
	onNext: () => void;
	/** Move to previous item */
	onPrevious: () => void;
	/** Open selected item */
	onSelect: () => void;
	/** Total items count (for bounds checking) */
	itemCount: number;
}

const KeyboardShortcutsContext = createContext<KeyboardShortcutsContextValue | null>(null);

/**
 * Check if an input element is currently focused
 */
function isInputFocused(): boolean {
	const active = document.activeElement;
	if (!active) return false;
	const tag = active.tagName.toLowerCase();
	return (
		tag === "input" ||
		tag === "textarea" ||
		tag === "select" ||
		(active as HTMLElement).isContentEditable
	);
}

/**
 * Navigation routes for "g then X" shortcuts
 */
const NAVIGATION_ROUTES: Record<string, string> = {
	d: "/", // Dashboard
	p: "/projects",
	t: "/tickets",
	b: "/board",
	i: "/inbox",
	a: "/analytics",
};

interface KeyboardShortcutsProviderProps {
	children: ReactNode;
}

export function KeyboardShortcutsProvider({ children }: KeyboardShortcutsProviderProps) {
	const navigate = useNavigate();
	const [modalOpen, setModalOpen] = useState(false);

	// Ref to track if we're in "g then X" mode
	const gPrefixActive = useRef(false);
	const gPrefixTimer = useRef<number | null>(null);

	// Ref for the current view's refresh handler
	const refreshHandler = useRef<(() => void) | null>(null);

	// Ref for list navigation handlers
	const listHandlers = useRef<ListNavigationHandlers | null>(null);

	const showShortcutsModal = useCallback(() => {
		setModalOpen(true);
	}, []);

	const registerRefresh = useCallback((handler: () => void) => {
		refreshHandler.current = handler;
	}, []);

	const unregisterRefresh = useCallback(() => {
		refreshHandler.current = null;
	}, []);

	const registerListNavigation = useCallback((handlers: ListNavigationHandlers) => {
		listHandlers.current = handlers;
	}, []);

	const unregisterListNavigation = useCallback(() => {
		listHandlers.current = null;
	}, []);

	// Clear g prefix mode
	const clearGPrefix = useCallback(() => {
		gPrefixActive.current = false;
		if (gPrefixTimer.current !== null) {
			window.clearTimeout(gPrefixTimer.current);
			gPrefixTimer.current = null;
		}
	}, []);

	// Main keyboard event handler
	useEffect(() => {
		const handleKeyDown = (e: KeyboardEvent) => {
			// Don't capture shortcuts when typing in inputs (except Escape)
			if (isInputFocused() && e.key !== "Escape") {
				return;
			}

			// Don't capture if modal is open (let modal handle Escape)
			if (modalOpen && e.key !== "Escape") {
				return;
			}

			// Handle Escape
			if (e.key === "Escape") {
				if (modalOpen) {
					setModalOpen(false);
					e.preventDefault();
					return;
				}
				// Clear g prefix mode
				clearGPrefix();
				// Blur any focused element
				(document.activeElement as HTMLElement)?.blur?.();
				return;
			}

			// Handle ? for shortcuts modal
			if (e.key === "?" && !e.metaKey && !e.ctrlKey && !e.altKey) {
				e.preventDefault();
				setModalOpen(true);
				return;
			}

			// Handle "g then X" navigation
			if (gPrefixActive.current) {
				const route = NAVIGATION_ROUTES[e.key.toLowerCase()];
				if (route) {
					e.preventDefault();
					navigate(route);
				}
				clearGPrefix();
				return;
			}

			// Start "g then X" sequence
			if (e.key === "g" && !e.metaKey && !e.ctrlKey && !e.altKey) {
				e.preventDefault();
				gPrefixActive.current = true;
				// Auto-clear after 1.5 seconds
				gPrefixTimer.current = window.setTimeout(clearGPrefix, 1500);
				return;
			}

			// Handle "r" for refresh
			if (e.key === "r" && !e.metaKey && !e.ctrlKey && !e.altKey) {
				if (refreshHandler.current) {
					e.preventDefault();
					refreshHandler.current();
				}
				return;
			}

			// Handle "j" for next item
			if (e.key === "j" && !e.metaKey && !e.ctrlKey && !e.altKey) {
				if (listHandlers.current) {
					e.preventDefault();
					listHandlers.current.onNext();
				}
				return;
			}

			// Handle "k" for previous item
			if (e.key === "k" && !e.metaKey && !e.ctrlKey && !e.altKey) {
				if (listHandlers.current) {
					e.preventDefault();
					listHandlers.current.onPrevious();
				}
				return;
			}

			// Handle "Enter" for select
			if (e.key === "Enter" && !e.metaKey && !e.ctrlKey && !e.altKey) {
				if (listHandlers.current) {
					e.preventDefault();
					listHandlers.current.onSelect();
				}
				return;
			}
		};

		document.addEventListener("keydown", handleKeyDown);
		return () => {
			document.removeEventListener("keydown", handleKeyDown);
			clearGPrefix();
		};
	}, [navigate, modalOpen, clearGPrefix]);

	const value: KeyboardShortcutsContextValue = {
		showShortcutsModal,
		registerRefresh,
		unregisterRefresh,
		registerListNavigation,
		unregisterListNavigation,
	};

	return (
		<KeyboardShortcutsContext.Provider value={value}>
			{children}
			<KeyboardShortcutsModal open={modalOpen} onOpenChange={setModalOpen} />
		</KeyboardShortcutsContext.Provider>
	);
}

/**
 * Hook to access keyboard shortcuts context
 */
export function useKeyboardShortcuts() {
	const context = useContext(KeyboardShortcutsContext);
	if (!context) {
		throw new Error("useKeyboardShortcuts must be used within KeyboardShortcutsProvider");
	}
	return context;
}

/**
 * Hook to register a refresh handler for the current view
 */
export function useRefreshShortcut(handler: () => void) {
	const { registerRefresh, unregisterRefresh } = useKeyboardShortcuts();

	useEffect(() => {
		registerRefresh(handler);
		return () => unregisterRefresh();
	}, [handler, registerRefresh, unregisterRefresh]);
}

/**
 * Hook to register list navigation for the current view
 */
export function useListNavigationShortcuts(handlers: ListNavigationHandlers) {
	const { registerListNavigation, unregisterListNavigation } = useKeyboardShortcuts();

	useEffect(() => {
		registerListNavigation(handlers);
		return () => unregisterListNavigation();
	}, [handlers, registerListNavigation, unregisterListNavigation]);
}
