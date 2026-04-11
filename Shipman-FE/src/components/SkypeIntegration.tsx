import { useState } from 'react';

interface SkypeIntegrationProps {
  dealId?: string;
  participantEmails?: string[];
}

export default function SkypeIntegration({ participantEmails = [] }: SkypeIntegrationProps) {
  const [isConfigured] = useState(false);

  const handleStartCall = () => {
    if (participantEmails.length > 0) {
      const skypeUri = `skype:${participantEmails.join(';')}?call`;
      window.open(skypeUri, '_blank');
    }
  };

  const handleStartChat = () => {
    if (participantEmails.length > 0) {
      const skypeUri = `skype:${participantEmails.join(';')}?chat`;
      window.open(skypeUri, '_blank');
    }
  };

  if (!isConfigured) {
    return (
      <div className="skype-integration">
        <div className="skype-placeholder">
          <div className="skype-icon">📞</div>
          <h4>Communication</h4>
          <p>Start a Skype call or chat with deal participants.</p>
          <div className="skype-actions">
            <button 
              className="btn-secondary"
              onClick={handleStartChat}
              disabled={participantEmails.length === 0}
            >
              Open Chat
            </button>
            <button 
              className="btn-primary"
              onClick={handleStartCall}
              disabled={participantEmails.length === 0}
            >
              Start Call
            </button>
          </div>
          {participantEmails.length === 0 && (
            <p className="hint">Add participants to enable communication.</p>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="skype-integration">
      <div className="skype-embed">
        <p>Skype Web SDK integration requires Azure AD configuration.</p>
        <p className="hint">
          To enable embedded Skype, configure REACT_APP_AZURE_CLIENT_ID 
          and REACT_APP_AZURE_TENANT_ID environment variables.
        </p>
      </div>
    </div>
  );
}
