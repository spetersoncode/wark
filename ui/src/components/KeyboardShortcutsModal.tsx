import { Keyboard, X } from "lucide-react";
import {
	CATEGORY_LABELS,
	KEYBOARD_SHORTCUTS,
	type KeyboardShortcut,
} from "../lib/keyboard-shortcuts";
import { cn } from "../lib/utils";
import { Dialog, DialogClose, DialogContent, DialogHeader, DialogTitle } from "./ui/dialog";

interface KeyboardShortcutsModalProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
}

/**
 * Renders a single keyboard key/key combo
 */
function Kbd({ children }: { children: React.ReactNode }) {
	return (
		<kbd
			className={cn(
				"inline-flex items-center justify-center min-w-6 h-6 px-1.5",
				"text-xs font-mono font-medium",
				"bg-[var(--background)] border border-[var(--border)] rounded",
				"shadow-sm",
			)}
		>
			{children}
		</kbd>
	);
}

/**
 * Renders the keys for a shortcut, handling combinations like "g d" or "⌘K or /"
 */
function ShortcutKeys({ keys }: { keys: string }) {
	// Handle "or" combinations like "⌘K or /"
	if (keys.includes(" or ")) {
		const parts = keys.split(" or ");
		return (
			<div className="flex items-center gap-1.5">
				{parts.map((part, i) => (
					<span key={part} className="flex items-center gap-1">
						{i > 0 && <span className="text-xs text-[var(--muted-foreground)] mx-0.5">or</span>}
						<ShortcutKeys keys={part.trim()} />
					</span>
				))}
			</div>
		);
	}

	// Handle space-separated sequences like "g d"
	if (keys.includes(" ")) {
		const parts = keys.split(" ");
		return (
			<div className="flex items-center gap-1">
				{parts.map((part) => (
					<Kbd key={part}>{part}</Kbd>
				))}
			</div>
		);
	}

	// Single key
	return <Kbd>{keys}</Kbd>;
}

/**
 * Groups shortcuts by category
 */
function groupByCategory(shortcuts: KeyboardShortcut[]) {
	const groups: Record<string, KeyboardShortcut[]> = {};
	for (const shortcut of shortcuts) {
		if (!groups[shortcut.category]) {
			groups[shortcut.category] = [];
		}
		groups[shortcut.category].push(shortcut);
	}
	return groups;
}

export function KeyboardShortcutsModal({ open, onOpenChange }: KeyboardShortcutsModalProps) {
	const grouped = groupByCategory(KEYBOARD_SHORTCUTS);

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-md">
				<DialogHeader className="flex flex-row items-center justify-between">
					<div className="flex items-center gap-2">
						<Keyboard className="w-5 h-5 text-[var(--muted-foreground)]" />
						<DialogTitle>Keyboard Shortcuts</DialogTitle>
					</div>
					<DialogClose asChild>
						<button
							type="button"
							className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
						>
							<X className="h-4 w-4" />
							<span className="sr-only">Close</span>
						</button>
					</DialogClose>
				</DialogHeader>

				<div className="space-y-6">
					{(Object.keys(CATEGORY_LABELS) as Array<keyof typeof CATEGORY_LABELS>).map((category) => {
						const shortcuts = grouped[category];
						if (!shortcuts || shortcuts.length === 0) return null;

						return (
							<div key={category}>
								<h3 className="text-sm font-medium text-[var(--muted-foreground)] mb-2">
									{CATEGORY_LABELS[category]}
								</h3>
								<div className="space-y-2">
									{shortcuts.map((shortcut) => (
										<div key={shortcut.keys} className="flex items-center justify-between py-1">
											<span className="text-sm">{shortcut.description}</span>
											<ShortcutKeys keys={shortcut.keys} />
										</div>
									))}
								</div>
							</div>
						);
					})}
				</div>

				<div className="pt-2 border-t border-[var(--border)] mt-2">
					<p className="text-xs text-[var(--muted-foreground)] text-center">
						Press <Kbd>?</Kbd> anytime to show this help
					</p>
				</div>
			</DialogContent>
		</Dialog>
	);
}
