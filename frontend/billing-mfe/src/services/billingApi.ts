import axios from 'axios';
import type { Plan, Subscription, Wallet, WalletTransaction, CheckoutSession, BillingPortalSession } from '../types/billing';

const API_BASE_URL = import.meta.env.VITE_BILLING_API_URL || 'http://localhost:8081';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add auth token interceptor
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('authToken');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export const billingApi = {
  // Plans
  async getPlans(): Promise<Plan[]> {
    const { data } = await api.get('/api/v1/plans');
    return data;
  },

  // Subscriptions
  async getSubscription(): Promise<Subscription> {
    const { data } = await api.get('/api/v1/subscriptions/current');
    return data;
  },

  async cancelSubscription(): Promise<void> {
    await api.delete('/api/v1/subscriptions/current');
  },

  // Wallet
  async getWallet(): Promise<Wallet> {
    const { data} = await api.get('/api/v1/wallet');
    return data;
  },

  async getWalletTransactions(limit = 10): Promise<WalletTransaction[]> {
    const { data } = await api.get('/api/v1/wallet/transactions', { params: { limit } });
    return data;
  },

  async topUpWallet(amount: number): Promise<CheckoutSession> {
    const { data } = await api.post('/api/v1/wallet/topup', { amount });
    return data;
  },

  // Checkout
  async createCheckoutSession(planId: string): Promise<CheckoutSession> {
    const { data } = await api.post('/api/v1/checkout-session', { planId });
    return data;
  },

  // Billing Portal
  async createBillingPortalSession(): Promise<BillingPortalSession> {
    const { data } = await api.post('/api/v1/portal-session');
    return data;
  },
};
