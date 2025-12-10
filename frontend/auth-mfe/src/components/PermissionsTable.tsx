import React from 'react';
import { usePermissions } from '../hooks/usePermissions';

export const PermissionsTable: React.FC = () => {
  const { tenantUsers } = usePermissions();
  return <div>Permissions Table - {tenantUsers?.length || 0} users</div>;
};
