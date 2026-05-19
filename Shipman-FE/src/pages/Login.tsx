import { useState, useRef, useEffect } from 'react';
import type { FormEvent } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export default function Login() {
  const [searchParams] = useSearchParams();
  const prefillEmail = searchParams.get('email') ?? '';
  const redirect = searchParams.get('redirect') ?? '/dashboard';

  const [email, setEmail] = useState(prefillEmail);
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { signin, error, clearError } = useAuth();
  const navigate = useNavigate();
  const passwordRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (prefillEmail) passwordRef.current?.focus();
  }, [prefillEmail]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await signin({ email, password });
      navigate(redirect);
    } catch {
      // Error is handled by context
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="auth-container">
      <div className="auth-card">
        <div className="auth-header">
          <h1>Shipman</h1>
          <p>Sign in to your account</p>
        </div>

        {error && (
          <div className="error-banner" onClick={clearError}>
            {error}
          </div>
        )}

        {prefillEmail && (
          <div className="invite-email-hint">
            Signing in as <strong>{prefillEmail}</strong> to join a negotiation
          </div>
        )}

        <form onSubmit={handleSubmit} className="auth-form">
          <div className="form-group">
            <label htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              required
              autoComplete="email"
            />
          </div>

          <div className="form-group">
            <label htmlFor="password">Password</label>
            <input
              ref={passwordRef}
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter your password"
              required
              autoComplete="current-password"
            />
          </div>

          <button type="submit" className="btn-primary" disabled={isSubmitting}>
            {isSubmitting ? 'Signing in...' : 'Sign In'}
          </button>
        </form>

        <div className="auth-footer">
          <p>
            Don't have an account? <Link to={`/register?redirect=${encodeURIComponent(redirect)}${prefillEmail ? `&email=${encodeURIComponent(prefillEmail)}` : ''}`}>Create one</Link>
          </p>
        </div>
      </div>
    </div>
  );
}
