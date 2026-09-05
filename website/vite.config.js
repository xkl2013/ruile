import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";

const rootDir = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  build: {
    rollupOptions: {
      input: {
        index: resolve(rootDir, "index.html"),
        privacy: resolve(rootDir, "privacy.html"),
      },
    },
  },
});
