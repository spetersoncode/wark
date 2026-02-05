import { Search, X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { searchTickets, type Ticket } from "../lib/api";
import { cn } from "../lib/utils";

/** Status badge color mapping */
const STATUS_COLORS: Record<string, string> = {
	blocked: "bg-red-500/20 text-red-400",
	ready: "bg-green-500/20 text-green-400",
	working: "bg-blue-500/20 text-blue-400",
	human: "bg-yellow-500/20 text-yellow-400",
	review: "bg-purple-500/20 text-purple-400",
	closed: "bg-gray-500/20 text-gray-400",
};

export function SearchBar() {
	const [query, setQuery] = useState("");
	const [results, setResults] = useState<Ticket[]>([]);
	const [isOpen, setIsOpen] = useState(false);
	const [isLoading, setIsLoading] = useState(false);
	const [selectedIndex, setSelectedIndex] = useState(-1);
	const inputRef = useRef<HTMLInputElement>(null);
	const containerRef = useRef<HTMLDivElement>(null);
	const navigate = useNavigate();

	// Debounced search
	useEffect(() => {
		if (!query.trim()) {
			setResults([]);
			setIsOpen(false);
			return;
		}

		const timer = setTimeout(async () => {
			setIsLoading(true);
			try {
				const tickets = await searchTickets(query, 10);
				setResults(tickets);
				setIsOpen(true);
				setSelectedIndex(-1);
			} catch (error) {
				console.error("Search failed:", error);
				setResults([]);
			} finally {
				setIsLoading(false);
			}
		}, 200);

		return () => clearTimeout(timer);
	}, [query]);

	// Global keyboard shortcut
	useEffect(() => {
		const handleKeyDown = (e: KeyboardEvent) => {
			// "/" or Cmd/Ctrl+K to focus search
			if ((e.key === "/" && !isInputFocused()) || ((e.metaKey || e.ctrlKey) && e.key === "k")) {
				e.preventDefault();
				inputRef.current?.focus();
				inputRef.current?.select();
			}
		};

		document.addEventListener("keydown", handleKeyDown);
		return () => document.removeEventListener("keydown", handleKeyDown);
	}, []);

	// Click outside to close
	useEffect(() => {
		const handleClickOutside = (e: MouseEvent) => {
			if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
				setIsOpen(false);
			}
		};

		document.addEventListener("mousedown", handleClickOutside);
		return () => document.removeEventListener("mousedown", handleClickOutside);
	}, []);

	const handleKeyDown = useCallback(
		(e: React.KeyboardEvent) => {
			if (!isOpen || results.length === 0) {
				if (e.key === "Escape") {
					inputRef.current?.blur();
					setIsOpen(false);
				}
				return;
			}

			switch (e.key) {
				case "ArrowDown":
					e.preventDefault();
					setSelectedIndex((prev) => (prev < results.length - 1 ? prev + 1 : prev));
					break;
				case "ArrowUp":
					e.preventDefault();
					setSelectedIndex((prev) => (prev > 0 ? prev - 1 : -1));
					break;
				case "Enter":
					e.preventDefault();
					{
						const ticketKey =
							selectedIndex >= 0 && selectedIndex < results.length
								? results[selectedIndex].ticket_key
								: results.length > 0
									? results[0].ticket_key
									: null;
						if (ticketKey) {
							navigate(`/tickets/${ticketKey}`);
							setQuery("");
							setResults([]);
							setIsOpen(false);
							inputRef.current?.blur();
						}
					}
					break;
				case "Escape":
					e.preventDefault();
					setIsOpen(false);
					inputRef.current?.blur();
					break;
			}
		},
		[isOpen, results, selectedIndex, navigate],
	);

	const navigateToTicket = (key: string) => {
		navigate(`/tickets/${key}`);
		setQuery("");
		setResults([]);
		setIsOpen(false);
		inputRef.current?.blur();
	};

	const clearSearch = () => {
		setQuery("");
		setResults([]);
		setIsOpen(false);
		inputRef.current?.focus();
	};

	return (
		<div ref={containerRef} className="relative">
			<div className="relative flex items-center">
				<Search className="absolute left-3 w-4 h-4 text-[var(--muted-foreground)]" />
				<input
					ref={inputRef}
					type="text"
					value={query}
					onChange={(e) => setQuery(e.target.value)}
					onFocus={() => query.trim() && results.length > 0 && setIsOpen(true)}
					onKeyDown={handleKeyDown}
					placeholder="Search tickets..."
					className={cn(
						"w-64 pl-9 pr-8 py-1.5 text-sm rounded-md",
						"bg-[var(--secondary)] border border-[var(--border)]",
						"text-[var(--foreground)] placeholder:text-[var(--muted-foreground)]",
						"focus:outline-none focus:ring-2 focus:ring-[var(--ring)]",
						"transition-all duration-200",
					)}
				/>
				{query && (
					<button
						type="button"
						onClick={clearSearch}
						className="absolute right-2 p-0.5 rounded hover:bg-[var(--accent)]"
					>
						<X className="w-3.5 h-3.5 text-[var(--muted-foreground)]" />
					</button>
				)}
				<kbd className="hidden sm:flex absolute right-2 items-center gap-0.5 px-1.5 py-0.5 text-[10px] font-mono text-[var(--muted-foreground)] bg-[var(--background)] rounded border border-[var(--border)]">
					{query ? null : (
						<>
							<span className="text-[9px]">⌘</span>K
						</>
					)}
				</kbd>
			</div>

			{/* Results dropdown */}
			{isOpen && (
				<div className="absolute top-full left-0 right-0 mt-1 py-1 bg-[var(--card)] border border-[var(--border)] rounded-md shadow-lg z-50 max-h-80 overflow-y-auto">
					{isLoading ? (
						<div className="px-3 py-2 text-sm text-[var(--muted-foreground)]">Searching...</div>
					) : results.length === 0 ? (
						<div className="px-3 py-2 text-sm text-[var(--muted-foreground)]">No tickets found</div>
					) : (
						results.map((ticket, index) => (
							<button
								key={ticket.id}
								type="button"
								onClick={() => navigateToTicket(ticket.ticket_key)}
								onMouseEnter={() => setSelectedIndex(index)}
								className={cn(
									"w-full px-3 py-2 text-left flex items-start gap-3",
									"hover:bg-[var(--accent)] transition-colors",
									selectedIndex === index && "bg-[var(--accent)]",
								)}
							>
								<span className="font-mono text-xs text-[var(--muted-foreground)] shrink-0 pt-0.5">
									{ticket.ticket_key}
								</span>
								<div className="flex-1 min-w-0">
									<div className="text-sm truncate">{ticket.title}</div>
									<div className="flex items-center gap-2 mt-0.5">
										<span
											className={cn(
												"text-[10px] px-1.5 py-0.5 rounded",
												STATUS_COLORS[ticket.status] || STATUS_COLORS.closed,
											)}
										>
											{ticket.status.replace("_", " ")}
										</span>
									</div>
								</div>
							</button>
						))
					)}
				</div>
			)}
		</div>
	);
}

/** Check if an input element is currently focused */
function isInputFocused(): boolean {
	const active = document.activeElement;
	if (!active) return false;
	const tag = active.tagName.toLowerCase();
	return tag === "input" || tag === "textarea" || (active as HTMLElement).isContentEditable;
}
