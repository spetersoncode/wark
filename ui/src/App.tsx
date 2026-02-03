import {
	BarChart3,
	FolderKanban,
	Home,
	Inbox as InboxIcon,
	KanbanSquare,
	ListTodo,
	Menu,
	X,
} from "lucide-react";
import { useEffect, useState } from "react";
import { Link, NavLink, Route, Routes, useLocation } from "react-router-dom";
import { ColorSwatches } from "./components/ColorSwatches";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { SearchBar } from "./components/SearchBar";
import { ThemeToggle } from "./components/ThemeToggle";
import { cn } from "./lib/utils";
import {
	Analytics,
	Board,
	ComponentDemo,
	Dashboard,
	Inbox,
	NotFound,
	Projects,
	TicketDetail,
	Tickets,
} from "./views";

const NAV_ITEMS = [
	{ to: "/", label: "Dashboard", icon: Home, end: true },
	{ to: "/projects", label: "Projects", icon: FolderKanban },
	{ to: "/tickets", label: "Tickets", icon: ListTodo },
	{ to: "/board", label: "Board", icon: KanbanSquare },
	{ to: "/inbox", label: "Inbox", icon: InboxIcon },
	{ to: "/analytics", label: "Analytics", icon: BarChart3 },
];

function App() {
	const location = useLocation();
	const isTicketDetail = location.pathname.startsWith("/tickets/");
	const [mobileNavOpen, setMobileNavOpen] = useState(false);

	// Close mobile nav on route change
	// biome-ignore lint/correctness/useExhaustiveDependencies: We want to close nav when path changes
	useEffect(() => {
		setMobileNavOpen(false);
	}, [location.pathname]);

	// Close mobile nav on escape key
	useEffect(() => {
		const handleEscape = (e: KeyboardEvent) => {
			if (e.key === "Escape") setMobileNavOpen(false);
		};
		document.addEventListener("keydown", handleEscape);
		return () => document.removeEventListener("keydown", handleEscape);
	}, []);

	return (
		<div className="min-h-screen bg-[var(--background)]">
			{/* Header */}
			<header className="sticky top-0 z-50 border-b border-[var(--border)] bg-[var(--card)]/95 backdrop-blur-sm">
				<div className="container mx-auto px-4 h-14 flex items-center justify-between">
					<div className="flex items-center gap-8">
						<Link to="/" className="text-xl font-bold hover:opacity-80 transition-opacity">wark</Link>
						{/* Desktop Navigation */}
						<nav className="hidden md:flex items-center gap-1">
							{NAV_ITEMS.map(({ to, label, icon: Icon, end }) => (
								<NavLink
									key={to}
									to={to}
									end={end}
									className={({ isActive }) =>
										cn(
											"flex items-center gap-2 px-3 py-2 text-sm rounded-md transition-colors",
											isActive || (isTicketDetail && to === "/board")
												? "bg-[var(--secondary)] text-[var(--foreground)]"
												: "text-[var(--muted-foreground)] hover:text-[var(--foreground)] hover:bg-[var(--accent-muted)]",
										)
									}
								>
									<Icon className="w-4 h-4" />
									{label}
								</NavLink>
							))}
						</nav>
					</div>
					<div className="flex items-center gap-2">
						<SearchBar />
						<ThemeToggle />
						{/* Mobile menu button */}
						<button
							type="button"
							onClick={() => setMobileNavOpen(!mobileNavOpen)}
							className="md:hidden p-2 rounded-md hover:bg-[var(--secondary)] transition-colors press-effect"
							aria-label={mobileNavOpen ? "Close menu" : "Open menu"}
						>
							{mobileNavOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
						</button>
					</div>
				</div>

				{/* Mobile Navigation Dropdown */}
				{mobileNavOpen && (
					<nav className="md:hidden border-t border-[var(--border)] bg-[var(--card)] animate-in slide-in-from-top-2 duration-200">
						<div className="container mx-auto px-4 py-2">
							{NAV_ITEMS.map(({ to, label, icon: Icon, end }) => (
								<NavLink
									key={to}
									to={to}
									end={end}
									className={({ isActive }) =>
										cn(
											"flex items-center gap-3 px-3 py-3 text-sm rounded-md transition-colors",
											isActive || (isTicketDetail && to === "/board")
												? "bg-[var(--secondary)] text-[var(--foreground)]"
												: "text-[var(--muted-foreground)] hover:text-[var(--foreground)] hover:bg-[var(--accent-muted)]",
										)
									}
								>
									<Icon className="w-5 h-5" />
									{label}
								</NavLink>
							))}
						</div>
					</nav>
				)}
			</header>

			{/* Mobile nav backdrop */}
			{mobileNavOpen && (
				<button
					type="button"
					className="fixed inset-0 z-40 bg-black/20 md:hidden"
					onClick={() => setMobileNavOpen(false)}
					aria-label="Close menu"
				/>
			)}

			{/* Main content with page transition */}
			<main className="container mx-auto px-4 py-6 md:py-8">
				<ErrorBoundary>
					<div key={location.pathname} className="page-enter">
						<Routes>
							<Route path="/" element={<Dashboard />} />
							<Route path="/dashboard" element={<Dashboard />} />
							<Route path="/projects" element={<Projects />} />
							<Route path="/tickets" element={<Tickets />} />
							<Route path="/board" element={<Board />} />
							<Route path="/inbox" element={<Inbox />} />
							<Route path="/analytics" element={<Analytics />} />
							<Route path="/tickets/:key" element={<TicketDetail />} />
							<Route path="/colors" element={<ColorSwatches />} />
							<Route path="/components" element={<ComponentDemo />} />
							<Route path="*" element={<NotFound />} />
						</Routes>
					</div>
				</ErrorBoundary>
			</main>
		</div>
	);
}

export default App;
