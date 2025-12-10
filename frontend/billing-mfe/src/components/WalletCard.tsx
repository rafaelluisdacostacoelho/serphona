import React from 'react';
import { useWallet } from '../hooks/useWallet';

export const WalletCard: React.FC = () => {
  const { wallet, isLoading, topUp } = useWallet();

  const handleTopUp = async () => {
    const amount = prompt('Enter amount in cents (e.g., 1000 for $10):');
    if (amount) {
      const session = await topUp(parseInt(amount));
      window.location.href = session.url;
    }
  };

  if (isLoading) return <div>Loading wallet...</div>;

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <h3 className="text-lg font-semibold mb-4">Credit Wallet</h3>
      <div className="flex items-baseline mb-4">
        <span className="text-4xl font-bold">{(wallet?.balance || 0) / 100}</span>
        <span className="text-gray-600 ml-2">credits</span>
      </div>
      <button
        onClick={handleTopUp}
        className="w-full bg-blue-500 text-white py-2 px-4 rounded-lg hover:bg-blue-600"
      >
        Top Up Credits
      </button>
    </div>
  );
};
