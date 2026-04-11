import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { api, ApiError } from '../api/client';
import { useAuth } from '../context/AuthContext';

type JoinState = 'loading' | 'preview' | 'joining' | 'done' | 'error';

interface InvitePreview {
  token: string;
  role: string;
  deal_id: string;
  deal_title: string;
  expires_at: string;
}

const ROLE_LABELS: Record<string, string> = {
  shipowner: 'Ship Owner',
  charterer: 'Charterer',
  broker: 'Broker',
};

export default function Join() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') ?? '';
  const navigate = useNavigate();
  const { user, isLoading: authLoading } = useAuth();

  const [state, setState] = useState<JoinState>('loading');
  const [preview, setPreview] = useState<InvitePreview | null>(null);
  const [errorMsg, setErrorMsg] = useState('');

  // Load invite preview (public endpoint — no auth needed)
  useEffect(() => {
    if (!token) {
      setState('error');
      setErrorMsg('No invite token in the link. Please ask for a new invite.');
      return;
    }

    api.deals.previewInvite(token)
      .then(data => {
        setPreview(data);
        setState('preview');
      })
      .catch(e => {
        setState('error');
        if (e instanceof ApiError && e.status === 410) {
          setErrorMsg('This invite has already been used or has expired.');
        } else if (e instanceof ApiError && e.status === 404) {
          setErrorMsg('This invite link is invalid.');
        } else {
          setErrorMsg('Failed to load invite. Please try again.');
        }
      });
  }, [token]);

  // If not logged in, redirect to register (or login) with the token preserved
  const handleNotLoggedIn = (page: 'login' | 'register') => {
    navigate(`/${page}?redirect=/join?token=${token}`);
  };

  const handleJoin = async () => {
    if (!preview) return;
    setState('joining');
    try {
      const result = await api.deals.join(token);
      setState('done');
      // Short delay so user sees confirmation, then enter deal room
      setTimeout(() => navigate(`/deals/${result.deal.id}`), 1200);
    } catch (e) {
      setState('error');
      setErrorMsg(e instanceof ApiError ? e.message : 'Failed to join deal');
    }
  };

  // ── Loading ──────────────────────────────────────────────────────────────
  if (state === 'loading') {
    return (
      <div className="join-page">
        <div className="join-card">
          <div className="loading-spinner" style={{ margin: '2rem auto' }} />
          <p style={{ textAlign: 'center', color: 'var(--color-text-secondary)' }}>Loading invite…</p>
        </div>
      </div>
    );
  }

  // ── Error ────────────────────────────────────────────────────────────────
  if (state === 'error') {
    return (
      <div className="join-page">
        <div className="join-card">
          <div className="join-icon join-icon--error">✗</div>
          <h2>Invite Invalid</h2>
          <p>{errorMsg}</p>
          <button className="btn-primary" onClick={() => navigate('/dashboard')}>Go to Dashboard</button>
        </div>
      </div>
    );
  }

  // ── Done ─────────────────────────────────────────────────────────────────
  if (state === 'done') {
    return (
      <div className="join-page">
        <div className="join-card">
          <div className="join-icon join-icon--success">✓</div>
          <h2>Joined Successfully!</h2>
          <p>Taking you to the deal room…</p>
        </div>
      </div>
    );
  }

  // ── Preview ──────────────────────────────────────────────────────────────
  return (
    <div className="join-page">
      <div className="join-card">
        <div className="join-logo">⚓ Shipman</div>

        <div className="join-deal-info">
          <p className="join-label">You've been invited to negotiate as</p>
          <span className={`join-role-badge join-role-${preview?.role}`}>
            {ROLE_LABELS[preview?.role ?? ''] ?? preview?.role}
          </span>
          <h2 className="join-deal-title">{preview?.deal_title}</h2>
          <p className="join-expires">
            Invite expires {preview?.expires_at ? new Date(preview.expires_at).toLocaleDateString() : ''}
          </p>
        </div>

        {authLoading ? (
          <div className="loading-spinner" style={{ margin: '1.5rem auto' }} />
        ) : !user ? (
          <div className="join-auth-prompt">
            <p>You need an account to join this negotiation.</p>
            <div className="join-auth-buttons">
              <button className="btn-secondary" onClick={() => handleNotLoggedIn('login')}>
                I already have an account
              </button>
              <button className="btn-primary" onClick={() => handleNotLoggedIn('register')}>
                Create a free account →
              </button>
            </div>
          </div>
        ) : (
          <div className="join-confirm">
            <p>Joining as <strong>{user.full_name}</strong> ({user.email})</p>
            <button
              className="btn-primary"
              style={{ width: '100%', marginTop: '1rem' }}
              onClick={handleJoin}
              disabled={state === 'joining'}
            >
              {state === 'joining' ? 'Joining…' : `Join as ${ROLE_LABELS[preview?.role ?? ''] ?? 'Participant'} →`}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
