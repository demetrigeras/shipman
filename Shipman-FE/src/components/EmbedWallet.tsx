import { useEffect, useRef, useState } from 'react';
import { api, ApiError } from '../api/client';

// ────────────────────────────────────────────────────────────────────────────
// EmbedWallet — renders a RocketRamp wallet iframe inline (no modal).
//
// On mount it asks the backend to mint a single-use embed_code prefilled with
// the recipient, then loads the iframe at <embed_base_url>/<embed_code>.
// Each embed_code is single-use, so we deliberately mint once per
// (recipientEmail, memo) pair and avoid re-minting on unrelated re-renders.
// ────────────────────────────────────────────────────────────────────────────

export interface EmbedWalletProps {
  recipientEmail: string;
  memo?: string;
  height?: number | string;
  title?: string;
}

export default function EmbedWallet({
  recipientEmail,
  memo,
  height = 720,
  title = 'RocketRamp Wallet',
}: EmbedWalletProps) {
  const [iframeSrc, setIframeSrc] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [minting, setMinting] = useState(true);
  const [loaded, setLoaded] = useState(false);
  const lastKey = useRef<string>('');

  useEffect(() => {
    const key = `${recipientEmail}|${memo ?? ''}`;
    if (lastKey.current === key) return;
    lastKey.current = key;

    let cancelled = false;
    setMinting(true);
    setError(null);
    setIframeSrc(null);
    setLoaded(false);

    api.payments
      .createEmbedCode(recipientEmail, memo)
      .then(res => {
        if (cancelled) return;
        setIframeSrc(`${res.embed_base_url}/${res.embed_code}?t=${Date.now()}`);
      })
      .catch(e => {
        if (cancelled) return;
        setError(e instanceof ApiError ? e.message : 'Failed to start RocketRamp session');
      })
      .finally(() => {
        if (!cancelled) setMinting(false);
      });

    return () => {
      cancelled = true;
    };
  }, [recipientEmail, memo]);

  const handleRetry = () => {
    lastKey.current = '';
    setError(null);
    setMinting(true);
    setIframeSrc(null);
    setLoaded(false);
    // bump effect by toggling the key
    setTimeout(() => {
      lastKey.current = '';
    }, 0);
  };

  return (
    <div className="embed-wallet" style={{ minHeight: height }}>
      {error ? (
        <div className="embed-wallet-error">
          <h4>Couldn't start payment</h4>
          <p>{error}</p>
          <p className="embed-wallet-error-hint">
            Make sure <code>ROCKETRAMP_MERCHANT_ID</code> and{' '}
            <code>ROCKETRAMP_API_KEY</code> are set on the backend.
          </p>
          <button type="button" className="btn-secondary btn-sm" onClick={handleRetry}>
            Try again
          </button>
        </div>
      ) : iframeSrc ? (
        <>
          {!loaded && (
            <div className="embed-wallet-loader">
              <div className="coinsub-spinner" />
              <p>Loading Wallet…</p>
              <a
                href={iframeSrc}
                target="_blank"
                rel="noopener noreferrer"
                className="coinsub-fallback-link"
              >
                Wallet not loading? Open in new tab →
              </a>
            </div>
          )}
          <iframe
            title={title}
            src={iframeSrc}
            className="embed-wallet-iframe"
            style={{ height }}
            onLoad={() => setLoaded(true)}
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox"
            allow="clipboard-read *; publickey-credentials-create *; publickey-credentials-get *"
          />
        </>
      ) : (
        <div className="embed-wallet-loader">
          <div className="coinsub-spinner" />
          <p>{minting ? 'Preparing payment…' : 'Loading…'}</p>
        </div>
      )}
    </div>
  );
}
