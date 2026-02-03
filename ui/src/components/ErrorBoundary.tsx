import { AlertCircle, RefreshCw } from "lucide-react";
import { Component, type ErrorInfo, type ReactNode } from "react";
import { cn } from "@/lib/utils";

interface ErrorBoundaryProps {
	children: ReactNode;
	/** Optional fallback component to show on error */
	fallback?: ReactNode;
	/** Callback when error occurs */
	onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface ErrorBoundaryState {
	hasError: boolean;
	error: Error | null;
}

/**
 * ErrorBoundary catches JavaScript errors in child components and displays
 * a friendly fallback UI instead of crashing the whole app.
 *
 * @example
 * <ErrorBoundary>
 *   <MyComponent />
 * </ErrorBoundary>
 *
 * @example
 * // With custom fallback
 * <ErrorBoundary fallback={<div>Something went wrong</div>}>
 *   <MyComponent />
 * </ErrorBoundary>
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
	constructor(props: ErrorBoundaryProps) {
		super(props);
		this.state = { hasError: false, error: null };
	}

	static getDerivedStateFromError(error: Error): ErrorBoundaryState {
		return { hasError: true, error };
	}

	componentDidCatch(error: Error, errorInfo: ErrorInfo) {
		console.error("ErrorBoundary caught an error:", error, errorInfo);
		this.props.onError?.(error, errorInfo);
	}

	handleRetry = () => {
		this.setState({ hasError: false, error: null });
	};

	render() {
		if (this.state.hasError) {
			if (this.props.fallback) {
				return this.props.fallback;
			}

			return <ErrorFallback error={this.state.error} onRetry={this.handleRetry} />;
		}

		return this.props.children;
	}
}

interface ErrorFallbackProps {
	error: Error | null;
	onRetry?: () => void;
	className?: string;
}

/**
 * ErrorFallback displays a friendly error message with optional retry button.
 * Can be used standalone or as part of ErrorBoundary.
 */
export function ErrorFallback({ error, onRetry, className }: ErrorFallbackProps) {
	return (
		<div
			className={cn(
				"flex flex-col items-center justify-center py-12 px-4 text-center",
				"animate-in fade-in duration-200",
				className,
			)}
		>
			<div className="w-16 h-16 rounded-full bg-error/10 flex items-center justify-center mb-4">
				<AlertCircle className="w-8 h-8 text-error" />
			</div>
			<h2 className="text-lg font-semibold text-foreground mb-2">Something went wrong</h2>
			<p className="text-sm text-foreground-muted mb-4 max-w-md">
				{error?.message || "An unexpected error occurred. Please try again."}
			</p>
			{onRetry && (
				<button
					type="button"
					onClick={onRetry}
					className={cn(
						"inline-flex items-center gap-2 px-4 py-2 text-sm font-medium",
						"bg-[var(--card)] border border-[var(--border)] rounded-md",
						"hover:bg-[var(--secondary)] hover:border-[var(--border-strong)]",
						"transition-colors press-effect",
					)}
				>
					<RefreshCw className="w-4 h-4" />
					Try again
				</button>
			)}
		</div>
	);
}

interface ErrorStateProps {
	title?: string;
	message: string;
	onRetry?: () => void;
	className?: string;
}

/**
 * ErrorState displays an inline error message with optional retry.
 * Use this for API errors and similar recoverable states.
 */
export function ErrorState({
	title = "Failed to load",
	message,
	onRetry,
	className,
}: ErrorStateProps) {
	return (
		<div
			className={cn(
				"flex items-center gap-3 p-4",
				"border border-error/20 bg-error/5 rounded-lg",
				"animate-in fade-in duration-200",
				className,
			)}
		>
			<AlertCircle className="w-5 h-5 text-error flex-shrink-0" />
			<div className="flex-1 min-w-0">
				<p className="font-medium text-error">{title}</p>
				<p className="text-sm text-error/80">{message}</p>
			</div>
			{onRetry && (
				<button
					type="button"
					onClick={onRetry}
					className={cn(
						"px-3 py-1.5 text-sm rounded-md",
						"text-error hover:bg-error/10",
						"transition-colors press-effect",
					)}
				>
					Retry
				</button>
			)}
		</div>
	);
}
