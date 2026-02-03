import { Monitor, Moon, Sun } from "lucide-react";
import { useEffect, useState } from "react";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "./ui/select";

type Theme = "light" | "dark" | "system";

const STORAGE_KEY = "wark-theme";

const THEME_OPTIONS: { value: Theme; label: string; icon: typeof Sun }[] = [
	{ value: "light", label: "Light", icon: Sun },
	{ value: "dark", label: "Dark", icon: Moon },
	{ value: "system", label: "System", icon: Monitor },
];

/**
 * Gets the system's preferred color scheme.
 */
function getSystemTheme(): "light" | "dark" {
	if (typeof window === "undefined") return "light";
	return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
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
 * ThemeToggle - Dropdown selector for light/dark/system modes.
 *
 * Shows current theme with icon, dropdown reveals all options.
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

	const currentOption = THEME_OPTIONS.find((opt) => opt.value === theme) ?? THEME_OPTIONS[2];
	const CurrentIcon = currentOption.icon;

	return (
		<Select value={theme} onValueChange={(value: Theme) => setTheme(value)}>
			<SelectTrigger
				size="sm"
				className="h-8 w-8 border-none bg-transparent px-0 shadow-none hover:bg-accent focus-visible:ring-0 focus-visible:ring-offset-0 [&>svg:last-child]:hidden"
				aria-label={`Theme: ${currentOption.label}`}
			>
				<SelectValue>
					<CurrentIcon className="size-4" />
				</SelectValue>
			</SelectTrigger>
			<SelectContent align="end">
				{THEME_OPTIONS.map(({ value, label, icon: Icon }) => (
					<SelectItem key={value} value={value}>
						<Icon className="size-4" />
						<span>{label}</span>
					</SelectItem>
				))}
			</SelectContent>
		</Select>
	);
}
