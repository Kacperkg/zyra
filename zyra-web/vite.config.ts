import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
export default defineConfig(({ mode }) => ({
  plugins: [
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    react(),
  ],
  server: {
    host: "127.0.0.1",
    proxy: {
      "/api":
        loadEnv(mode, ".", "").API_PROXY_TARGET || "http://127.0.0.1:8080",
    },
  },
}));
