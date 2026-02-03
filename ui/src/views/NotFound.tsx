import { FileQuestion, Home } from "lucide-react";
import { Link, useLocation } from "react-router-dom";

export default function NotFound() {
	const location = useLocation();

	return (
		<div className="flex flex-col items-center justify-center min-h-[60vh] text-center">
			<div className="mb-6">
				<FileQuestion className="w-24 h-24 text-[var(--muted-foreground)]" />
			</div>
			<h2 className="text-3xl font-bold mb-2">Page not found</h2>
			<p className="text-[var(--muted-foreground)] mb-2">
				The page <code className="px-2 py-1 bg-[var(--secondary)] rounded text-sm">{location.pathname}</code> doesn't exist.
			</p>
			<p className="text-[var(--muted-foreground)] mb-8">
				Check the URL or head back to the dashboard.
			</p>
			<Link
				to="/"
				className="flex items-center gap-2 px-4 py-2 bg-[var(--primary)] text-[var(--primary-foreground)] rounded-md hover:opacity-90 transition-opacity"
			>
				<Home className="w-4 h-4" />
				Back to Dashboard
			</Link>
		</div>
	);
}
