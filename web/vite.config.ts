import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [
    svelte({
      preprocess: vitePreprocess(),
    }),
  ],
  server: {
    host: '0.0.0.0',
    port: 5173,
    allowedHosts: ['wwwill-1-t'],
    proxy: {
      '/files':   { target: 'http://127.0.0.1:8888', changeOrigin: true },
      '/dirs':    { target: 'http://127.0.0.1:8888', changeOrigin: true },
      '/peers':   { target: 'http://127.0.0.1:8888', changeOrigin: true },
      '/health':  { target: 'http://127.0.0.1:8888', changeOrigin: true },
      '/search':  { target: 'http://127.0.0.1:8888', changeOrigin: true },
    },
  },
  build: {
    outDir: 'build',
    sourcemap: true
  },
});