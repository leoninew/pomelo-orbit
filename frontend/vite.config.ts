import { resolve } from "node:path";
import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";
import Components from "unplugin-vue-components/vite";
import { AntDesignVueResolver } from "unplugin-vue-components/resolvers";

export default defineConfig({
	plugins: [
		vue(),
		Components({
			resolvers: [
				AntDesignVueResolver({
					importStyle: false,
					resolveIcons: true
				}),
			],
		}),
	],
	server: {
		port: 9002,
		proxy: {
			"/api": {
				target: "http://127.0.0.1:9001",
				changeOrigin: true,
			},
			"/hooks": {
				target: "http://127.0.0.1:9001",
				changeOrigin: true,
			},
		},
	},
	resolve: {
		alias: {
			"@": resolve(__dirname, "src"),
		},
	}
});
