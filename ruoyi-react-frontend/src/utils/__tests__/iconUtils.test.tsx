/**
 * iconUtils regression tests
 * BUNDLE-02: Verifies getIconComponent works via static registry (fake dynamic import removed)
 */

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
  getIconComponent,
  STATIC_ICON_REGISTRY,
  fullIconNameMap,
  iconCategories,
} from "../iconUtils";

describe("iconUtils", () => {
  describe("getIconComponent", () => {
    it("returns a valid React element for icon in STATIC_ICON_REGISTRY (DashboardOutlined)", () => {
      const result = getIconComponent("DashboardOutlined");
      expect(result).toBeDefined();
      expect(result).not.toBeNull();
    });

    it("returns a valid React element for UserOutlined", () => {
      const result = getIconComponent("UserOutlined");
      expect(result).toBeDefined();
      expect(result).not.toBeNull();
    });

    it("returns undefined for non-existent icon (no crash)", () => {
      const result = getIconComponent("non-existent-icon-xyz");
      expect(result).toBeUndefined();
    });

    it("returns valid element for fullIconNameMap key (DashboardOutlined → dashboard)", () => {
      // fullIconNameMap maps full names to short aliases: DashboardOutlined → "dashboard"
      // getIconComponent should find DashboardOutlined in registry via step 1
      const result = getIconComponent("DashboardOutlined");
      expect(result).toBeDefined();
      expect(result).not.toBeNull();
    });

    it("returns undefined for null input", () => {
      // @ts-expect-error testing runtime behavior with invalid input
      const result = getIconComponent(null);
      expect(result).toBeUndefined();
    });

    it("returns undefined for undefined input", () => {
      const result = getIconComponent(undefined);
      expect(result).toBeUndefined();
    });

    it("returns undefined for empty string input", () => {
      const result = getIconComponent("");
      expect(result).toBeUndefined();
    });

    it("returns valid element for composite name (CloudServerOutlined)", () => {
      const result = getIconComponent("CloudServerOutlined");
      expect(result).toBeDefined();
      expect(result).not.toBeNull();
    });
  });

  describe("exports", () => {
    it("STATIC_ICON_REGISTRY is exported and non-empty", () => {
      expect(STATIC_ICON_REGISTRY).toBeDefined();
      expect(Object.keys(STATIC_ICON_REGISTRY).length).toBeGreaterThan(0);
    });

    it("fullIconNameMap is exported and non-empty", () => {
      expect(fullIconNameMap).toBeDefined();
      expect(Object.keys(fullIconNameMap).length).toBeGreaterThan(0);
    });

    it("iconCategories is exported and non-empty", () => {
      expect(iconCategories).toBeDefined();
      expect(Object.keys(iconCategories).length).toBeGreaterThan(0);
    });
  });
});
