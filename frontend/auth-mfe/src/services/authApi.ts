import axios from 'axios';
import type { User, LoginCredentials, RegisterData, AuthResponse, Role, Permission, TenantUser } from '../types/auth';

const API_BASE_URL = import.meta.env.VITE_AUTH_API_URL || 'http://localhost:8080';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('accessToken');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

export const authApi = {
  async login(credentials: LoginCredentials): Promise<AuthResponse> {
    const { data } = await api.post('/auth/login', credentials);
    return data;
  },

  async register(data: RegisterData): Promise<AuthResponse> {
    const { data: response } = await api.post('/auth/register', data);
    return response;
  },

  async logout(): Promise<void> {
    await api.post('/auth/logout');
  },

  async getMe(): Promise<User> {
    const { data } = await api.get('/auth/me');
    return data;
  },

  async refreshToken(): Promise<AuthResponse> {
    const { data } = await api.post('/auth/refresh');
    return data;
  },

  async getRoles(): Promise<Role[]> {
    const { data } = await api.get('/roles');
    return data;
  },

  async getPermissions(): Promise<Permission[]> {
    const { data } = await api.get('/permissions');
    return data;
  },

  async getTenantUsers(): Promise<TenantUser[]> {
    const { data } = await api.get('/tenant/users');
    return data;
  },

  async inviteUser(email: string, roleId: string): Promise<void> {
    await api.post('/tenant/invite', { email, roleId });
  },

  async updateUserRole(userId: string, roleId: string): Promise<void> {
    await api.put(`/tenant/users/${userId}/role`, { roleId });
  },
};
