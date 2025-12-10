import { useQuery, useMutation } from '@tanstack/react-query';
import { billingApi } from '../services/billingApi';

export const useBilling = () => {
  const { data: plans, isLoading } = useQuery({
    queryKey: ['plans'],
    queryFn: billingApi.getPlans,
  });

  const { data: subscription } = useQuery({
    queryKey: ['subscription'],
    queryFn: billingApi.getSubscription,
  });

  const createCheckoutMutation = useMutation({
    mutationFn: billingApi.createCheckoutSession,
  });

  const createPortalMutation = useMutation({
    mutationFn: billingApi.createBillingPortalSession,
  });

  return {
    plans,
    subscription,
    isLoading,
    createCheckout: createCheckoutMutation.mutateAsync,
    createPortal: createPortalMutation.mutateAsync,
  };
};
