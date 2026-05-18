import { defineConfig } from 'vitest/config';

export default defineConfig({
  optimizeDeps: {
    exclude: ['firebase', '@firebase/auth', '@firebase/app']
  },
  test: {
    deps: {
      inline: ['firebase', '@firebase/auth', '@firebase/app']
    },
    server: {
      deps: {
        inline: ['firebase', '@firebase/auth', '@firebase/app']
      }
    }
  }
});
