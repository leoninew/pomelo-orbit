import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import js from '@eslint/js';
import pluginVue from 'eslint-plugin-vue';
import tseslint from 'typescript-eslint';
import eslintConfigPrettier from 'eslint-config-prettier';
import { createNodeResolver, importX } from 'eslint-plugin-import-x';
import { createTypeScriptImportResolver } from 'eslint-import-resolver-typescript';
import globals from 'globals';

const __dirname = dirname(fileURLToPath(import.meta.url));
const srcPath = resolve(__dirname, 'src');

export default [
  {
    ignores: ['dist', 'node_modules', 'coverage', '*.config.js', '*.config.ts'],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...pluginVue.configs['flat/recommended'],
  importX.flatConfigs.recommended,
  importX.flatConfigs.typescript,
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
      },
    },
  },
  {
    languageOptions: {
      globals: globals.browser,
    },
    settings: {
      'import-x/resolver-next': [
        createTypeScriptImportResolver({
          alwaysTryTypes: true,
          project: './tsconfig.json',
        }),
        createNodeResolver({
          alias: { '@': [srcPath] },
          extensions: ['.ts', '.tsx', '.vue', '.js', '.jsx', '.json'],
        }),
      ],
      'import-x/core-modules': ['vue', 'vue-router', 'vue-i18n', 'lucide-vue-next'],
    },
    rules: {
      'import-x/no-unresolved': 'error',
      // Vue 规则
      'vue/multi-word-component-names': 'off',
      'vue/no-v-html': 'warn',
      'vue/require-default-prop': 'off',
      'vue/require-explicit-emits': 'warn',
      'vue/component-definition-name-casing': ['error', 'PascalCase'],
      'vue/custom-event-name-casing': ['error', 'kebab-case', { ignores: ['update:modelValue'] }],
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

      // 通用规则
      'no-console': ['warn', { allow: ['error'] }],
      'no-debugger': 'warn',
      'prefer-const': 'warn',
      'no-var': 'error',
      curly: ['error', 'all'],
    },
  },
  eslintConfigPrettier,
];
