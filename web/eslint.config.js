import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'
import eslintConfigPrettier from 'eslint-config-prettier'
import stylistic from '@stylistic/eslint-plugin'

export default [
	{
		ignores: ['dist', 'node_modules', 'coverage', '*.config.js', '*.config.ts'],
	},
	js.configs.recommended,
	...tseslint.configs.recommended,
	...pluginVue.configs['flat/recommended'],
	eslintConfigPrettier,
	{
		files: ['**/*.vue'],
		languageOptions: {
			parserOptions: {
				parser: tseslint.parser,
			},
		},
	},
	{
		plugins: {
			'@stylistic': stylistic,
		},
		rules: {
			// Vue 规则
			'vue/multi-word-component-names': 'off',
			'vue/no-v-html': 'warn',
			'vue/require-default-prop': 'off',
			'vue/require-explicit-emits': 'warn',
			'vue/component-definition-name-casing': ['error', 'PascalCase'],
			'vue/custom-event-name-casing': ['error', 'kebab-case'],
			'vue/no-unused-refs': 'warn',
			'vue/block-order': ['error', { order: ['template', 'script', 'style'] }],
			// 缩进由 Prettier 统一处理，避免 lint:fix 与 format 来回改动
			'vue/html-indent': 'off',
			'vue/script-indent': 'off',

			// TypeScript 规则
			'@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
			'@typescript-eslint/no-explicit-any': 'error',
			'@typescript-eslint/no-non-null-assertion': 'warn',
			'@typescript-eslint/no-empty-function': 'off',
			'@typescript-eslint/no-require-imports': 'off',

			// 代码风格规则 - interface 成员使用分号，与 Prettier 默认风格保持一致
			'@stylistic/member-delimiter-style': [
				'error',
				{
					multiline: { delimiter: 'semi', requireLast: true },
					singleline: { delimiter: 'semi', requireLast: false },
				},
			],

			// 通用规则
			'no-console': ['warn', { allow: ['error'] }],
			'no-debugger': 'warn',
			'prefer-const': 'warn',
			'no-var': 'error',
			'curly': ['error', 'all'],
		},
	},
]
