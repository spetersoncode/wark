import { Monitor, Moon, Sun } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Button } from "./ui/button";

type Theme = "light" | "dark" | "system";

const STORAGE_KEY = "wark-theme";

/**
 * Gets the system's preferred color scheme.
 */
function getSystemTheme(): "light" | "dark" {
	if (typeof window === "undefined") return "light";
	return window.matchMedia("(prefers-color-scheme: dark)").matches
		? "dark"
		: "light";
}

/**
 * Applies the theme to the document root.
 */
function applyTheme(theme: Theme) {
	const root = document.documentElement;
	const effectiveTheme = theme === "system" ? getSystemTheme() : theme;

	if (effectiveTheme === "dark") {
		root.classList.add("dark");
	} else {
		root.classList.remove("dark");
	}
}

/**
 * Gets the stored theme preference from localStorage.
 */
function getStoredTheme(): Theme {
	if (typeof window === "undefined") return "system";
	const stored = localStorage.getItem(STORAGE_KEY);
	if (stored === "light" || stored === "dark" || stored === "system") {
		return stored;
	}
	return "system";
}

/**
 * ThemeToggle - Cycles through light/dark/system modes.
 *
 * Click to cycle: light → dark → system → light...
 */
export function ThemeToggle() {
	const [theme, setTheme] = useState<Theme>(getStoredTheme);

	// Apply theme on mount and when it changes
	useEffect(() => {
		applyTheme(theme);
		localStorage.setItem(STORAGE_KEY, theme);
	}, [theme]);

	// Listen for system theme changes when in system mode
	useEffect(() => {
		if (theme !== "system") return;

		const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
		const handler = () => applyTheme("system");

		mediaQuery.addEventListener("change", handler);
		return () => mediaQuery.removeEventListener("change", handler);
	}, [theme]);

	const cycleTheme = useCallback(() => {
		setTheme((current) => {
			switch (current) {
				case "light":
					return "dark";
				case "dark":
					return "system";
				case "system":
					return "light";
			}
		});
	}, []);

	const Icon = theme === "light" ? Sun : theme === "dark" ? Moon : Monitor;
	const label =
		theme === "light" ? "Light" : theme === "dark" ? "Dark" : "System";

	return (
		<Button
			variant="ghost"
			size="icon"
			onClick={cycleTheme}
			title={`Theme: ${label} (click to change)`}
			aria-label={`Current theme: ${label}. Click to change.`}
		>
			<Icon className="size-4" />
		</Button>
	);
}
