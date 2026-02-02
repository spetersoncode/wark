import type { Components } from "react-markdown";
import ReactMarkdown from "react-markdown";

interface MarkdownProps {
	children: string;
	className?: string;
}

const components: Components = {
	// Headers
	h1: ({ children }) => <h1 className="text-2xl font-bold mt-6 mb-4 first:mt-0">{children}</h1>,
	h2: ({ children }) => <h2 className="text-xl font-semibold mt-5 mb-3 first:mt-0">{children}</h2>,
	h3: ({ children }) => <h3 className="text-lg font-semibold mt-4 mb-2 first:mt-0">{children}</h3>,
	h4: ({ children }) => (
		<h4 className="text-base font-semibold mt-3 mb-2 first:mt-0">{children}</h4>
	),
	h5: ({ children }) => <h5 className="text-sm font-semibold mt-3 mb-1 first:mt-0">{children}</h5>,
	h6: ({ children }) => (
		<h6 className="text-sm font-semibold mt-3 mb-1 text-[var(--muted-foreground)] first:mt-0">
			{children}
		</h6>
	),

	// Paragraphs
	p: ({ children }) => <p className="mb-3 last:mb-0 leading-relaxed">{children}</p>,

	// Lists
	ul: ({ children }) => <ul className="mb-3 ml-6 list-disc space-y-1">{children}</ul>,
	ol: ({ children }) => <ol className="mb-3 ml-6 list-decimal space-y-1">{children}</ol>,
	li: ({ children }) => <li className="leading-relaxed">{children}</li>,

	// Code
	code: ({ className, children }) => {
		const isInline = !className;
		if (isInline) {
			return (
				<code className="px-1.5 py-0.5 rounded bg-[var(--secondary)] font-mono text-sm">
					{children}
				</code>
			);
		}
		// Code block (language from className like "language-js")
		return <code className="block overflow-x-auto font-mono text-sm">{children}</code>;
	},
	pre: ({ children }) => (
		<pre className="mb-3 p-4 rounded-lg bg-[var(--secondary)] overflow-x-auto">{children}</pre>
	),

	// Blockquotes
	blockquote: ({ children }) => (
		<blockquote className="mb-3 pl-4 border-l-4 border-[var(--border)] text-[var(--muted-foreground)] italic">
			{children}
		</blockquote>
	),

	// Links
	a: ({ href, children }) => (
		<a
			href={href}
			target="_blank"
			rel="noopener noreferrer"
			className="text-blue-600 dark:text-blue-400 hover:underline"
		>
			{children}
		</a>
	),

	// Emphasis
	strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
	em: ({ children }) => <em className="italic">{children}</em>,

	// Horizontal rule
	hr: () => <hr className="my-4 border-[var(--border)]" />,

	// Tables
	table: ({ children }) => (
		<div className="mb-3 overflow-x-auto">
			<table className="min-w-full border-collapse">{children}</table>
		</div>
	),
	thead: ({ children }) => <thead className="bg-[var(--secondary)]">{children}</thead>,
	tbody: ({ children }) => <tbody>{children}</tbody>,
	tr: ({ children }) => <tr className="border-b border-[var(--border)]">{children}</tr>,
	th: ({ children }) => <th className="px-3 py-2 text-left font-semibold text-sm">{children}</th>,
	td: ({ children }) => <td className="px-3 py-2 text-sm">{children}</td>,
};

/**
 * Renders markdown content with consistent styling.
 */
export function Markdown({ children, className = "" }: MarkdownProps) {
	return (
		<div className={`text-sm ${className}`}>
			<ReactMarkdown components={components}>{children}</ReactMarkdown>
		</div>
	);
}
