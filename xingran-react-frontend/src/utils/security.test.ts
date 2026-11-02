import { escapeHtml } from "./security";

describe("escapeHtml", () => {
  it("escapes ampersand", () => {
    expect(escapeHtml("A & B")).toBe("A &amp; B");
  });

  it("escapes less-than", () => {
    expect(escapeHtml("a < b")).toBe("a &lt; b");
  });

  it("escapes greater-than", () => {
    expect(escapeHtml("a > b")).toBe("a &gt; b");
  });

  it("escapes double quote", () => {
    expect(escapeHtml('say "hello"')).toBe("say &quot;hello&quot;");
  });

  it("escapes single quote", () => {
    expect(escapeHtml("it's")).toBe("it&#39;s");
  });

  it("neutralises script injection attempt", () => {
    expect(escapeHtml("<script>alert(1)</script>")).toBe("&lt;script&gt;alert(1)&lt;/script&gt;");
  });

  it("escapes img onerror XSS vector", () => {
    expect(escapeHtml("<img src=x onerror=alert(1)>")).toBe("&lt;img src=x onerror=alert(1)&gt;");
  });

  it("handles empty string", () => {
    expect(escapeHtml("")).toBe("");
  });

  it("passes through plain ASCII text unchanged (entities not triggered)", () => {
    expect(escapeHtml("湖北省武汉市")).toBe("湖北省武汉市");
    expect(escapeHtml("Building A")).toBe("Building A");
  });
});
