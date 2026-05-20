import { useState } from 'react';
import { api, ApiError } from '../api/client';

// ────────────────────────────────────────────────────────────────────────────
// PayButton — opens a RocketRamp wallet iframe modal with the given recipient
// already prefilled. On click:
//   1. Backend mints a fresh single-use embed_code via the Vantack prefill API
//   2. Modal opens at <embed_base_url>/<embed_code>
//   3. User signs in to their wallet, picks amount, sends
//
// The merchant API key never touches the browser; the FE only ever sees the
// per-session embed_code returned from our backend.
// ────────────────────────────────────────────────────────────────────────────

export interface PayButtonProps {
  recipientEmail: string;
  label?: string;
  memo?: string;
  className?: string;
}

export default function PayButton({ recipientEmail, label, memo, className }: PayButtonProps) {
  const [open, setOpen] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [iframeSrc, setIframeSrc] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [minting, setMinting] = useState(false);

  const close = () => {
    setOpen(false);
    setLoaded(false);
    setIframeSrc(null);
    setError(null);
  };

  const handleClick = async () => {
    setOpen(true);
    setMinting(true);
    setError(null);
    try {
      const result = await api.payments.createEmbedCode(recipientEmail, memo);
      setIframeSrc(`${result.embed_base_url}/${result.embed_code}?t=${Date.now()}`);
    } catch (e) {
      setError(
        e instanceof ApiError ? e.message : 'Failed to start RocketRamp session',
      );
    } finally {
      setMinting(false);
    }
  };

  return (
    <>
      <button
        type="button"
        className={className ?? 'btn-coinsub'}
        onClick={handleClick}
        title={`Send funds to ${recipientEmail} via RocketRamp`}
      >
        {label ?? `Pay ${recipientEmail}`}
      </button>

      {open && (
        <div className="coinsub-modal-overlay" onClick={close}>
          <div className="coinsub-modal" onClick={e => e.stopPropagation()}>
            <button className="coinsub-modal-close" onClick={close} aria-label="Close">✕</button>

            {error ? (
              <div className="coinsub-no-key">
                <h3>Couldn't start payment</h3>
                <p>{error}</p>
                <p style={{ fontSize: '0.8rem', color: '#6b7280' }}>
                  Make sure <code>ROCKETRAMP_MERCHANT_ID</code> and{' '}
                  <code>ROCKETRAMP_API_KEY</code> are set on the backend.
                </p>
              </div>
            ) : iframeSrc ? (
              <>
                {!loaded && (
                  <div className="coinsub-loader">
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
                  title="RocketRamp Wallet"
                  src={iframeSrc}
                  className="coinsub-iframe"
                  onLoad={() => setLoaded(true)}
                  sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox"
                  allow="clipboard-read *; publickey-credentials-create *; publickey-credentials-get *"
                />
              </>
            ) : (
              <div className="coinsub-loader">
                <div className="coinsub-spinner" />
                <p>{minting ? 'Preparing payment…' : 'Loading…'}</p>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
}
