import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [svelte()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    allowedHosts: ['wwwill-1-t'],
    proxy: {
      '/files':  { target: 'http://127.0.0.1:8888', changeOrigin: true },
      '/health': { target: 'http://127.0.0.1:8888', changeOrigin: true }
    }
  },
  build: {
    outDir: 'build'
  }
});