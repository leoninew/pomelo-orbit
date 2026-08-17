import { fileURLToPath, URL } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [tailwindcss(), vue()],
	server: {
		port: 9020,
		watch: {
			ignored: ["**/src/gen/proto/**"],
		},
		proxy: {
			"/api": {
				target: "http://127.0.0.1:9021",
				changeOrigin: true,
			},
			"/hooks": {
				target: "http://127.0.0.1:9021",
				changeOrigin: true,
			},
		},
	},
	resolve: {
		alias: {
			"@": fileURLToPath(new URL("./src", import.meta.url)),
		},
	},
	optimizeDeps: {
		include: ["monaco-editor"],
	},
	build: {
		rolldownOptions: {
			checks: {
				invalidAnnotation: false,
			},
		},
		rollupOptions: {
			output: {
				manualChunks: {
					monaco: ["monaco-editor"],
				},
			},
		},
	},
});
