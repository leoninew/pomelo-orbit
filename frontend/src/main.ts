import { createPinia } from 'pinia';
import { createApp } from 'vue';
import './style.css';
import '@vue-flow/core/dist/style.css';
import '@vue-flow/core/dist/theme-default.css';
import '@vue-flow/controls/dist/style.css';
import '@vue-flow/minimap/dist/style.css';
import App from './App.vue';
import router from './router';
import i18n from './i18n';
import { install as VueMonacoEditorPlugin } from '@guolao/vue-monaco-editor';

const app = createApp(App);
app.use(createPinia());
app.use(router);
app.use(i18n);
app.use(VueMonacoEditorPlugin, {
	paths: {
		vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.55.1/min/vs',
	},
});
app.mount('#app');
