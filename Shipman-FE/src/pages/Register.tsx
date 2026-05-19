import { useState, useRef, useEffect } from 'react';
import type { FormEvent } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

type UserRole = 'shipowner' | 'charterer' | 'broker';

export default function Register() {
  const [searchParams] = useSearchParams();
  const prefillEmail = searchParams.get('email') ?? '';
  const redirect = searchParams.get('redirect') ?? '/dashboard';

  const [email, setEmail] = useState(prefillEmail);
  const [password, setPassword] = useState('');
  const [fullName, setFullName] = useState('');
  const [role, setRole] = useState<UserRole>('charterer');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { signup, error, clearError } = useAuth();
  const navigate = useNavigate();
  const nameRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (prefillEmail) nameRef.current?.focus();
  }, [prefillEmail]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await signup({ email, password, full_name: fullName, role });
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
          <p>Create your account</p>
        </div>

        {error && (
          <div className="error-banner" onClick={clearError}>
            {error}
          </div>
        )}

        {prefillEmail && (
          <div className="invite-email-hint">
            Creating an account for <strong>{prefillEmail}</strong> to join a negotiation
          </div>
        )}

        <form onSubmit={handleSubmit} className="auth-form">
          <div className="form-group">
            <label htmlFor="fullName">Full Name</label>
            <input
              ref={nameRef}
              id="fullName"
              type="text"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              placeholder="John Smith"
              required
              autoComplete="name"
            />
          </div>

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
              readOnly={!!prefillEmail}
              style={prefillEmail ? { background: 'var(--color-bg)', color: 'var(--color-text-secondary)' } : undefined}
            />
          </div>

          <div className="form-group">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Minimum 8 characters"
              required
              minLength={8}
              autoComplete="new-password"
            />
          </div>

          <div className="form-group">
            <label htmlFor="role">I am a...</label>
            <select
              id="role"
              value={role}
              onChange={(e) => setRole(e.target.value as UserRole)}
              required
            >
              <option value="charterer">Charterer</option>
              <option value="shipowner">Ship Owner</option>
              <option value="broker">Broker</option>
            </select>
          </div>

          <button type="submit" className="btn-primary" disabled={isSubmitting}>
            {isSubmitting ? 'Creating account...' : 'Create Account'}
          </button>
        </form>

        <div className="auth-footer">
          <p>
            Already have an account? <Link to={`/login?redirect=${encodeURIComponent(redirect)}${prefillEmail ? `&email=${encodeURIComponent(prefillEmail)}` : ''}`}>Sign in</Link>
          </p>
        </div>
      </div>
    </div>
  );
}
