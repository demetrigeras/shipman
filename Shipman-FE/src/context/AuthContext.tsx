import { createContext, useContext, useState, useEffect } from 'react';
import type { ReactNode } from 'react';
import { api, ApiError } from '../api/client';
import type { User, SignupData, SigninData } from '../api/client';

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  signup: (data: SignupData) => Promise<void>;
  signin: (data: SigninData) => Promise<void>;
  signout: () => void;
  error: string | null;
  clearError: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (token) {
      api.auth.me()
        .then(setUser)
        .catch(() => {
          localStorage.removeItem('token');
        })
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, []);

  const signup = async (data: SignupData) => {
    setError(null);
    try {
      const response = await api.auth.signup(data);
      localStorage.setItem('token', response.token);
      setUser(response.user);
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Signup failed';
      setError(message);
      throw e;
    }
  };

  const signin = async (data: SigninData) => {
    setError(null);
    try {
      const response = await api.auth.signin(data);
      localStorage.setItem('token', response.token);
      setUser(response.user);
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Sign in failed';
      setError(message);
      throw e;
    }
  };

  const signout = () => {
    localStorage.removeItem('token');
    setUser(null);
  };

  const clearError = () => setError(null);

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        signup,
        signin,
        signout,
        error,
        clearError,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
