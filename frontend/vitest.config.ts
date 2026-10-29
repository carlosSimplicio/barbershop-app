import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "server-only": new URL(
        "./node_modules/next/dist/compiled/server-only/empty.js",
        import.meta.url,
      ).pathname,
      "@": new URL(".", import.meta.url).pathname,
    },
  },
  test: {
    environment: "jsdom",
    pool: "vmThreads",
    setupFiles: ["./vitest.setup.ts"],
  },
});
