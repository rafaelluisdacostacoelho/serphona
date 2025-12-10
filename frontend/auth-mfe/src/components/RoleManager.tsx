import React from 'react';
import { usePermissions } from '../hooks/usePermissions';

export const RoleManager: React.FC = () => {
  const { roles } = usePermissions();
  return <div>Role Manager - {roles?.length || 0} roles</div>;
};
