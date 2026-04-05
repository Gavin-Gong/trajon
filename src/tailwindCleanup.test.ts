// @ts-ignore Node-only test helper import; project tsconfig excludes node typings.
import { readFileSync } from "node:fs";
// @ts-ignore Node-only test helper import; project tsconfig excludes node typings.
import path from "node:path";
import { describe, expect, it } from "vitest";

declare const process: {
  cwd(): string;
};

function readProjectFile(relativePath: string) {
  return readFileSync(path.join(process.cwd(), relativePath), "utf8");
}

const appCss = readProjectFile("src/App.css");
const viteConfig = readProjectFile("vite.config.ts");
const packageJson = readProjectFile("package.json");
const packageLock = readProjectFile("package-lock.json");

describe("Tailwind cleanup", () => {
  it("removes Tailwind from app styles and Vite plugin wiring", () => {
    expect(appCss).not.toContain('@import "tailwindcss"');
    expect(viteConfig).not.toContain("@tailwindcss/vite");
    expect(viteConfig).not.toContain("tailwindcss()");
    expect(appCss).toContain("--app-surface-strong");
    expect(appCss).toContain(".ant-input-outlined");
  });

  it("removes Tailwind packages from npm manifests", () => {
    expect(packageJson).not.toContain("tailwindcss");
    expect(packageJson).not.toContain("@tailwindcss/vite");
    expect(packageLock).not.toContain("tailwindcss");
    expect(packageLock).not.toContain("@tailwindcss/vite");
  });
});
