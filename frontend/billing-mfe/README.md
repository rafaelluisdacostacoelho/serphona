# Billing Microfrontend

Microfrontend de billing da Serphona com Module Federation. Compartilha componentes de billing entre Console e Website.

## 🚀 Features

- **Module Federation** - Componentes compartilhados em runtime
- **Pricing Plans** - Exibição de planos de assinatura
- **Wallet Management** - Gerenciamento de créditos
- **Stripe Integration** - Checkout e Billing Portal
- **React Query** - Cache e state management
- **TypeScript** - Type-safe
- **Tailwind CSS** - Styling

## 📦 Componentes Exportados

- `PricingPlans` - Grade de planos com preços
- `WalletCard` - Card de saldo de créditos
- `CheckoutButton` - Botão de checkout Stripe
- `BillingPortal` - Link para Stripe Billing Portal

## 🛠️ Hooks Exportados

- `useBilling` - Gerenciamento de planos e subscriptions
- `useWallet` - Gerenciamento de wallet e transações

## 🏃 Quick Start

```bash
# Install dependencies
npm install

# Start dev server
npm run dev

# Build for production
npm run build
```

## 🔧 Configuration

Create `.env` file:

```bash
VITE_BILLING_API_URL=http://localhost:8081
VITE_STRIPE_PUBLISHABLE_KEY=pk_test_...
```

## 📖 Usage in Host App

### 1. Configure Module Federation

```typescript
// vite.config.ts do console ou website
import federation from '@originjs/vite-plugin-federation';

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: 'console', // ou 'website'
      remotes: {
        billing: 'http://localhost:3001/assets/remoteEntry.js'
      },
      shared: ['react', 'react-dom']
    })
  ]
});
```

### 2. Add Type Declarations

```typescript
// src/types/billing-mfe.d.ts
declare module 'billing/PricingPlans' {
  export const PricingPlans: React.FC<any>;
}

declare module 'billing/WalletCard' {
  export const WalletCard: React.FC;
}

declare module 'billing/CheckoutButton' {
  export const CheckoutButton: React.FC<{
    planId: string;
    children?: React.ReactNode;
    className?: string;
  }>;
}

declare module 'billing/BillingPortal' {
  export const BillingPortal: React.FC<{
    children?: React.ReactNode;
    className?: string;
  }>;
}

declare module 'billing/useBilling' {
  export const useBilling: () => any;
}

declare module 'billing/useWallet' {
  export const useWallet: () => any;
}
```

### 3. Use Components

```typescript
import { PricingPlans } from 'billing/PricingPlans';
import { WalletCard } from 'billing/WalletCard';
import { CheckoutButton } from 'billing/CheckoutButton';
import { BillingPortal } from 'billing/BillingPortal';

function MyComponent() {
  return (
    <div>
      <WalletCard />
      <PricingPlans />
      <CheckoutButton planId="plan_123" />
      <BillingPortal />
    </div>
  );
}
```

### 4. Wrap with QueryClientProvider

```typescript
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <MyComponent />
    </QueryClientProvider>
  );
}
```

## 🎨 Styling

Components use Tailwind CSS. Make sure Tailwind is configured in your host app:

```javascript
// tailwind.config.js
module.exports = {
  content: [
    './src/**/*.{js,jsx,ts,tsx}',
    // Add if using remote components
    './node_modules/@serphona/billing-mfe/**/*.{js,jsx,ts,tsx}'
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
```

## 🔌 API Integration

The billing MFE connects to the billing-service backend:

```
GET  /api/v1/plans - List plans
GET  /api/v1/subscriptions/current - Get subscription
GET  /api/v1/wallet - Get wallet
GET  /api/v1/wallet/transactions - Get transactions
POST /api/v1/checkout-session - Create checkout
POST /api/v1/portal-session - Create portal session
POST /api/v1/wallet/topup - Top up wallet
```

## 📁 Project Structure

```
billing-mfe/
├── src/
│   ├── components/        # React components
│   │   ├── PricingPlans.tsx
│   │   ├── WalletCard.tsx
│   │   ├── CheckoutButton.tsx
│   │   └── BillingPortal.tsx
│   ├── hooks/            # Custom hooks
│   │   ├── useBilling.ts
│   │   └── useWallet.ts
│   ├── services/         # API services
│   │   └── billingApi.ts
│   └── types/            # TypeScript types
│       └── billing.ts
├── package.json
├── vite.config.ts        # Module Federation config
└── tsconfig.json
```

## 🚢 Deployment

### Build

```bash
npm run build
```

### Serve

Serve the `dist/` directory:

```bash
npm run preview
```

Or deploy to CDN/Static hosting:

- Vercel
- Netlify
- AWS S3 + CloudFront
- Azure Static Web Apps

Update remote URL in host apps to production URL.

## 🧪 Testing

```bash
# Run tests (to be implemented)
npm test
```

## 📝 Development Tips

1. **Run billing-mfe first** before starting host apps
2. **Keep billing-mfe running** while developing
3. **Hot reload** works with Module Federation
4. **Check CORS** if having connection issues
5. **Use same React version** in all apps

## 🔗 Related

- [Billing Service Backend](../../backend/go/services/billing-service)
- [Console Frontend](../console)
- [Website Frontend](../website)

## 📄 License

Private - Serphona Platform
