import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { api, ApiError } from '../api/client';
import { useAuth } from '../context/AuthContext';

type JoinState = 'loading' | 'preview' | 'joining' | 'done' | 'error';
type InviteType = 'deal' | 'voyage';

interface InvitePreview {
  token: string;
  type: InviteType;
  role: string;
  // deal fields
  deal_id?: string;
  deal_title?: string;
  // voyage fields
  voyage_id?: string;
  fixture_title?: string;
  // common
  invited_email: string;
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
  const typeHint = (searchParams.get('type') ?? 'deal') as InviteType;
  const navigate = useNavigate();
  const { user, isLoading: authLoading } = useAuth();

  const [state, setState] = useState<JoinState>('loading');
  const [preview, setPreview] = useState<InvitePreview | null>(null);
  const [errorMsg, setErrorMsg] = useState('');

  useEffect(() => {
    if (!token) {
      setState('error');
      setErrorMsg('No invite token in the link. Please ask for a new invite.');
      return;
    }

    if (typeHint === 'voyage') {
      api.voyages.previewInvite(token)
        .then(data => {
          setPreview({ ...data, type: 'voyage' });
          setState('preview');
        })
        .catch(e => {
          setState('error');
          if (e instanceof ApiError && (e.status === 410 || e.status === 404)) {
            setErrorMsg('This invite has already been used, expired, or is invalid.');
          } else {
            setErrorMsg('Failed to load invite. Please try again.');
          }
        });
    } else {
      api.deals.previewInvite(token)
        .then(data => {
          setPreview({ ...data, type: 'deal' });
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
    }
  }, [token, typeHint]);

  const handleNotLoggedIn = (page: 'login' | 'register') => {
    const returnUrl = `/join?token=${token}&type=${typeHint}`;
    const emailParam = preview?.invited_email ? `&email=${encodeURIComponent(preview.invited_email)}` : '';
    navigate(`/${page}?redirect=${encodeURIComponent(returnUrl)}${emailParam}`);
  };

  const handleJoin = async () => {
    if (!preview) return;
    setState('joining');
    try {
      if (preview.type === 'voyage') {
        const result = await api.voyages.joinVoyage(token);
        setState('done');
        setTimeout(() => navigate(`/voyages/${result.voyage_id}`), 1200);
      } else {
        const result = await api.deals.join(token);
        setState('done');
        setTimeout(() => navigate(`/deals/${result.deal.id}`), 1200);
      }
    } catch (e) {
      if (e instanceof ApiError && e.message.includes('already a participant')) {
        setState('done');
        const dest = preview.type === 'voyage' ? `/voyages/${preview.voyage_id}` : `/deals/${preview.deal_id}`;
        setTimeout(() => navigate(dest), 800);
      } else {
        setState('error');
        setErrorMsg(e instanceof ApiError ? e.message : 'Failed to join');
      }
    }
  };

  const title = preview?.type === 'voyage' ? (preview.fixture_title || 'Fixed C/P') : (preview?.deal_title || 'Deal');
  const inviteLabel = preview?.type === 'voyage' ? 'join this charter party as' : 'negotiate as';
  const successLabel = preview?.type === 'voyage' ? 'Taking you to the fixture…' : 'Taking you to the deal room…';

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
          <p>{successLabel}</p>
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
          <p className="join-label">You've been invited to {inviteLabel}</p>
          <span className={`join-role-badge join-role-${preview?.role}`}>
            {ROLE_LABELS[preview?.role ?? ''] ?? preview?.role}
          </span>
          <h2 className="join-deal-title">{title}</h2>
          <p className="join-expires">
            Invite expires {preview?.expires_at ? new Date(preview.expires_at).toLocaleDateString() : ''}
          </p>
        </div>

        {authLoading ? (
          <div className="loading-spinner" style={{ margin: '1.5rem auto' }} />
        ) : !user ? (
          <div className="join-auth-prompt">
            <p>You need an account to join.</p>
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
            {preview?.invited_email && user.email.toLowerCase() !== preview.invited_email.toLowerCase() && (
              <div className="join-wrong-user-warning">
                <strong>⚠ Wrong account</strong>
                <p>
                  This invite was sent to <strong>{preview.invited_email}</strong>.<br />
                  You are currently signed in as <strong>{user.email}</strong>.
                </p>
                <div className="join-auth-buttons" style={{ marginTop: '0.75rem' }}>
                  <button className="btn-secondary" onClick={() => handleNotLoggedIn('login')}>
                    Sign in with the correct account
                  </button>
                </div>
              </div>
            )}
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
