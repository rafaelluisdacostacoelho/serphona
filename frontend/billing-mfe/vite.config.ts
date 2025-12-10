import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import federation from '@originjs/vite-plugin-federation';

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: 'billing',
      filename: 'remoteEntry.js',
      exposes: {
        './PricingPlans': './src/components/PricingPlans',
        './WalletCard': './src/components/WalletCard',
        './CheckoutButton': './src/components/CheckoutButton',
        './BillingPortal': './src/components/BillingPortal',
        './useBilling': './src/hooks/useBilling',
        './useWallet': './src/hooks/useWallet',
      },
      shared: ['react', 'react-dom'],
    }),
  ],
  build: {
    modulePreload: false,
    target: 'esnext',
    minify: false,
    cssCodeSplit: false,
  },
  server: {
    port: 3001,
    cors: true,
  },
});
