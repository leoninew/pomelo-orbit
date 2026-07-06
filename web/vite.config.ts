import { resolve } from "node:path";
import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [tailwindcss(), vue()],
	server: {
		port: 9020,
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
			"@": resolve(__dirname, "src"),
		},
	},
	optimizeDeps: {
		include: ["monaco-editor"],
	},
	build: {
		rollupOptions: {
			output: {
				manualChunks: {
					monaco: ["monaco-editor"],
				},
			},
		},
	},
});
