export interface Plan {
  id: string;
  name: string;
  description: string;
  price: number;
  currency: string;
  interval: 'month' | 'year';
  features: string[];
  creditsPerMonth: number;
  stripePriceId: string;
  popular?: boolean;
}

export interface Subscription {
  id: string;
  tenantId: string;
  planId: string;
  status: 'active' | 'canceled' | 'past_due' | 'trialing';
  currentPeriodStart: string;
  currentPeriodEnd: string;
  cancelAtPeriodEnd: boolean;
  stripeSubscriptionId: string;
}

export interface Wallet {
  id: string;
  tenantId: string;
  balance: number;
  currency: string;
  isLocked: boolean;
}

export interface WalletTransaction {
  id: string;
  walletId: string;
  amount: number;
  type: 'credit' | 'debit';
  description: string;
  reference?: string;
  createdAt: string;
}

export interface CheckoutSession {
  url: string;
  sessionId: string;
}

export interface BillingPortalSession {
  url: string;
}
