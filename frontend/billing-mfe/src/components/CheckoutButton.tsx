import React from 'react';
import { useBilling } from '../hooks/useBilling';

interface CheckoutButtonProps {
  planId: string;
  children?: React.ReactNode;
  className?: string;
}

export const CheckoutButton: React.FC<CheckoutButtonProps> = ({ 
  planId, 
  children = 'Subscribe', 
  className = 'bg-blue-500 text-white py-2 px-6 rounded-lg hover:bg-blue-600' 
}) => {
  const { createCheckout } = useBilling();
  const [loading, setLoading] = React.useState(false);

  const handleClick = async () => {
    setLoading(true);
    try {
      const session = await createCheckout(planId);
      window.location.href = session.url;
    } catch (error) {
      console.error('Checkout error:', error);
      setLoading(false);
    }
  };

  return (
    <button onClick={handleClick} disabled={loading} className={className}>
      {loading ? 'Processing...' : children}
    </button>
  );
};
