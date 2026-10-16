/**
 * HTML entity escaping utilities for security-sensitive contexts.
 * Use these when interpolating user-controlled strings into HTML content.
 */

/**
 * Escape HTML special characters to prevent XSS attacks when
 * interpolating untrusted strings into innerHTML/textContent.
 *
 * Escapes: & < > " '
 */
export const escapeHtml = (s: string): string =>
  s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
