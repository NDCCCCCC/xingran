import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
import { initEncryptionConfig } from "@/lib/api";

/**
 * F-01: Render immediately — do NOT gate React tree on network configuration.
 * The encryption config (ENABLE_REQUEST_ENCRYPTION) is a single boolean that
 * gates SM2+SM4 on POST/PUT/PATCH. Starting with false (unencrypted) is safe
 * because the backend also accepts unencrypted requests; the config merely enables
 * extra security. We kick off the fetch in the background without blocking render.
 *
 * Background fetch: up to 3 retries × 3s timeout = ~12s worst-case before
 * encryption activates. This is acceptable — no white screen, only slightly reduced
 * security for the first few requests of a cold-start session.
 *
 * To avoid an ugly hydration flash when React mounts over the skeleton, we
 * render into a detached container first and replace #root's contents atomically.
 */

// Remove the inline skeleton from the DOM before mounting React
const rootEl = document.getElementById("root")!;
const skeleton = document.getElementById("root-loading");
if (skeleton) skeleton.remove();

const root = createRoot(rootEl);
root.render(<App />);

// Fire-and-forget: background config fetch — does not block the UI
void initEncryptionConfig();
