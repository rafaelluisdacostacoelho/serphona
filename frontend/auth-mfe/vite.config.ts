import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import federation from '@originjs/vite-plugin-federation';

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: 'auth',
      filename: 'remoteEntry.js',
      exposes: {
        './LoginForm': './src/components/LoginForm',
        './RegisterForm': './src/components/RegisterForm',
        './UserProfile': './src/components/UserProfile',
        './PermissionsTable': './src/components/PermissionsTable',
        './RoleManager': './src/components/RoleManager',
        './AuthProvider': './src/context/AuthContext',
        './useAuth': './src/hooks/useAuth',
        './usePermissions': './src/hooks/usePermissions',
      },
      shared: ['react', 'react-dom'],
    }),
  ],
  test: {
    environment: 'jsdom',
    setupFiles: './src/setupTests.ts',
    globals: true,
  },
  build: {
    modulePreload: false,
    target: 'esnext',
    minify: false,
    cssCodeSplit: false,
  },
  server: {
    port: 3002,
    cors: true,
  },
});
