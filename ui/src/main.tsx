import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import "./index.css";
import App from "./App.tsx";
import { KeyboardShortcutsProvider } from "./components/KeyboardShortcutsProvider.tsx";

const rootElement = document.getElementById("root");
if (!rootElement) throw new Error("Root element not found");

createRoot(rootElement).render(
	<StrictMode>
		<BrowserRouter>
			<KeyboardShortcutsProvider>
				<App />
			</KeyboardShortcutsProvider>
		</BrowserRouter>
	</StrictMode>,
);
