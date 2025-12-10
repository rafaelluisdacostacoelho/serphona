import React from 'react';
import { useBilling } from '../hooks/useBilling';

interface BillingPortalProps {
  children?: React.ReactNode;
  className?: string;
}

export const BillingPortal: React.FC<BillingPortalProps> = ({ 
  children = 'Manage Billing', 
  className = 'text-blue-500 hover:text-blue-600 underline' 
}) => {
  const { createPortal } = useBilling();
  const [loading, setLoading] = React.useState(false);

  const handleClick = async () => {
    setLoading(true);
    try {
      const session = await createPortal();
      window.location.href = session.url;
    } catch (error) {
      console.error('Portal error:', error);
      setLoading(false);
    }
  };

  return (
    <button onClick={handleClick} disabled={loading} className={className}>
      {loading ? 'Opening...' : children}
    </button>
  );
};
