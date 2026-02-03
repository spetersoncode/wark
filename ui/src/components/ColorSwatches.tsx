/**
 * Color Swatches - Design System Visual Test
 * Displays all oklch colors from the design system for verification.
 * Import this component temporarily to verify colors render correctly.
 */
export function ColorSwatches() {
	return (
		<div className="p-8 space-y-8">
			<h1 className="text-2xl font-bold">Design System Color Palette</h1>
			<p className="text-foreground-muted">oklch color space - Industrial theme</p>

			{/* Status Colors */}
			<section>
				<h2 className="text-lg font-semibold mb-4">Status Colors</h2>
				<div className="flex flex-wrap gap-4">
					<Swatch name="Ready" cssVar="--status-ready" />
					<Swatch name="In Progress" cssVar="--status-in-progress" />
					<Swatch name="Human" cssVar="--status-human" />
					<Swatch name="Review" cssVar="--status-review" />
					<Swatch name="Blocked" cssVar="--status-blocked" />
					<Swatch name="Closed" cssVar="--status-closed" />
				</div>
			</section>

			{/* Priority Colors */}
			<section>
				<h2 className="text-lg font-semibold mb-4">Priority Colors</h2>
				<div className="flex flex-wrap gap-4">
					<Swatch name="Highest" cssVar="--priority-highest" />
					<Swatch name="High" cssVar="--priority-high" />
					<Swatch name="Medium" cssVar="--priority-medium" />
					<Swatch name="Low" cssVar="--priority-low" />
					<Swatch name="Lowest" cssVar="--priority-lowest" />
				</div>
			</section>

			{/* Feedback Colors */}
			<section>
				<h2 className="text-lg font-semibold mb-4">Feedback Colors</h2>
				<div className="flex flex-wrap gap-4">
					<Swatch name="Success" cssVar="--success" />
					<Swatch name="Warning" cssVar="--warning" />
					<Swatch name="Error" cssVar="--error" />
					<Swatch name="Info" cssVar="--info" />
				</div>
			</section>

			{/* Accent */}
			<section>
				<h2 className="text-lg font-semibold mb-4">Accent Colors</h2>
				<div className="flex flex-wrap gap-4">
					<Swatch name="Accent" cssVar="--accent" />
					<Swatch name="Accent Hover" cssVar="--accent-hover" />
					<Swatch name="Accent Muted" cssVar="--accent-muted" />
				</div>
			</section>

			{/* Background Layers */}
			<section>
				<h2 className="text-lg font-semibold mb-4">Background Layers</h2>
				<div className="flex flex-wrap gap-4">
					<Swatch name="Background" cssVar="--background" border />
					<Swatch name="Background Subtle" cssVar="--background-subtle" border />
					<Swatch name="Background Muted" cssVar="--background-muted" border />
				</div>
			</section>

			{/* Foreground */}
			<section>
				<h2 className="text-lg font-semibold mb-4">Foreground (Text)</h2>
				<div className="flex flex-wrap gap-4">
					<Swatch name="Foreground" cssVar="--foreground" />
					<Swatch name="Foreground Muted" cssVar="--foreground-muted" />
					<Swatch name="Foreground Subtle" cssVar="--foreground-subtle" />
				</div>
			</section>

			{/* Borders */}
			<section>
				<h2 className="text-lg font-semibold mb-4">Borders</h2>
				<div className="flex flex-wrap gap-4">
					<Swatch name="Border" cssVar="--border" border />
					<Swatch name="Border Muted" cssVar="--border-muted" border />
					<Swatch name="Border Strong" cssVar="--border-strong" border />
				</div>
			</section>
		</div>
	);
}

function Swatch({
	name,
	cssVar,
	border = false,
}: {
	name: string;
	cssVar: string;
	border?: boolean;
}) {
	return (
		<div className="flex flex-col items-center gap-2">
			<div
				className={`w-16 h-16 rounded-md ${border ? "border border-border" : ""}`}
				style={{ backgroundColor: `var(${cssVar})` }}
			/>
			<span className="text-sm font-medium">{name}</span>
			<code className="text-xs text-foreground-subtle">{cssVar}</code>
		</div>
	);
}
