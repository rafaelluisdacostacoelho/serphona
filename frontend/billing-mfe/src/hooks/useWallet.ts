import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { billingApi } from '../services/billingApi';

export const useWallet = () => {
  const queryClient = useQueryClient();

  const { data: wallet, isLoading } = useQuery({
    queryKey: ['wallet'],
    queryFn: billingApi.getWallet,
  });

  const { data: transactions } = useQuery({
    queryKey: ['wallet-transactions'],
    queryFn: () => billingApi.getWalletTransactions(10),
  });

  const topUpMutation = useMutation({
    mutationFn: billingApi.topUpWallet,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wallet'] });
    },
  });

  return {
    wallet,
    transactions,
    isLoading,
    topUp: topUpMutation.mutateAsync,
  };
};
