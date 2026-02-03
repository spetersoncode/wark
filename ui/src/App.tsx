import {
	BarChart3,
	FolderKanban,
	Home,
	Inbox as InboxIcon,
	KanbanSquare,
	ListTodo,
} from "lucide-react";
import { NavLink, Route, Routes, useLocation } from "react-router-dom";
import { ColorSwatches } from "./components/ColorSwatches";
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

	return (
		<div className="min-h-screen bg-[var(--background)]">
			{/* Header */}
			<header className="border-b border-[var(--border)] bg-[var(--card)]">
				<div className="container mx-auto px-4 h-14 flex items-center justify-between">
					<div className="flex items-center gap-8">
						<h1 className="text-xl font-bold">wark</h1>
						<nav className="flex items-center gap-1">
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
												: "text-[var(--muted-foreground)] hover:text-[var(--foreground)] hover:bg-[var(--accent)]",
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
					</div>
				</div>
			</header>

			{/* Main content */}
			<main className="container mx-auto px-4 py-8">
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
			</main>
		</div>
	);
}

export default App;
