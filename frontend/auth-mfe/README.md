# Auth Microfrontend

Auth MFE com Module Federation - Autenticação, Autorização e Permissionamento.

## 🚀 Features

- **Auth**entication (Login/Register)
- **Authorization** (Roles & Permissions)
- **User Management** (Tenant users)
- **Module Federation** (porta 3002)
- **AuthContext Provider**

## 📦 Exports

**Components:**
- `LoginForm` - Login form
- `RegisterForm` - Registration form
- `UserProfile` - User profile card
- `PermissionsTable` - Permissions/users table
- `RoleManager` - Role management

**Context:**
- `AuthProvider` - Auth context provider

**Hooks:**
- `useAuth` - Auth state & actions
- `usePermissions` - Roles & permissions

## 🏃 Quick Start

```bash
npm install
npm run dev  # Port 3002
```

## 🔧 Config

`.env`:
```
VITE_AUTH_API_URL=http://localhost:8080
```

## 📖 Usage in Host

```typescript
// vite.config.ts
federation({
  remotes: {
    auth: 'http://localhost:3002/assets/remoteEntry.js'
  }
})

// App.tsx
import { AuthProvider } from 'auth/AuthProvider';
import { LoginForm } from 'auth/LoginForm';

<AuthProvider>
  <LoginForm />
</AuthProvider>
```

## 🔗 API Endpoints

- POST /auth/login
- POST /auth/register
- GET /auth/me
- GET /roles
- GET /permissions
- GET /tenant/users

Private - Serphona Platform
