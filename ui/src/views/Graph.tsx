import { RefreshCw, ZoomIn, ZoomOut } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { listTickets, type Ticket } from "../lib/api";
import { cn } from "../lib/utils";

interface GraphNode {
	id: string;
	ticket: Ticket;
	x: number;
	y: number;
	vx: number;
	vy: number;
}

interface Edge {
	source: string;
	target: string;
}

const NODE_RADIUS = 40;
const LINK_DISTANCE = 150;
const REPULSION = 500;
const ATTRACTION = 0.05;
const DAMPING = 0.8;
const MIN_VELOCITY = 0.1;

export default function Graph() {
	const navigate = useNavigate();
	const [tickets, setTickets] = useState<Ticket[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [refreshing, setRefreshing] = useState(false);
	const [nodes, setNodes] = useState<GraphNode[]>([]);
	const [edges, setEdges] = useState<Edge[]>([]);
	const [zoom, setZoom] = useState(1);
	const [pan, setPan] = useState({ x: 0, y: 0 });
	const [isDragging, setIsDragging] = useState(false);
	const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
	const [selectedNode, setSelectedNode] = useState<string | null>(null);
	const [hoveredNode, setHoveredNode] = useState<string | null>(null);

	const fetchTickets = useCallback(async () => {
		try {
			const data = await listTickets({ limit: 100 });
			setTickets(data);
			setError(null);
		} catch (e) {
			setError(e instanceof Error ? e.message : "Failed to fetch tickets");
		} finally {
			setLoading(false);
			setRefreshing(false);
		}
	}, []);

	useEffect(() => {
		fetchTickets();
	}, [fetchTickets]);

	// Build graph from tickets (we need dependency info which isn't in list response)
	// For now, show all tickets as nodes without edges
	// In a real implementation, you'd need an API that returns dependency relationships
	useEffect(() => {
		if (tickets.length === 0) return;

		// Initialize nodes with random positions
		const width = 800;
		const height = 600;
		const initialNodes: GraphNode[] = tickets.map((ticket) => ({
			id: ticket.ticket_key,
			ticket,
			x: width / 2 + (Math.random() - 0.5) * 400,
			y: height / 2 + (Math.random() - 0.5) * 300,
			vx: 0,
			vy: 0,
		}));

		// For demo purposes, create edges based on ticket numbers (sequential dependencies)
		// In reality, you'd get this from the API
		const demoEdges: Edge[] = [];
		const ticketsByProject = new Map<string, Ticket[]>();
		for (const t of tickets) {
			const proj = ticketsByProject.get(t.project_key) || [];
			proj.push(t);
			ticketsByProject.set(t.project_key, proj);
		}

		// Create some demo edges (in practice, fetch from deps API)
		for (const [, projTickets] of ticketsByProject) {
			const sorted = [...projTickets].sort((a, b) => a.number - b.number);
			for (let i = 1; i < sorted.length && i < 5; i++) {
				// Limit edges for visibility
				if (Math.random() > 0.6) {
					demoEdges.push({
						source: sorted[Math.floor(Math.random() * i)].ticket_key,
						target: sorted[i].ticket_key,
					});
				}
			}
		}

		setNodes(initialNodes);
		setEdges(demoEdges);
	}, [tickets]);

	// Force-directed layout simulation
	useEffect(() => {
		if (nodes.length === 0) return;

		let animationFrame: number;
		let running = true;

		function simulate() {
			if (!running) return;

			setNodes((currentNodes) => {
				const newNodes = currentNodes.map((node) => ({ ...node }));
				const nodeMap = new Map(newNodes.map((n) => [n.id, n]));

				// Apply forces
				for (const node of newNodes) {
					// Repulsion from other nodes
					for (const other of newNodes) {
						if (node.id === other.id) continue;
						const dx = node.x - other.x;
						const dy = node.y - other.y;
						const dist = Math.sqrt(dx * dx + dy * dy) || 1;
						const force = REPULSION / (dist * dist);
						node.vx += (dx / dist) * force;
						node.vy += (dy / dist) * force;
					}

					// Center gravity
					node.vx -= node.x * 0.001;
					node.vy -= node.y * 0.001;
				}

				// Edge attraction
				for (const edge of edges) {
					const source = nodeMap.get(edge.source);
					const target = nodeMap.get(edge.target);
					if (!source || !target) continue;

					const dx = target.x - source.x;
					const dy = target.y - source.y;
					const dist = Math.sqrt(dx * dx + dy * dy) || 1;
					const force = (dist - LINK_DISTANCE) * ATTRACTION;

					source.vx += (dx / dist) * force;
					source.vy += (dy / dist) * force;
					target.vx -= (dx / dist) * force;
					target.vy -= (dy / dist) * force;
				}

				// Apply velocity and damping
				let totalVelocity = 0;
				for (const node of newNodes) {
					node.vx *= DAMPING;
					node.vy *= DAMPING;
					node.x += node.vx;
					node.y += node.vy;
					totalVelocity += Math.abs(node.vx) + Math.abs(node.vy);
				}

				// Stop when settled
				if (totalVelocity < MIN_VELOCITY * newNodes.length) {
					running = false;
				}

				return newNodes;
			});

			animationFrame = requestAnimationFrame(simulate);
		}

		simulate();

		return () => {
			running = false;
			cancelAnimationFrame(animationFrame);
		};
	}, [nodes.length, edges]);

	function handleRefresh() {
		setRefreshing(true);
		fetchTickets();
	}

	const handleMouseDown = useCallback(
		(e: React.MouseEvent) => {
			if (e.target === e.currentTarget) {
				setIsDragging(true);
				setDragStart({ x: e.clientX - pan.x, y: e.clientY - pan.y });
			}
		},
		[pan],
	);

	const handleMouseMove = useCallback(
		(e: React.MouseEvent) => {
			if (isDragging) {
				setPan({
					x: e.clientX - dragStart.x,
					y: e.clientY - dragStart.y,
				});
			}
		},
		[isDragging, dragStart],
	);

	const handleMouseUp = useCallback(() => {
		setIsDragging(false);
	}, []);

	const handleWheel = useCallback((e: React.WheelEvent) => {
		e.preventDefault();
		setZoom((z) => Math.max(0.25, Math.min(2, z - e.deltaY * 0.001)));
	}, []);

	const handleNodeClick = useCallback(
		(nodeId: string) => {
			if (selectedNode === nodeId) {
				navigate(`/tickets/${nodeId}`);
			} else {
				setSelectedNode(nodeId);
			}
		},
		[selectedNode, navigate],
	);

	const viewBox = useMemo(() => {
		if (nodes.length === 0) return "0 0 800 600";
		const minX = Math.min(...nodes.map((n) => n.x)) - 100;
		const maxX = Math.max(...nodes.map((n) => n.x)) + 100;
		const minY = Math.min(...nodes.map((n) => n.y)) - 100;
		const maxY = Math.max(...nodes.map((n) => n.y)) + 100;
		return `${minX} ${minY} ${maxX - minX} ${maxY - minY}`;
	}, [nodes]);

	const getStatusFill = (status: string) => {
		switch (status) {
			case "ready":
				return "#22c55e";
			case "in_progress":
				return "#3b82f6";
			case "blocked":
				return "#f97316";
			case "human":
				return "#a855f7";
			case "review":
				return "#eab308";
			case "closed":
				return "#6b7280";
			default:
				return "#6b7280";
		}
	};

	if (loading) {
		return (
			<div className="flex items-center justify-center h-64">
				<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-[var(--primary)]" />
			</div>
		);
	}

	if (error) {
		return (
			<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 text-red-700 dark:text-red-300">
				{error}
			</div>
		);
	}

	return (
		<div className="space-y-4">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-4">
					<h2 className="text-2xl font-bold">Dependency Graph</h2>
					<span className="text-sm text-[var(--muted-foreground)]">
						{tickets.length} tickets • {edges.length} dependencies
					</span>
				</div>
				<div className="flex items-center gap-2">
					<button
						type="button"
						onClick={() => setZoom((z) => Math.min(2, z + 0.25))}
						className="p-2 rounded-md bg-[var(--secondary)] hover:bg-[var(--accent)] transition-colors"
						aria-label="Zoom in"
					>
						<ZoomIn className="w-4 h-4" />
					</button>
					<button
						type="button"
						onClick={() => setZoom((z) => Math.max(0.25, z - 0.25))}
						className="p-2 rounded-md bg-[var(--secondary)] hover:bg-[var(--accent)] transition-colors"
						aria-label="Zoom out"
					>
						<ZoomOut className="w-4 h-4" />
					</button>
					<button
						type="button"
						onClick={handleRefresh}
						disabled={refreshing}
						className="flex items-center gap-2 px-3 py-2 text-sm rounded-md bg-[var(--secondary)] hover:bg-[var(--accent)] transition-colors disabled:opacity-50"
					>
						<RefreshCw className={cn("w-4 h-4", refreshing && "animate-spin")} />
						Refresh
					</button>
				</div>
			</div>

			{/* Legend */}
			<div className="flex flex-wrap gap-4 text-sm">
				<span className="flex items-center gap-1">
					<span className="w-3 h-3 rounded-full bg-green-500" /> Ready
				</span>
				<span className="flex items-center gap-1">
					<span className="w-3 h-3 rounded-full bg-blue-500" /> In Progress
				</span>
				<span className="flex items-center gap-1">
					<span className="w-3 h-3 rounded-full bg-orange-500" /> Blocked
				</span>
				<span className="flex items-center gap-1">
					<span className="w-3 h-3 rounded-full bg-purple-500" /> Human
				</span>
				<span className="flex items-center gap-1">
					<span className="w-3 h-3 rounded-full bg-yellow-500" /> Review
				</span>
				<span className="flex items-center gap-1">
					<span className="w-3 h-3 rounded-full bg-gray-500" /> Closed
				</span>
			</div>

			{/* Graph */}
			<div
				role="application"
				aria-label="Dependency graph visualization"
				className="bg-[var(--card)] border border-[var(--border)] rounded-lg overflow-hidden"
				style={{ height: "calc(100vh - 280px)", cursor: isDragging ? "grabbing" : "grab" }}
				onMouseDown={handleMouseDown}
				onMouseMove={handleMouseMove}
				onMouseUp={handleMouseUp}
				onMouseLeave={handleMouseUp}
				onWheel={handleWheel}
			>
				<svg
					width="100%"
					height="100%"
					viewBox={viewBox}
					role="img"
					aria-label="Ticket dependency graph"
					style={{
						transform: `scale(${zoom}) translate(${pan.x / zoom}px, ${pan.y / zoom}px)`,
					}}
				>
					<title>Ticket Dependency Graph</title>
					{/* Edges */}
					<g>
						{edges.map((edge) => {
							const source = nodes.find((n) => n.id === edge.source);
							const target = nodes.find((n) => n.id === edge.target);
							if (!source || !target) return null;

							return (
								<line
									key={`${edge.source}-${edge.target}`}
									x1={source.x}
									y1={source.y}
									x2={target.x}
									y2={target.y}
									stroke="var(--border)"
									strokeWidth={2}
									markerEnd="url(#arrowhead)"
								/>
							);
						})}
					</g>

					{/* Arrow marker */}
					<defs>
						<marker
							id="arrowhead"
							markerWidth="10"
							markerHeight="7"
							refX="10"
							refY="3.5"
							orient="auto"
						>
							<polygon points="0 0, 10 3.5, 0 7" fill="var(--muted-foreground)" />
						</marker>
					</defs>

					{/* Nodes */}
					<g>
						{nodes.map((node) => (
							// biome-ignore lint/a11y/noStaticElementInteractions: SVG g elements need click handlers for interactivity
							<g
								key={node.id}
								transform={`translate(${node.x}, ${node.y})`}
								onClick={() => handleNodeClick(node.id)}
								onKeyDown={(e) => {
									if (e.key === "Enter" || e.key === " ") {
										handleNodeClick(node.id);
									}
								}}
								onMouseEnter={() => setHoveredNode(node.id)}
								onMouseLeave={() => setHoveredNode(null)}
								style={{ cursor: "pointer" }}
								tabIndex={0}
							>
								<circle
									r={NODE_RADIUS}
									fill={getStatusFill(node.ticket.status)}
									stroke={selectedNode === node.id ? "var(--primary)" : "var(--border)"}
									strokeWidth={selectedNode === node.id ? 4 : 2}
									opacity={hoveredNode && hoveredNode !== node.id ? 0.5 : 1}
								/>
								<text
									textAnchor="middle"
									dy="4"
									className="fill-white text-xs font-medium pointer-events-none"
								>
									{node.ticket.number}
								</text>
								{(hoveredNode === node.id || selectedNode === node.id) && (
									<g transform={`translate(0, ${NODE_RADIUS + 15})`}>
										<rect
											x={-75}
											y={-12}
											width={150}
											height={24}
											rx={4}
											fill="var(--card)"
											stroke="var(--border)"
										/>
										<text
											textAnchor="middle"
											dy="4"
											className="fill-[var(--foreground)] text-xs pointer-events-none"
										>
											{node.id}
										</text>
									</g>
								)}
							</g>
						))}
					</g>
				</svg>
			</div>

			{/* Help text */}
			<p className="text-sm text-[var(--muted-foreground)]">
				Click a node to select, click again to view details. Scroll to zoom, drag to pan.
			</p>
		</div>
	);
}
