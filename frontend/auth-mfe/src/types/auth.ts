export interface User {
  id: string;
  email: string;
  name: string;
  tenantId: string;
  role: string;
  permissions: string[];
  createdAt: string;
}

export interface LoginCredentials {
  email: string;
  password: string;
}

export interface RegisterData {
  email: string;
  password: string;
  name: string;
  tenantName?: string;
}

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

export interface AuthResponse {
  user: User;
  tokens: AuthTokens;
}

export interface Permission {
  id: string;
  name: string;
  description: string;
  resource: string;
  action: string;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  permissions: Permission[];
  tenantId: string;
}

export interface TenantUser {
  id: string;
  userId: string;
  tenantId: string;
  role: Role;
  user: User;
  invitedBy: string;
  invitedAt: string;
  status: 'pending' | 'active' | 'inactive';
}
