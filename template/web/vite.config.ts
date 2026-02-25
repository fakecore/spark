import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import * as path from 'path';
import { defineConfig, loadEnv } from 'vite';
import svgr from 'vite-plugin-svgr';

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const envConfig = loadEnv(mode, './');

  return {
    base: envConfig.VITE_PUBLIC_PATH || '/',
    mode: envConfig.VITE_NODE_ENV,
    plugins: [react(), svgr(), tailwindcss()],
    server: {
      open: true,
      port: 5173,
      host: '0.0.0.0',
      proxy: {
        '^/api/v1': {
          target: 'http://127.0.0.1:9988',
          changeOrigin: true,
          ws: true,
        },
      },
    },
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    define: {
      'process.env': envConfig,
    },
  };
});
