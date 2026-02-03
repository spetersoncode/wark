import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Auto-refresh interval in milliseconds (10 seconds)
 */
const AUTO_REFRESH_INTERVAL = 10_000;

/**
 * Hook that provides auto-refresh functionality with visibility detection.
 *
 * - Polls every 10 seconds when the tab is visible
 * - Pauses polling when the tab is hidden (using Page Visibility API)
 * - Provides manual refresh capability as fallback
 *
 * @param fetchFn - Async function to call for fetching data
 * @param deps - Dependencies array for the fetch function (similar to useCallback deps)
 * @returns Object with refreshing state and manual refresh trigger
 */
export function useAutoRefresh(
	fetchFn: () => Promise<void>,
	deps: React.DependencyList = [],
): {
	refreshing: boolean;
	refresh: () => void;
} {
	const [refreshing, setRefreshing] = useState(false);
	const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

	// Memoize the fetch function with provided dependencies
	// biome-ignore lint/correctness/useExhaustiveDependencies: deps is explicitly passed by caller
	const stableFetchFn = useCallback(fetchFn, deps);

	// Manual refresh handler
	const refresh = useCallback(() => {
		setRefreshing(true);
		stableFetchFn().finally(() => setRefreshing(false));
	}, [stableFetchFn]);

	// Silent background refresh (no spinner)
	const backgroundRefresh = useCallback(() => {
		stableFetchFn();
	}, [stableFetchFn]);

	// Start polling interval
	const startPolling = useCallback(() => {
		if (intervalRef.current) return; // Already polling
		intervalRef.current = setInterval(backgroundRefresh, AUTO_REFRESH_INTERVAL);
	}, [backgroundRefresh]);

	// Stop polling interval
	const stopPolling = useCallback(() => {
		if (intervalRef.current) {
			clearInterval(intervalRef.current);
			intervalRef.current = null;
		}
	}, []);

	useEffect(() => {
		// Handle visibility change
		const handleVisibilityChange = () => {
			if (document.hidden) {
				stopPolling();
			} else {
				// Refresh immediately when tab becomes visible, then resume polling
				backgroundRefresh();
				startPolling();
			}
		};

		// Start polling if tab is visible
		if (!document.hidden) {
			startPolling();
		}

		// Listen for visibility changes
		document.addEventListener("visibilitychange", handleVisibilityChange);

		return () => {
			stopPolling();
			document.removeEventListener("visibilitychange", handleVisibilityChange);
		};
	}, [startPolling, stopPolling, backgroundRefresh]);

	return { refreshing, refresh };
}
