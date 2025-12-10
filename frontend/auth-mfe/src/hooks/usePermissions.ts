import { useQuery } from '@tanstack/react-query';
import { authApi } from '../services/authApi';

export const usePermissions = () => {
  const { data: roles } = useQuery({
    queryKey: ['roles'],
    queryFn: authApi.getRoles,
  });

  const { data: permissions } = useQuery({
    queryKey: ['permissions'],
    queryFn: authApi.getPermissions,
  });

  const { data: tenantUsers } = useQuery({
    queryKey: ['tenantUsers'],
    queryFn: authApi.getTenantUsers,
  });

  return {
    roles,
    permissions,
    tenantUsers,
  };
};
