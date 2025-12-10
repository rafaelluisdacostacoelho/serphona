import React from 'react';
import { useBilling } from '../hooks/useBilling';
import type { Plan } from '../types/billing';

interface PricingPlansProps {
  onSelectPlan?: (plan: Plan) => void;
}

export const PricingPlans: React.FC<PricingPlansProps> = ({ onSelectPlan }) => {
  const { plans, isLoading, createCheckout } = useBilling();

  const handleSelectPlan = async (plan: Plan) => {
    if (onSelectPlan) {
      onSelectPlan(plan);
    } else {
      const session = await createCheckout(plan.id);
      window.location.href = session.url;
    }
  };

  if (isLoading) return <div className="text-center py-8">Loading plans...</div>;

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      {plans?.map((plan) => (
        <div
          key={plan.id}
          className={`rounded-lg border-2 p-6 ${
            plan.popular ? 'border-blue-500 shadow-lg' : 'border-gray-200'
          }`}
        >
          {plan.popular && (
            <span className="bg-blue-500 text-white px-3 py-1 rounded-full text-sm">
              Popular
            </span>
          )}
          <h3 className="text-2xl font-bold mt-4">{plan.name}</h3>
          <p className="text-gray-600 mt-2">{plan.description}</p>
          <div className="mt-4">
            <span className="text-4xl font-bold">${plan.price}</span>
            <span className="text-gray-600">/{plan.interval}</span>
          </div>
          <div className="mt-6 text-sm text-gray-600">
            {plan.creditsPerMonth} credits/month
          </div>
          <ul className="mt-6 space-y-3">
            {plan.features.map((feature, idx) => (
              <li key={idx} className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mr-2" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                </svg>
                {feature}
              </li>
            ))}
          </ul>
          <button
            onClick={() => handleSelectPlan(plan)}
            className={`w-full mt-8 py-3 px-6 rounded-lg font-semibold ${
              plan.popular
                ? 'bg-blue-500 text-white hover:bg-blue-600'
                : 'bg-gray-100 text-gray-900 hover:bg-gray-200'
            }`}
          >
            Choose Plan
          </button>
        </div>
      ))}
    </div>
  );
};
