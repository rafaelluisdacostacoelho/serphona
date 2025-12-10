import React from 'react';
import { useAuth } from '../hooks/useAuth';

export const UserProfile: React.FC = () => {
  const { user, logout } = useAuth();
  if (!user) return null;
  return (
    <div className="p-4 border rounded">
      <h3>{user.name}</h3>
      <p>{user.email}</p>
      <button onClick={logout} className="mt-2 bg-red-500 text-white px-4 py-2 rounded">Logout</button>
    </div>
  );
};
