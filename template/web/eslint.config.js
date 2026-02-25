import react from '@eslint-react/eslint-plugin';
import prettier from 'eslint-config-prettier';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import globals from 'globals';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  // 忽略目录
  { ignores: ['dist/', 'node_modules/'] },

  // 全局设置
  {
    linterOptions: {
      reportUnusedDisableDirectives: 'error',
    },
  },

  // TypeScript + React
  ...tseslint.configs.recommended,
  {
    files: ['src/**/*.{ts,tsx}'],
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    languageOptions: {
      globals: {
        ...globals.browser,
      },
      parserOptions: {
        ecmaFeatures: { jsx: true },
      },
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
    },
  },

  // @eslint-react 推荐规则
  {
    files: ['src/**/*.{ts,tsx}'],
    ...react.configs['recommended-typescript'],
  },

  // 模板工具文件放宽类型约束
  {
    files: ['src/utils/oss.ts', 'src/types/common.ts'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },

  // Prettier 放最后（关闭冲突规则）
  prettier,
);
