import React from 'react';
import { useAuth } from '../hooks/useAuth';

export const RegisterForm: React.FC = () => {
  const { register } = useAuth();
  return <div>Register Form - To be implemented</div>;
};
