# Wark UI Redesign Plan

> **Theme:** Simple, industrial-inspired mission control for AI agents — understated, functional, information-dense.
>
> **NO:** Purple gradients, glowing effects, neural network imagery, or tech-bro aesthetics.
>
> **Reference Apps:** Linear (clean, fast, keyboard-driven), Jira (information density, status visibility)
>
> **User:** Human operator monitoring AI agent work. This is a **read-only dashboard** — agents use the CLI, not the UI.

---

## Table of Contents

1. [Current State Audit](#1-current-state-audit)
2. [Design System](#2-design-system)
3. [Component Inventory](#3-component-inventory)
4. [Icon Strategy](#4-icon-strategy)
5. [Layout Architecture](#5-layout-architecture)
6. [Per-View Redesign Specs](#6-per-view-redesign-specs)
7. [Interaction Patterns](#7-interaction-patterns)

---

## 1. Current State Audit

### 1.1 Dashboard (`/`)

**Current Implementation:**
- Header with logo, horizontal nav, search bar, settings icon
- 4 status cards in a grid (Workable, In Progress, Blocked, Pending Inbox)
- "Claims Expiring Soon" section (list)
- "Recent Activity" section (list with action badges)

**Problems Identified:**
- ❌ Status cards feel generic — no sense of priority or urgency
- ❌ "Workable" is an unusual term (should be "Ready" for consistency with board)
- ❌ Large numbers (0) dominate when there's nothing to show — feels empty
- ❌ Activity feed action badges have inconsistent styling (plain gray background)
- ❌ No visual hierarchy between critical and informational items
- ❌ Refresh button looks like a navigation item, not an action
- ❌ Missing quick access to critical states (what needs attention NOW?)

### 1.2 Projects (`/projects`)

**Current Implementation:**
- Grid of project cards (3 columns on desktop)
- Each card shows: icon, key, name, description, stats (Open, Ready, Active)

**Problems Identified:**
- ❌ Project cards are too tall for the content they hold
- ❌ Stats row uses colored labels (Ready:, Active:) — inconsistent with rest of UI
- ❌ Empty state shows generic icon — could be more informative
- ❌ No indication of project health at a glance
- ❌ Cards don't show recent activity or momentum

### 1.3 Tickets (`/tickets`)

**Current Implementation:**
- Sortable table with columns: Key, Title, Status, Priority, Complexity, Created
- Status shown as colored pill badges
- Priority shown as colored text
- Sort indicators on column headers

**Problems Identified:**
- ❌ Table feels cramped — rows too tight
- ❌ Status pills use different color scheme than board columns
- ❌ No row hover preview or quick actions
- ❌ Created date column takes space but isn't always useful
- ❌ No bulk selection or batch operations (even view-only filters)
- ❌ Missing project column when viewing all tickets
- ❌ Long titles get truncated with no way to see full text

### 1.4 Board (`/board`)

**Current Implementation:**
- Filter bar (Project, Priority, Complexity dropdowns)
- 5 Kanban columns: Ready (green), In Progress (blue), Human (purple), Review (yellow), Closed (gray)
- Ticket cards show: key, priority badge, title, branch name
- Columns have colored top borders

**Problems Identified:**
- ❌ Column colors are too saturated — distracting
- ❌ Closed column takes equal space but has "View all" link for overflow
- ❌ Human flag reason shown inline in small purple text — easy to miss
- ❌ Branch name shown for all tickets — clutters cards without active work
- ❌ No swim lanes for grouping (by project, by priority, by assignee)
- ❌ Filter bar takes too much vertical space
- ❌ No collapse/expand for columns
- ❌ Priority badge colors clash with column colors

### 1.5 Inbox (`/inbox`)

**Current Implementation:**
- List of message cards
- Each card: type badge, ticket link, content, response input
- Type-specific icons (Question, Decision, Review, Escalation, Info)
- Empty state: "No pending messages"

**Problems Identified:**
- ❌ Response input shown for ALL messages — should be read-only view
- ❌ Pending badge in header is purple — matches Human column, confusing
- ❌ Empty state is too minimal — should reinforce that inbox is under control
- ❌ Message type hierarchy unclear — escalations should stand out more
- ❌ No grouping by urgency or ticket

### 1.6 Analytics (`/analytics`)

**Current Implementation:**
- Multiple metric sections: Success Metrics, Human Interaction, Throughput, WIP, Cycle Time, Completion Trend
- Metric cards with large numbers
- Tables for WIP and Cycle Time
- Bar chart for completion trend (using Recharts)

**Problems Identified:**
- ❌ Currently shows API error (endpoint not implemented or broken)
- ❌ Too many metrics shown at once — overwhelming
- ❌ No clear story: "Is my agent fleet healthy?"
- ❌ Metric cards all look the same — no visual hierarchy
- ❌ Chart is cramped at bottom of page
- ❌ No time range selector visible (beyond trend_days param)

### 1.7 Ticket Detail (`/tickets/:key`)

**Current Implementation:**
- Back button, ticket key, status/priority badges, title
- Action buttons row (Accept, Reject, Close — depends on state)
- 2-column layout: Description + Activity (left), Details sidebar (right)
- Activity shows timeline with actor icons

**Problems Identified:**
- ❌ Action buttons should be REMOVED — this is a read-only dashboard
- ❌ Status badges using text colors without backgrounds — hard to scan
- ❌ Activity timeline icons are generic circles — could show action type
- ❌ Details sidebar wastes space — most fields are empty
- ❌ Dependencies section only shows when populated — layout jumps
- ❌ Human flag reason buried in sidebar — should be prominent if present
- ❌ No way to navigate between tickets (prev/next)

### 1.8 Global Issues

- ❌ No dark mode toggle — users must rely on system preference
- ❌ Settings icon (gear) has no function
- ❌ Search only searches tickets — no global search
- ❌ No keyboard shortcuts visible (Cmd+K hint shown but no cheat sheet)
- ❌ No breadcrumbs on detail pages
- ❌ Loading states are simple spinners — should be skeleton loaders
- ❌ Error states are generic red boxes — could be more helpful
- ❌ No favicon or brand identity

---

## 2. Design System

### 2.1 Color Palette

Using oklch for better perceptual uniformity. Colors chosen for industrial, functional feel — no hype, just clarity.

#### Light Mode

```css
:root {
  /* Background layers */
  --background: oklch(0.985 0 0);           /* Near-white, warm */
  --background-subtle: oklch(0.97 0 0);     /* Panels, cards */
  --background-muted: oklch(0.94 0 0);      /* Hover states, wells */
  
  /* Foreground */
  --foreground: oklch(0.20 0 0);            /* Primary text */
  --foreground-muted: oklch(0.45 0 0);      /* Secondary text */
  --foreground-subtle: oklch(0.60 0 0);     /* Tertiary, captions */
  
  /* Borders */
  --border: oklch(0.90 0 0);                /* Default borders */
  --border-muted: oklch(0.93 0 0);          /* Subtle dividers */
  --border-strong: oklch(0.80 0 0);         /* Focus rings */
  
  /* Status Colors — Muted, Industrial */
  --status-ready: oklch(0.65 0.15 145);     /* Muted green */
  --status-in-progress: oklch(0.60 0.12 250); /* Slate blue */
  --status-human: oklch(0.55 0.14 30);      /* Muted amber/rust */
  --status-review: oklch(0.62 0.10 85);     /* Muted gold */
  --status-blocked: oklch(0.50 0 0);        /* Neutral gray */
  --status-closed: oklch(0.70 0 0);         /* Light gray */
  
  /* Priority Colors */
  --priority-highest: oklch(0.55 0.18 25);  /* Deep rust */
  --priority-high: oklch(0.60 0.14 45);     /* Amber */
  --priority-medium: oklch(0.55 0.08 70);   /* Olive */
  --priority-low: oklch(0.55 0.08 250);     /* Slate */
  --priority-lowest: oklch(0.60 0 0);       /* Gray */
  
  /* Interactive */
  --accent: oklch(0.55 0.10 250);           /* Slate blue for links/buttons */
  --accent-hover: oklch(0.50 0.12 250);
  --accent-muted: oklch(0.65 0.05 250);
  
  /* Feedback */
  --success: oklch(0.60 0.15 150);
  --warning: oklch(0.65 0.15 80);
  --error: oklch(0.55 0.18 25);
  --info: oklch(0.60 0.10 250);
}
```

#### Dark Mode

```css
.dark {
  /* Background layers */
  --background: oklch(0.14 0 0);            /* Near-black */
  --background-subtle: oklch(0.18 0 0);     /* Panels, cards */
  --background-muted: oklch(0.22 0 0);      /* Hover states */
  
  /* Foreground */
  --foreground: oklch(0.92 0 0);            /* Primary text */
  --foreground-muted: oklch(0.65 0 0);      /* Secondary text */
  --foreground-subtle: oklch(0.50 0 0);     /* Tertiary */
  
  /* Borders */
  --border: oklch(0.25 0 0);
  --border-muted: oklch(0.20 0 0);
  --border-strong: oklch(0.35 0 0);
  
  /* Status Colors — Slightly brighter for dark mode */
  --status-ready: oklch(0.70 0.14 145);
  --status-in-progress: oklch(0.65 0.12 250);
  --status-human: oklch(0.65 0.14 30);
  --status-review: oklch(0.68 0.10 85);
  --status-blocked: oklch(0.55 0 0);
  --status-closed: oklch(0.45 0 0);
  
  /* Priority Colors */
  --priority-highest: oklch(0.65 0.16 25);
  --priority-high: oklch(0.68 0.13 45);
  --priority-medium: oklch(0.62 0.07 70);
  --priority-low: oklch(0.60 0.07 250);
  --priority-lowest: oklch(0.50 0 0);
  
  /* Interactive */
  --accent: oklch(0.65 0.10 250);
  --accent-hover: oklch(0.70 0.12 250);
  --accent-muted: oklch(0.45 0.05 250);
}
```

### 2.2 Typography Scale

System font stack for reliability and native feel:

```css
:root {
  --font-sans: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 
    "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  --font-mono: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, 
    "Liberation Mono", monospace;
  
  /* Type scale (1.2 ratio, minor third) */
  --text-xs: 0.694rem;    /* 11.1px - Captions, badges */
  --text-sm: 0.833rem;    /* 13.3px - Secondary text, labels */
  --text-base: 1rem;      /* 16px - Body text */
  --text-lg: 1.2rem;      /* 19.2px - Section headers */
  --text-xl: 1.44rem;     /* 23px - Page titles */
  --text-2xl: 1.728rem;   /* 27.6px - Hero numbers */
  
  /* Line heights */
  --leading-tight: 1.25;
  --leading-normal: 1.5;
  --leading-relaxed: 1.625;
  
  /* Font weights */
  --font-normal: 400;
  --font-medium: 500;
  --font-semibold: 600;
  --font-bold: 700;
  
  /* Letter spacing */
  --tracking-tight: -0.02em;
  --tracking-normal: 0;
  --tracking-wide: 0.02em;
}
```

### 2.3 Spacing System

4px base unit, following Tailwind conventions:

```css
:root {
  --space-0: 0;
  --space-1: 0.25rem;   /* 4px */
  --space-2: 0.5rem;    /* 8px */
  --space-3: 0.75rem;   /* 12px */
  --space-4: 1rem;      /* 16px */
  --space-5: 1.25rem;   /* 20px */
  --space-6: 1.5rem;    /* 24px */
  --space-8: 2rem;      /* 32px */
  --space-10: 2.5rem;   /* 40px */
  --space-12: 3rem;     /* 48px */
  --space-16: 4rem;     /* 64px */
}
```

### 2.4 Shadows & Depth

Minimal shadows — use borders and subtle background shifts instead:

```css
:root {
  --shadow-sm: 0 1px 2px oklch(0 0 0 / 0.04);
  --shadow-md: 0 2px 4px oklch(0 0 0 / 0.06);
  --shadow-lg: 0 4px 8px oklch(0 0 0 / 0.08);
  
  /* For dropdowns and popovers only */
  --shadow-popover: 0 4px 16px oklch(0 0 0 / 0.12);
}

.dark {
  --shadow-sm: 0 1px 2px oklch(0 0 0 / 0.2);
  --shadow-md: 0 2px 4px oklch(0 0 0 / 0.3);
  --shadow-lg: 0 4px 8px oklch(0 0 0 / 0.4);
  --shadow-popover: 0 4px 16px oklch(0 0 0 / 0.5);
}
```

### 2.5 Border Radius

```css
:root {
  --radius-sm: 0.25rem;   /* 4px - Badges, small buttons */
  --radius-md: 0.375rem;  /* 6px - Default cards, inputs */
  --radius-lg: 0.5rem;    /* 8px - Modals, large cards */
  --radius-full: 9999px;  /* Pills */
}
```

---

## 3. Component Inventory

### 3.1 shadcn/ui Components to Use

| Component | Use Case | Notes |
|-----------|----------|-------|
| `Button` | Refresh, theme toggle | Ghost/outline variants only |
| `Badge` | Status, priority, action types | Custom colors per status |
| `Card` | Stat cards, project cards, ticket cards | Minimal border style |
| `Table` | Tickets list | With sticky header |
| `Select` | Filters (project, priority, complexity) | Native-like styling |
| `Input` | Search | With icon prefix |
| `Separator` | Section dividers | Subtle horizontal rules |
| `Tooltip` | Truncated text, icon buttons | Delay 300ms |
| `DropdownMenu` | Overflow menus if needed | Minimal use |
| `Skeleton` | Loading states | All views |
| `ScrollArea` | Kanban columns, long lists | Custom scrollbar |
| `Tabs` | Analytics sections (optional) | Underline style |
| `Sheet` | Mobile nav (responsive) | Slide from left |

### 3.2 Custom Components Needed

| Component | Description |
|-----------|-------------|
| `StatusBadge` | Unified status indicator with icon + text |
| `PriorityIndicator` | Compact priority display (dot or text) |
| `TicketKey` | Monospace project-number link |
| `StatCard` | Metric with label, value, optional change |
| `ActivityItem` | Timeline item with icon, action, summary |
| `KanbanColumn` | Column container with header + scrollable body |
| `KanbanCard` | Compact ticket card for board |
| `EmptyState` | Consistent empty/zero state display |
| `ErrorBoundary` | Graceful error handling |
| `PageHeader` | Title + actions + optional description |
| `NavItem` | Navigation link with icon and active state |
| `SearchCommand` | Command palette-style search (Cmd+K) |
| `ThemeToggle` | Light/dark/system switcher |
| `BranchLink` | Git branch name with copy |

---

## 4. Icon Strategy

### 4.1 Lucide Icons to Use

**Navigation:**
- `LayoutDashboard` — Dashboard (replacing Home for clearer purpose)
- `FolderKanban` — Projects
- `ListTodo` — Tickets (list view)
- `KanbanSquare` — Board (kanban view)
- `Inbox` — Inbox
- `BarChart3` — Analytics
- `Settings` — Settings/preferences

**Status:**
- `CircleCheck` — Ready / Completed
- `CircleDot` — In Progress / Active
- `UserRound` — Human (needs human attention)
- `Eye` — Review
- `CircleMinus` — Blocked
- `CircleX` — Closed (not completed)

**Actions:**
- `RefreshCw` — Refresh
- `Search` — Search
- `Filter` — Filters
- `ArrowUpDown` — Sort (unsorted)
- `ArrowUp` — Sort ascending
- `ArrowDown` — Sort descending
- `ArrowLeft` — Back navigation
- `ExternalLink` — External links
- `Copy` — Copy to clipboard
- `MoreHorizontal` — Overflow menu

**Message Types:**
- `HelpCircle` — Question
- `Scale` — Decision
- `FileSearch` — Review
- `AlertTriangle` — Escalation
- `Info` — Info

**Misc:**
- `GitBranch` — Branch name
- `Clock` — Time/duration
- `Calendar` — Date
- `Activity` — Activity feed
- `AlertCircle` — Warning/attention
- `CheckCircle2` — Success
- `XCircle` — Error
- `Sun` — Light mode
- `Moon` — Dark mode
- `Monitor` — System theme

### 4.2 Icon Sizing

- Navigation: 18px (`w-[18px] h-[18px]`)
- Inline with text: 14px (`w-3.5 h-3.5`)
- Status badges: 12px (`w-3 h-3`)
- Hero icons (empty states): 48px (`w-12 h-12`)

### 4.3 Icon Color Rules

- Navigation icons: `text-foreground-muted`, `text-foreground` when active
- Status icons: Match status color
- Action icons: `text-foreground-muted`, `text-foreground` on hover
- Decorative icons: `text-foreground-subtle`

---

## 5. Layout Architecture

### 5.1 Shell Structure

```
┌─────────────────────────────────────────────────────────┐
│ Header (fixed, h-12)                                    │
│ ┌─────┬─────────────────────┬─────────────────────────┐ │
│ │Logo │ Navigation          │ Search │ Theme │ Status │ │
│ └─────┴─────────────────────┴─────────────────────────┘ │
├─────────────────────────────────────────────────────────┤
│ Main Content (flex-1, scrollable)                       │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Page Header (sticky within scroll)                  │ │
│ ├─────────────────────────────────────────────────────┤ │
│ │ Page Content                                        │ │
│ │                                                     │ │
│ │                                                     │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### 5.2 Header Layout (Redesigned)

```
┌────────────────────────────────────────────────────────────────┐
│ wark    Dashboard  Projects  Tickets  Board  Inbox  Analytics  │
│ [logo]  ────────────────────────────────────────────────────── │
│         [nav items as text buttons with underline active state]│
│                                                                │
│                              [Search (Cmd+K)]  [🌙]  [⚙]       │
└────────────────────────────────────────────────────────────────┘
```

- Logo: Bold "wark" text, links to dashboard
- Nav: Horizontal tabs with underline active indicator (Linear-style)
- Right side: Search input, theme toggle, settings (if needed)

### 5.3 Responsive Breakpoints

```css
/* Tailwind breakpoints */
sm: 640px   /* Stack cards, collapse nav */
md: 768px   /* 2-column layouts */
lg: 1024px  /* 3-column layouts, full nav */
xl: 1280px  /* Max content width */
2xl: 1536px /* Extra wide content for boards */
```

### 5.4 Max Content Width

- Dashboard, Tickets, Analytics: `max-w-6xl` (1152px)
- Board: `max-w-none` (full width for columns)
- Ticket Detail: `max-w-5xl` (1024px)

---

## 6. Per-View Redesign Specs

### 6.1 Dashboard (`/`)

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Dashboard                              [↻ Last updated] │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐        │
│ │ Ready   │ │ Active  │ │ Blocked │ │ Inbox   │        │
│ │ 12      │ │ 3       │ │ 2       │ │ 5       │        │
│ └─────────┘ └─────────┘ └─────────┘ └─────────┘        │
│                                                         │
│ ┌───────────────────────────────────────────────────┐  │
│ │ Needs Attention                                   │  │
│ │ ─────────────────────────────────────────────────│  │
│ │ 🔔 WARK-42 blocked: Dependency on WARK-38       │  │
│ │ 👤 WARK-45 needs human decision                 │  │
│ │ ⏰ WARK-31 claim expiring in 12m                │  │
│ └───────────────────────────────────────────────────┘  │
│                                                         │
│ ┌───────────────────────────────────────────────────┐  │
│ │ Recent Activity                                   │  │
│ │ ─────────────────────────────────────────────────│  │
│ │ WARK-35  completed   Auto-accepted          1m   │  │
│ │ WARK-35  claimed     By agent-1            2m   │  │
│ │ WARK-35  created     Ticket created        30m  │  │
│ └───────────────────────────────────────────────────┘  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Changes:**
1. Rename "Workable" → "Ready" for consistency
2. Stat cards: Smaller, more compact. Show trend arrow if available (↑2 from yesterday)
3. NEW "Needs Attention" section consolidating:
   - Claims expiring soon (yellow warning)
   - Blocked tickets (gray)
   - Human-flagged tickets (amber)
4. Activity feed: Streamlined, no explicit "Refresh" button (auto-refresh is on)
5. Timestamp format: "1m" instead of "just now" for consistency

**Components:**
- `StatCard` × 4
- `AttentionList` (new component)
- `ActivityFeed` (simplified)

### 6.2 Projects (`/projects`)

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Projects                                                │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ ┌──────────────────────┐ ┌──────────────────────┐      │
│ │ WARK                 │ │ ACME                 │      │
│ │ Wark Development     │ │ Acme Corp Project    │      │
│ │ ─────────────────────│ │ ─────────────────────│      │
│ │ Ready 2 · Active 1   │ │ Ready 5 · Active 0   │      │
│ │ ████████░░ 80%       │ │ ██████████ 100%      │      │
│ └──────────────────────┘ └──────────────────────┘      │
│                                                         │
│ ┌──────────────────────┐                               │
│ │ TEST                 │                               │
│ │ Test Project         │                               │
│ │ ─────────────────────│                               │
│ │ Ready 0 · Active 0   │                               │
│ │ (no open tickets)    │                               │
│ └──────────────────────┘                               │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Changes:**
1. Project cards: More compact, single line stats
2. Health indicator: Simple progress bar showing % complete (closed / total)
3. Remove description preview (move to hover tooltip if needed)
4. 2-column grid on md, 3-column on lg+
5. Show "(no open tickets)" for empty projects instead of zeros

**Components:**
- `ProjectCard` (redesigned)
- `ProgressBar` (simple horizontal bar)

### 6.3 Tickets (`/tickets`)

**Layout:**
```
┌─────────────────────────────────────────────────────────────────┐
│ Tickets                    [Filter: All] [Status: All] [Reset] │
├─────────────────────────────────────────────────────────────────┤
│ Key      Title                         Status    Pri   Created  │
│ ─────────────────────────────────────────────────────────────── │
│ WARK-35  Add unique constraint...      ● closed  med   Feb 2    │
│ WARK-34  Remove redundant project...   ● closed  med   Feb 2    │
│ WARK-33  Add skill install command...  ● closed  med   Feb 2    │
│ WARK-32  Remove action buttons...      ○ review  med   Feb 2    │
│ WARK-31  Log claim releases...         ○ review  high  Feb 2    │
│ ─────────────────────────────────────────────────────────────── │
│                                            Showing 35 tickets   │
└─────────────────────────────────────────────────────────────────┘
```

**Changes:**
1. Remove "Complexity" column — low value, clutters table
2. Status: Dot indicator + text, not pill badge (cleaner)
3. Priority: Abbreviated (high, med, low) with subtle color
4. Filters: Inline pill-style filters instead of dropdowns
5. Row hover: Slight background highlight
6. Add project column when no project filter is active
7. Sticky header when scrolling
8. Footer with count

**Components:**
- `DataTable` (using shadcn Table)
- `StatusDot` (small colored dot)
- `FilterPills` (toggle-style filters)

### 6.4 Board (`/board`)

**Layout:**
```
┌───────────────────────────────────────────────────────────────────────┐
│ Board       Project: [All ▾]  Priority: [All ▾]  [× Clear]           │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│ ┌─ Ready (2)─┐ ┌─ Active (0)─┐ ┌─ Human (1)──┐ ┌─ Review (2)─┐ ┌─ Closed ─┐
│ │            │ │             │ │             │ │             │ │          │
│ │ ┌────────┐ │ │             │ │ ┌────────┐  │ │ ┌────────┐  │ │ ● 33     │
│ │ │WARK-36 │ │ │ (no tickets)│ │ │WARK-40 │  │ │ │WARK-31 │  │ │ ● 34     │
│ │ │Title...│ │ │             │ │ │Title...│  │ │ │Title...│  │ │ ● 35     │
│ │ │high    │ │ │             │ │ │⚠ needs │  │ │ │high    │  │ │ ...      │
│ │ └────────┘ │ │             │ │ │ decision│  │ │ └────────┘  │ │ +29 more │
│ │            │ │             │ │ └────────┘  │ │             │ │          │
│ │ ┌────────┐ │ │             │ │             │ │ ┌────────┐  │ │          │
│ │ │WARK-37 │ │ │             │ │             │ │ │WARK-32 │  │ │          │
│ │ │Title...│ │ │             │ │             │ │ │Title...│  │ │          │
│ │ │medium  │ │ │             │ │             │ │ │medium  │  │ │          │
│ │ └────────┘ │ │             │ │             │ │ └────────┘  │ │          │
│ │            │ │             │ │             │ │             │ │          │
│ └────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ └──────────┘
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

**Changes:**
1. Column headers: Softer colors, just tinted top border (not full saturated stripe)
2. Remove "Blocked" as separate column (blocked tickets show in Ready with blocked badge)
3. Closed column: Compact list view (just key + dot), expandable
4. Ticket cards: Minimal — key, title (2 lines max), priority dot
5. Human flag: Prominent warning icon + reason text on card
6. Branch name: Hidden by default, show on hover or in detail
7. Empty columns: "(no tickets)" placeholder, dimmed
8. Filters: Inline selects, same row as title
9. Auto-refresh indicator in header (pulsing dot when refreshing)

**Components:**
- `KanbanColumn` (with collapse option)
- `KanbanCard` (redesigned, minimal)
- `CompactClosedList` (for closed column)
- `FilterSelect` (styled native select)

### 6.5 Inbox (`/inbox`)

**Layout:**
```
┌─────────────────────────────────────────────────────────────────┐
│ Inbox                                        2 pending          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ ┌───────────────────────────────────────────────────────────┐  │
│ │ 🔺 Escalation                     WARK-45 · 2h ago        │  │
│ │ ─────────────────────────────────────────────────────────│  │
│ │ From: agent-claude-01                                    │  │
│ │                                                           │  │
│ │ Unable to proceed with implementation. The specification │  │
│ │ requires feature X but the codebase doesn't have...      │  │
│ └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│ ┌───────────────────────────────────────────────────────────┐  │
│ │ ? Question                        WARK-42 · 5h ago        │  │
│ │ ─────────────────────────────────────────────────────────│  │
│ │ From: agent-claude-02                                    │  │
│ │                                                           │  │
│ │ Should I use the existing API or create a new endpoint?  │  │
│ └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│ ┌───────────────────────────────────────────────────────────┐  │
│ │ ℹ Info                             WARK-38 · 1d ago       │  │
│ │ ─────────────────────────────────────────────────────────│  │
│ │ Work completed. Summary: Implemented the new feature...  │  │
│ │ ─────────────────────────────────────────────────────────│  │
│ │ ✓ Responded 22h ago                                      │  │
│ └───────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Changes:**
1. **REMOVE response input** — this is read-only! Responses happen via CLI
2. Visual hierarchy: Escalations > Questions > Decisions > Reviews > Info
3. Escalation cards: Amber/red left border, always at top
4. Responded messages: Collapsed by default, muted styling
5. Sort: By urgency (type) first, then by time
6. Agent name: Show "From: agent-name" for context
7. Pending count in header

**Components:**
- `InboxCard` (redesigned, no response input)
- `MessageTypeBadge` (icon + label)
- `ExpandableContent` (for long messages)

### 6.6 Analytics (`/analytics`)

**Layout:**
```
┌─────────────────────────────────────────────────────────────────┐
│ Analytics              [Project: All ▾]   [Last 30 days ▾]      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Fleet Health Score                                              │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │
│ │ 94.2%        │ │ 3.2h         │ │ 12%          │             │
│ │ Success Rate │ │ Avg Cycle    │ │ Human Rate   │             │
│ │ ↑2% vs last  │ │ ↓0.5h        │ │ ↓3%          │             │
│ └──────────────┘ └──────────────┘ └──────────────┘             │
│                                                                 │
│ Throughput                                                      │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│ ┌───────────────────────────────────────────────────────────┐  │
│ │ ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆▇█▇▆▅▄▃▂▁  (bar chart)              │  │
│ │ Jan 3  ────────────────────────────────────────  Feb 2    │  │
│ └───────────────────────────────────────────────────────────┘  │
│ Total: 142 completed │ Avg: 4.7/day                            │
│                                                                 │
│ Current Work                                                    │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│ Ready: 12  │  Active: 3  │  Review: 2  │  Human: 1             │
│                                                                 │
│ Cycle Time by Complexity                                        │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│ Trivial    ██░░░░░░░░   0.8h (23 tickets)                      │
│ Small      ████░░░░░░   2.1h (45 tickets)                      │
│ Medium     ██████░░░░   4.2h (38 tickets)                      │
│ Large      ████████░░   8.5h (12 tickets)                      │
│ X-Large    ██████████  12.3h (4 tickets)                       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Changes:**
1. Lead with "Fleet Health Score" — 3 key metrics at glance
2. Trend indicators: Show comparison to previous period (↑↓)
3. Throughput chart: Larger, more prominent
4. Cycle time: Horizontal bar chart instead of table (more visual)
5. WIP section: Simple inline stats, not a separate table
6. Remove or collapse less important metrics (retry rate, etc.)
7. Time range selector: "Last 7/30/90 days" + custom

**Components:**
- `MetricCard` (with trend indicator)
- `ThroughputChart` (Recharts bar)
- `CycleTimeChart` (horizontal bars)
- `PeriodSelect` (time range picker)

### 6.7 Ticket Detail (`/tickets/:key`)

**Layout:**
```
┌─────────────────────────────────────────────────────────────────┐
│ ← Tickets                                                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ WARK-31  ○ review  high                                        │
│ ────────────────────────────────────────────────────────────── │
│ Log claim releases in activity log                             │
│                                                                 │
│ ┌─────────────────────────────────────┐ ┌─────────────────────┐│
│ │ Description                         │ │ Details             ││
│ │                                     │ │                     ││
│ │ When a claim is released (via       │ │ Complexity: small   ││
│ │ `wark ticket release` or            │ │ Retries: 0/3        ││
│ │ expiration), it should be recorded  │ │ Created: Feb 2      ││
│ │ in the activity_log table...        │ │                     ││
│ │                                     │ │ Branch:             ││
│ │                                     │ │ wark/WARK-31-log... ││
│ │                                     │ │ [copy]              ││
│ │                                     │ │                     ││
│ │                                     │ │ Dependencies: none  ││
│ │                                     │ │ Dependents: none    ││
│ └─────────────────────────────────────┘ └─────────────────────┘│
│                                                                 │
│ Activity                                                        │
│ ────────────────────────────────────────────────────────────── │
│ ● completed   agent-01          Verified claim release...  1m  │
│ ● claimed     agent-01          Claimed (60m)              2m  │
│ ○ created     system            Ticket created            30m  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Changes:**
1. **REMOVE all action buttons** — read-only dashboard
2. Back link: "← Tickets" breadcrumb-style
3. Title block: Key, status dot, priority on one line
4. 2-column layout: Description (left, wider), Details (right, narrow)
5. Activity: Simplified timeline, action type as colored dot
6. Branch: Copy button for convenience
7. Dependencies/Dependents: Show as linked ticket keys
8. Human flag reason: If present, show as prominent banner below title

**Components:**
- `PageHeader` (with back navigation)
- `TicketMeta` (status, priority badges)
- `DetailsSidebar` (metadata panel)
- `ActivityTimeline` (simplified)
- `AlertBanner` (for human flag reason)

---

## 7. Interaction Patterns

### 7.1 Loading States

**Skeleton Loaders (not spinners):**
```tsx
// Stat card skeleton
<div className="animate-pulse">
  <div className="h-4 w-16 bg-background-muted rounded mb-2" />
  <div className="h-8 w-12 bg-background-muted rounded" />
</div>

// Table row skeleton
<tr className="animate-pulse">
  <td><div className="h-4 w-16 bg-background-muted rounded" /></td>
  <td><div className="h-4 w-48 bg-background-muted rounded" /></td>
  <td><div className="h-4 w-12 bg-background-muted rounded" /></td>
</tr>

// Kanban card skeleton
<div className="animate-pulse p-3 border border-border rounded">
  <div className="h-3 w-12 bg-background-muted rounded mb-2" />
  <div className="h-4 w-full bg-background-muted rounded" />
</div>
```

### 7.2 Empty States

**Consistent pattern for all views:**
```tsx
<div className="flex flex-col items-center justify-center py-16 text-foreground-muted">
  <Icon className="w-12 h-12 mb-4 text-foreground-subtle" />
  <p className="text-lg font-medium mb-1">{title}</p>
  <p className="text-sm text-foreground-subtle">{description}</p>
</div>
```

**Examples:**
- Inbox: "All clear" / "No pending messages. Agents are working independently."
- Tickets (filtered): "No matches" / "No tickets match the current filters."
- Board column: "(no tickets)" in muted text, centered

### 7.3 Error States

**API errors:**
```tsx
<div className="flex items-center gap-3 p-4 border border-error/20 bg-error/5 rounded-md text-error">
  <AlertCircle className="w-5 h-5 flex-shrink-0" />
  <div>
    <p className="font-medium">Failed to load data</p>
    <p className="text-sm text-error/80">{error.message}</p>
  </div>
  <Button variant="ghost" size="sm" onClick={retry}>Retry</Button>
</div>
```

### 7.4 Hover States

**Cards:**
```css
.card {
  transition: border-color 150ms ease;
}
.card:hover {
  border-color: var(--border-strong);
}
```

**Table rows:**
```css
tr {
  transition: background-color 100ms ease;
}
tr:hover {
  background-color: var(--background-muted);
}
```

**Links:**
```css
a {
  color: var(--accent);
  text-decoration: none;
}
a:hover {
  text-decoration: underline;
}
```

### 7.5 Focus States

**Keyboard navigation:**
```css
*:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

button:focus-visible {
  ring: 2px var(--accent);
}
```

### 7.6 Refresh Behavior

**Auto-refresh (silent):**
- Every 10 seconds when tab is visible
- No visual indicator during normal refresh
- Subtle fade-in animation when data changes

**Manual refresh:**
- Refresh icon spins while loading
- Data fades in when complete

**Stale data indicator:**
```tsx
// Show if data is >1 minute old and auto-refresh failed
<span className="text-xs text-foreground-subtle">
  Updated 3m ago · <button className="underline">Refresh</button>
</span>
```

### 7.7 Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `⌘K` / `/` | Focus search |
| `Escape` | Close search/modal, blur focus |
| `g d` | Go to Dashboard |
| `g p` | Go to Projects |
| `g t` | Go to Tickets |
| `g b` | Go to Board |
| `g i` | Go to Inbox |
| `g a` | Go to Analytics |
| `r` | Refresh current view |
| `←` | Go back (on detail pages) |
| `j` / `k` | Navigate list items (table, activity) |
| `Enter` | Open selected item |

**Shortcut hint:**
```tsx
// Show in footer or as tooltip on '?'
<div className="fixed bottom-4 right-4 text-xs text-foreground-subtle">
  Press <kbd>?</kbd> for keyboard shortcuts
</div>
```

### 7.8 Tooltips

**Truncated text:**
- Show full text on hover after 300ms delay
- Position: Prefer top, fallback to bottom

**Icon buttons:**
- Always show tooltip with label
- Delay: 200ms

**Implementation:**
```tsx
<Tooltip delayDuration={300}>
  <TooltipTrigger asChild>
    <span className="truncate">{longText}</span>
  </TooltipTrigger>
  <TooltipContent>{longText}</TooltipContent>
</Tooltip>
```

### 7.9 Dark Mode Toggle

**Location:** Header, right side (before settings if present)

**Behavior:**
```tsx
// Cycle: light → dark → system → light
const themes = ['light', 'dark', 'system'] as const;

// Store preference in localStorage
// Apply class to <html> element
// Icon changes based on current effective theme
```

**Icons:**
- Light mode: `Sun`
- Dark mode: `Moon`
- System: `Monitor`

---

## Implementation Priority

### Phase 1: Foundation (Critical)
1. Update CSS variables (new color palette)
2. Remove action buttons from all views (read-only)
3. Implement dark mode toggle
4. Update status/priority colors throughout
5. Replace spinners with skeleton loaders

### Phase 2: Key Views
1. Dashboard redesign (needs attention section)
2. Board view cleanup (simpler cards, closed column)
3. Tickets table polish (remove clutter, add filters)

### Phase 3: Detail & Polish
1. Ticket detail page (remove actions, clean layout)
2. Inbox redesign (no response input, visual hierarchy)
3. Analytics (when API is working)

### Phase 4: Refinements
1. Keyboard shortcuts
2. Animations and transitions
3. Empty states with personality
4. Error boundary improvements
5. Responsive tweaks

---

## File Structure Recommendation

```
ui/src/
├── components/
│   ├── ui/            # shadcn primitives
│   ├── layout/
│   │   ├── Header.tsx
│   │   ├── NavItem.tsx
│   │   └── PageHeader.tsx
│   ├── shared/
│   │   ├── StatusBadge.tsx
│   │   ├── PriorityIndicator.tsx
│   │   ├── TicketKey.tsx
│   │   ├── StatCard.tsx
│   │   ├── EmptyState.tsx
│   │   ├── ErrorState.tsx
│   │   ├── Skeleton/
│   │   │   ├── StatCardSkeleton.tsx
│   │   │   ├── TableRowSkeleton.tsx
│   │   │   └── KanbanCardSkeleton.tsx
│   │   └── ThemeToggle.tsx
│   ├── dashboard/
│   │   ├── AttentionList.tsx
│   │   └── ActivityFeed.tsx
│   ├── board/
│   │   ├── KanbanColumn.tsx
│   │   ├── KanbanCard.tsx
│   │   └── CompactClosedList.tsx
│   ├── tickets/
│   │   ├── TicketTable.tsx
│   │   ├── FilterPills.tsx
│   │   └── TicketRow.tsx
│   ├── inbox/
│   │   ├── InboxCard.tsx
│   │   └── MessageTypeBadge.tsx
│   ├── analytics/
│   │   ├── MetricCard.tsx
│   │   ├── ThroughputChart.tsx
│   │   └── CycleTimeChart.tsx
│   └── ticket-detail/
│       ├── DetailsSidebar.tsx
│       └── ActivityTimeline.tsx
├── styles/
│   └── index.css      # CSS variables, base styles
├── lib/
│   ├── utils.ts
│   ├── api.ts
│   ├── hooks.ts
│   └── theme.ts       # Theme management
└── views/
    ├── Dashboard.tsx
    ├── Projects.tsx
    ├── Tickets.tsx
    ├── Board.tsx
    ├── Inbox.tsx
    ├── Analytics.tsx
    └── TicketDetail.tsx
```

---

## Notes for Implementation Agent

1. **CSS Variables First**: Start by updating `index.css` with the new color palette. Test both light and dark modes.

2. **One View at a Time**: Don't try to refactor everything at once. Complete one view fully before moving to the next.

3. **Preserve Functionality**: The read-only nature means removing interactivity is additive (less code). Don't break existing data flows.

4. **Test Responsive**: Board view especially needs testing at different widths. Use Chrome DevTools to simulate.

5. **Commit Often**: Small, atomic commits for each component or feature.

6. **Use Existing shadcn**: Don't reinvent — use the shadcn components already available where possible.

7. **Performance**: Skeleton loaders should render immediately. Don't wait for any data before showing the shell.

8. **Accessibility**: All interactive elements need focus states. Color alone should not convey meaning (always pair with text/icon).
