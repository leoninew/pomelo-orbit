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
			'vue/html-indent': ['error', 'tab'],
			'vue/script-indent': ['error', 'tab', { baseIndent: 1 }],

			// TypeScript 规则
			'@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
			'@typescript-eslint/no-explicit-any': 'error',
			'@typescript-eslint/no-non-null-assertion': 'warn',
			'@typescript-eslint/no-empty-function': 'off',
			'@typescript-eslint/no-require-imports': 'off',

			// 代码风格规则 - interface 成员不使用分号
			'@stylistic/member-delimiter-style': [
				'error',
				{
					multiline: { delimiter: 'none', requireLast: false },
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
