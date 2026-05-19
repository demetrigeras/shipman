import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api, ApiError } from '../api/client';
import type {
  Deal, DealParticipant, ClauseNegotiation, DealVesselDetails, DealCargoDetails,
  ClauseProposal, AIAnalysis,
} from '../api/client';
import { useAuth } from '../context/AuthContext';
import NavBar from '../components/NavBar';

// ────────────────────────────────────────────────────────────────────────────
// Helper: relative time (e.g. "2h ago")
// ────────────────────────────────────────────────────────────────────────────
function timeAgo(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diffMs = now - then;
  const mins = Math.floor(diffMs / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(dateStr).toLocaleDateString();
}

// ────────────────────────────────────────────────────────────────────────────
// CoinSub / RocketRamp button. Opens the wallet in a new browser tab.
//
// We deliberately do NOT use an iframe: RocketRamp's /embed/{code} endpoint
// enforces a domain allowlist via Sec-Fetch-Dest=iframe + Referer, which
// can't be configured to permit http://localhost:* on the Vantack dashboard.
// Top-level navigation (Sec-Fetch-Dest=document) bypasses that check by
// design, since it's no longer an embed.
//
// Sandbox creds (test.vantack.com) → test.myrocketramp.com.
// Production creds (app.vantack.com) → app.myrocketramp.com.
// ────────────────────────────────────────────────────────────────────────────

const ROCKETRAMP_TEST_BASE = 'https://test.myrocketramp.com/embed';
const ROCKETRAMP_PROD_BASE = 'https://app.myrocketramp.com/embed';

interface CoinSubButtonProps {
  embedKey?: string;
  label?: string;
  testMode?: boolean;
}

function CoinSubButton({ embedKey, label = 'Get Credits', testMode = true }: CoinSubButtonProps) {
  const base = testMode ? ROCKETRAMP_TEST_BASE : ROCKETRAMP_PROD_BASE;

  const handleClick = () => {
    if (!embedKey) {
      alert(
        'No embed code configured. Mint one by POSTing to ' +
          (testMode ? 'test-api.vantack.com' : 'api.vantack.com') +
          '/v1/merchants/embed/prefill, then pass it as the embedKey prop.',
      );
      return;
    }
    const url = `${base}/${embedKey}?t=${Date.now()}`;
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  return (
    <button
      type="button"
      className="btn-coinsub"
      onClick={handleClick}
      title={embedKey ? 'Open RocketRamp wallet in a new tab' : 'No embed code configured'}
    >
      {label}
    </button>
  );
}

// ────────────────────────────────────────────────────────────────────────────
// Small sub-components
// ────────────────────────────────────────────────────────────────────────────

function DetailRow({ label, value }: { label: string; value?: string | number | null }) {
  if (!value && value !== 0) return null;
  return (
    <div className="detail-row">
      <span className="detail-label">{label}</span>
      <span className="detail-value">{value}</span>
    </div>
  );
}

// ────────────────────────────────────────────────────────────────────────────
// Vessel / Cargo edit forms (inline panels)
// ────────────────────────────────────────────────────────────────────────────

interface VesselEditProps {
  dealId: string;
  initial?: DealVesselDetails | null;
  onSaved: (d: DealVesselDetails) => void;
  onCancel?: () => void;
  compact?: boolean;
}

function VesselEditForm({ dealId, initial, onSaved, onCancel, compact }: VesselEditProps) {
  const [form, setForm] = useState({
    vessel_name: initial?.vessel_name ?? '',
    imo_number: initial?.imo_number ?? '',
    vessel_type: initial?.vessel_type ?? '',
    flag_state: initial?.flag_state ?? '',
    deadweight_tonnage: initial?.deadweight_tonnage?.toString() ?? '',
    gross_tonnage: initial?.gross_tonnage?.toString() ?? '',
    build_year: initial?.build_year?.toString() ?? '',
    class_society: initial?.class_society ?? '',
    current_position: initial?.current_position ?? '',
    available_from: initial?.available_from?.slice(0, 10) ?? '',
    asking_rate: initial?.asking_rate?.toString() ?? '',
    asking_rate_currency: initial?.asking_rate_currency ?? 'USD',
    asking_rate_type: initial?.asking_rate_type ?? 'per_day',
    notes: initial?.notes ?? '',
  });
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const f = (field: keyof typeof form) => ({
    value: form[field],
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
      setForm(prev => ({ ...prev, [field]: e.target.value })),
  });

  const handleSave = async () => {
    setSaving(true);
    setErr(null);
    try {
      const payload: Record<string, string | number | undefined> = {};
      if (form.vessel_name) payload.vessel_name = form.vessel_name;
      if (form.imo_number) payload.imo_number = form.imo_number;
      if (form.vessel_type) payload.vessel_type = form.vessel_type;
      if (form.flag_state) payload.flag_state = form.flag_state;
      if (form.deadweight_tonnage) payload.deadweight_tonnage = parseFloat(form.deadweight_tonnage);
      if (form.gross_tonnage) payload.gross_tonnage = parseFloat(form.gross_tonnage);
      if (form.build_year) payload.build_year = parseInt(form.build_year);
      if (form.class_society) payload.class_society = form.class_society;
      if (form.current_position) payload.current_position = form.current_position;
      if (form.available_from) payload.available_from = form.available_from;
      if (form.asking_rate) payload.asking_rate = parseFloat(form.asking_rate);
      payload.asking_rate_currency = form.asking_rate_currency;
      payload.asking_rate_type = form.asking_rate_type;
      if (form.notes) payload.notes = form.notes;
      const result = await api.deals.upsertVesselDetails(dealId, payload);
      onSaved(result);
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Failed to save');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className={compact ? 'details-form compact' : 'details-form'}>
      {err && <div className="form-error">{err}</div>}
      <div className="form-grid">
        <div className="form-group"><label>Vessel Name</label><input type="text" placeholder="MV Ocean Star" {...f('vessel_name')} /></div>
        <div className="form-group"><label>IMO</label><input type="text" placeholder="9876543" {...f('imo_number')} /></div>
        <div className="form-group"><label>Type</label><input type="text" placeholder="Bulk Carrier" {...f('vessel_type')} /></div>
        <div className="form-group"><label>Flag</label><input type="text" placeholder="Panama" {...f('flag_state')} /></div>
        <div className="form-group"><label>DWT (t)</label><input type="number" {...f('deadweight_tonnage')} /></div>
        <div className="form-group"><label>GRT</label><input type="number" {...f('gross_tonnage')} /></div>
        <div className="form-group"><label>Built</label><input type="number" placeholder="2018" {...f('build_year')} /></div>
        <div className="form-group"><label>Class</label><input type="text" placeholder="Lloyd's" {...f('class_society')} /></div>
        <div className="form-group"><label>Position</label><input type="text" placeholder="Rotterdam" {...f('current_position')} /></div>
        <div className="form-group"><label>Available</label><input type="date" {...f('available_from')} /></div>
        <div className="form-group"><label>Rate</label><input type="number" placeholder="0.00" {...f('asking_rate')} /></div>
        <div className="form-group">
          <label>Rate Type</label>
          <select {...f('asking_rate_type')}>
            <option value="per_day">Per Day</option>
            <option value="lumpsum">Lump Sum</option>
          </select>
        </div>
      </div>
      <div className="form-group"><label>Notes</label><textarea rows={2} {...f('notes')} /></div>
      <div className="form-actions">
        {onCancel && <button className="btn-secondary btn-sm" onClick={onCancel}>Cancel</button>}
        <button className="btn-primary btn-sm" onClick={handleSave} disabled={saving}>
          {saving ? 'Saving…' : 'Save Vessel Details'}
        </button>
      </div>
    </div>
  );
}

interface CargoEditProps {
  dealId: string;
  initial?: DealCargoDetails | null;
  onSaved: (d: DealCargoDetails) => void;
  onCancel?: () => void;
  compact?: boolean;
}

function CargoEditForm({ dealId, initial, onSaved, onCancel, compact }: CargoEditProps) {
  const [form, setForm] = useState({
    commodity: initial?.commodity ?? '',
    quantity: initial?.quantity?.toString() ?? '',
    quantity_unit: initial?.quantity_unit ?? 'MT',
    load_port: initial?.load_port ?? '',
    discharge_port: initial?.discharge_port ?? '',
    laycan_from: initial?.laycan_from?.slice(0, 10) ?? '',
    laycan_to: initial?.laycan_to?.slice(0, 10) ?? '',
    freight_idea: initial?.freight_idea?.toString() ?? '',
    freight_currency: initial?.freight_currency ?? 'USD',
    freight_type: initial?.freight_type ?? 'per_mt',
    special_requirements: initial?.special_requirements ?? '',
    notes: initial?.notes ?? '',
  });
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const f = (field: keyof typeof form) => ({
    value: form[field],
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
      setForm(prev => ({ ...prev, [field]: e.target.value })),
  });

  const handleSave = async () => {
    setSaving(true);
    setErr(null);
    try {
      const payload: Record<string, string | number | undefined> = {};
      if (form.commodity) payload.commodity = form.commodity;
      if (form.quantity) payload.quantity = parseFloat(form.quantity);
      payload.quantity_unit = form.quantity_unit;
      if (form.load_port) payload.load_port = form.load_port;
      if (form.discharge_port) payload.discharge_port = form.discharge_port;
      if (form.laycan_from) payload.laycan_from = form.laycan_from;
      if (form.laycan_to) payload.laycan_to = form.laycan_to;
      if (form.freight_idea) payload.freight_idea = parseFloat(form.freight_idea);
      payload.freight_currency = form.freight_currency;
      payload.freight_type = form.freight_type;
      if (form.special_requirements) payload.special_requirements = form.special_requirements;
      if (form.notes) payload.notes = form.notes;
      const result = await api.deals.upsertCargoDetails(dealId, payload);
      onSaved(result);
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Failed to save');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className={compact ? 'details-form compact' : 'details-form'}>
      {err && <div className="form-error">{err}</div>}
      <div className="form-grid">
        <div className="form-group"><label>Commodity</label><input type="text" placeholder="Iron Ore" {...f('commodity')} /></div>
        <div className="form-group">
          <label>Quantity</label>
          <div className="input-with-addon">
            <input type="number" placeholder="50000" {...f('quantity')} />
            <select style={{ width: 90 }} {...f('quantity_unit')}>
              <option value="MT">MT</option>
              <option value="CBM">CBM</option>
              <option value="TEU">TEU</option>
            </select>
          </div>
        </div>
        <div className="form-group"><label>Load Port</label><input type="text" placeholder="Rotterdam" {...f('load_port')} /></div>
        <div className="form-group"><label>Discharge Port</label><input type="text" placeholder="Singapore" {...f('discharge_port')} /></div>
        <div className="form-group"><label>Laycan From</label><input type="date" {...f('laycan_from')} /></div>
        <div className="form-group"><label>Laycan To</label><input type="date" {...f('laycan_to')} /></div>
        <div className="form-group"><label>Freight Idea</label><input type="number" placeholder="0.00" {...f('freight_idea')} /></div>
        <div className="form-group">
          <label>Freight Type</label>
          <select {...f('freight_type')}>
            <option value="per_mt">Per MT</option>
            <option value="lumpsum">Lump Sum</option>
            <option value="per_day">Per Day</option>
          </select>
        </div>
      </div>
      <div className="form-group"><label>Special Requirements</label><textarea rows={2} {...f('special_requirements')} /></div>
      <div className="form-group"><label>Notes</label><textarea rows={2} {...f('notes')} /></div>
      <div className="form-actions">
        {onCancel && <button className="btn-secondary btn-sm" onClick={onCancel}>Cancel</button>}
        <button className="btn-primary btn-sm" onClick={handleSave} disabled={saving}>
          {saving ? 'Saving…' : 'Save Cargo Details'}
        </button>
      </div>
    </div>
  );
}

// ────────────────────────────────────────────────────────────────────────────
// Main DealViewer
// ────────────────────────────────────────────────────────────────────────────

export default function DealViewer() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();

  const [deal, setDeal] = useState<Deal | null>(null);
  const [participants, setParticipants] = useState<DealParticipant[]>([]);
  const [vesselDetails, setVesselDetails] = useState<DealVesselDetails | null>(null);
  const [cargoDetails, setCargoDetails] = useState<DealCargoDetails | null>(null);
  const [negotiations, setNegotiations] = useState<ClauseNegotiation[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // UI state
  const [editingVessel, setEditingVessel] = useState(false);
  const [editingCargo, setEditingCargo] = useState(false);
  const [showInvite, setShowInvite] = useState(false);
  const [inviteRole, setInviteRole] = useState<'shipowner' | 'charterer' | 'broker'>('charterer');
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteResult, setInviteResult] = useState<{ link: string; email_sent: boolean } | null>(null);
  const [linkCopied, setLinkCopied] = useState(false);
  const [activeNeg, setActiveNeg] = useState<ClauseNegotiation | null>(null);
  const [proposalText, setProposalText] = useState('');
  const [commentText, setCommentText] = useState('');
  const [proposals, setProposals] = useState<ClauseProposal[]>([]);
  const [loadingProposals, setLoadingProposals] = useState(false);
  const [dealCompleted, setDealCompleted] = useState(false);
  const conversationEndRef = useRef<HTMLDivElement>(null);

  // Charter party upload state
  const [cpUploadState, setCpUploadState] = useState<'idle' | 'uploading' | 'extracting' | 'done'>('idle');
  const [cpDocId, setCpDocId] = useState<string | null>(null);
  const [cpDocText, setCpDocText] = useState<string | null>(null);
  const cpInputRef = useRef<HTMLInputElement>(null);

  // AI scanning state
  const [aiScanState, setAiScanState] = useState<'idle' | 'scanning' | 'done'>('idle');
  const [aiAnalysis, setAiAnalysis] = useState<AIAnalysis | null>(null);

  const loadDeal = useCallback(async (dealId: string) => {
    try {
      const response = await api.deals.get(dealId);
      setDeal(response.deal);
      setParticipants(response.participants || []);
      setVesselDetails(response.vessel_details ?? null);
      setCargoDetails(response.cargo_details ?? null);

      const negsResponse = await api.deals.listNegotiations(dealId);
      setNegotiations(negsResponse.data || []);
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        navigate('/deals');
        return;
      }
      setError('Failed to load deal');
    } finally {
      setIsLoading(false);
    }
  }, [navigate]);

  useEffect(() => {
    if (id) loadDeal(id);
  }, [id, loadDeal]);

  // Determine my role in this deal
  const myParticipant = participants.find(p => p.user_id === user?.id);
  const myRole = myParticipant?.role ?? user?.role ?? 'broker';
  const canEditVessel = myRole === 'shipowner' || myRole === 'broker';
  const canEditCargo = myRole === 'charterer' || myRole === 'broker';

  // Should we show the "fill in your side" prompt?
  const needsVessel = canEditVessel && myRole === 'shipowner' && !vesselDetails;
  const needsCargo = canEditCargo && myRole === 'charterer' && !cargoDetails;
  const [showFillPrompt, setShowFillPrompt] = useState(false);

  useEffect(() => {
    if (!isLoading && (needsVessel || needsCargo)) {
      setShowFillPrompt(true);
    }
  }, [isLoading, needsVessel, needsCargo]);

  const handleCreateInvite = async () => {
    if (!id || !inviteEmail.trim()) return;
    try {
      const result = await api.deals.createInvite(id, inviteEmail.trim(), inviteRole);
      setInviteResult({ link: result.invite_link, email_sent: result.email_sent });
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to create invite');
    }
  };

  const handleUploadCharterParty = async (file: File) => {
    if (!id) return;
    setError(null);
    try {
      setCpUploadState('uploading');
      const doc = await api.documents.upload(file);

      setCpUploadState('extracting');
      // Just extract text — free, no AI needed
      const processed = await api.documents.process(doc.id);
      await api.deals.attachDocument(id, doc.id);

      setCpDocId(doc.id);
      setCpDocText(processed.extracted_text ?? null);
      setCpUploadState('done');
    } catch (e) {
      setCpUploadState('idle');
      setError(e instanceof ApiError ? e.message : 'Failed to process charter party');
    }
  };

  // AI scan the charter party for negotiation clauses
  const handleAIScan = async () => {
    if (!id || !cpDocId) return;
    setError(null);
    setAiScanState('scanning');
    try {
      const result = await api.documents.analyze(cpDocId);
      setAiAnalysis(result.analysis);
      setAiScanState('done');

      // Auto-create negotiations from high/medium importance clauses
      if (result.analysis?.clauses) {
        const importantClauses = result.analysis.clauses.filter(c => c.importance === 'high' || c.importance === 'medium');
        for (let i = 0; i < importantClauses.length; i++) {
          const clause = importantClauses[i];
          await api.deals.createNegotiation(id, {
            clause_type: clause.type,
            clause_title: clause.title,
            original_content: clause.content,
            sort_order: i,
          }).catch(() => {});
        }
        const negsResponse = await api.deals.listNegotiations(id);
        setNegotiations(negsResponse.data || []);
      }
    } catch (e) {
      setAiScanState('idle');
      setError(e instanceof ApiError ? e.message : 'AI scan failed');
    }
  };

  // Load proposals for a negotiation
  const loadProposals = async (negId: string) => {
    if (!id) return;
    setLoadingProposals(true);
    try {
      const result = await api.deals.getNegotiation(id, negId);
      setProposals(result.proposals || []);
    } catch (e) {
      console.error('Failed to load proposals:', e);
      setProposals([]);
    } finally {
      setLoadingProposals(false);
    }
  };

  // Handle clicking on a negotiation card
  const handleSelectNeg = (neg: ClauseNegotiation) => {
    if (activeNeg?.id === neg.id) {
      setActiveNeg(null);
      setProposals([]);
      setProposalText('');
      setCommentText('');
    } else {
      setActiveNeg(neg);
      setProposalText('');
      setCommentText('');
      loadProposals(neg.id);
    }
  };

  const handleCreateProposal = async () => {
    if (!id || !activeNeg || !proposalText.trim()) return;
    const negId = activeNeg.id;
    try {
      const body: { proposed_content: string; comment?: string } = {
        proposed_content: proposalText.trim(),
      };
      if (commentText.trim()) body.comment = commentText.trim();

      await api.deals.createProposal(id, negId, body);
      setProposalText('');
      setCommentText('');

      const updatedNeg = await api.deals.getNegotiation(id, negId);
      setProposals(updatedNeg.proposals || []);

      const negsResponse = await api.deals.listNegotiations(id);
      setNegotiations(negsResponse.data || []);
      const refreshed = negsResponse.data?.find(n => n.id === negId);
      if (refreshed) setActiveNeg(refreshed);

      setTimeout(() => conversationEndRef.current?.scrollIntoView({ behavior: 'smooth' }), 100);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to submit proposal');
    }
  };

  const reloadAfterAction = async (negId: string) => {
    const updatedNeg = await api.deals.getNegotiation(id!, negId);
    setProposals(updatedNeg.proposals || []);
    const negsResponse = await api.deals.listNegotiations(id!);
    setNegotiations(negsResponse.data || []);
    const refreshed = negsResponse.data?.find(n => n.id === negId);
    if (refreshed) setActiveNeg(refreshed);
  };

  const handleAcceptProposal = async (proposalId: string) => {
    if (!id || !activeNeg) return;
    try {
      const result = await api.deals.updateProposalStatus(id, activeNeg.id, proposalId, 'accepted');
      await reloadAfterAction(activeNeg.id);
      if (result.deal_completed) {
        setDealCompleted(true);
      }
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to accept proposal');
    }
  };

  const handleRejectProposal = async (proposalId: string) => {
    if (!id || !activeNeg) return;
    try {
      await api.deals.updateProposalStatus(id, activeNeg.id, proposalId, 'rejected');
      await reloadAfterAction(activeNeg.id);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to reject proposal');
    }
  };

  const statusColor: Record<string, string> = {
    pending: 'badge-open',
    open: 'badge-open',
    accepted: 'badge-success',
    rejected: 'badge-error',
    countered: 'badge-info',
  };

  const statusLabel: Record<string, string> = {
    pending: 'Open',
    open: 'Open',
    accepted: 'Agreed',
    rejected: 'Rejected',
    countered: 'Counter proposed',
  };

  if (isLoading) {
    return (
      <div className="loading-container">
        <div className="loading-spinner" />
        <p>Loading negotiation room…</p>
      </div>
    );
  }

  if (!deal) {
    return (
      <div className="error-container">
        <p>Negotiation not found</p>
        <button className="btn-primary" onClick={() => navigate('/deals')}>Back to Negotiations</button>
      </div>
    );
  }

  // ── Render ──────────────────────────────────────────────────────────────
  return (
    <div className="deal-room">
      <NavBar backTo="/deals" backLabel="Negotiations" />

      {/* TOP BAR */}
      <header className="deal-room-header">
        <div className="deal-room-title">
          <h1>{deal.title}</h1>
          <span className={`badge badge-${deal.status}`}>{deal.status}</span>
        </div>
        <div className="deal-room-actions">
          {/* TEMP smoke-test embed code from test.vantack.com. Single-use;
              re-mint with the prefill API and replace if it stops loading. */}
          <CoinSubButton embedKey="3363ccdf-20da-4236-bb64-0a20e983f894" testMode />
          <button className="btn-primary btn-sm" onClick={() => setShowInvite(true)}>
            + Invite Party
          </button>
        </div>
      </header>

      {error && <div className="error-banner" onClick={() => setError(null)}>{error}</div>}

      {(dealCompleted || deal?.status === 'completed') && (
        <div className="deal-completed-banner">
          <span>🎉 All clauses agreed — this deal is complete.</span>
          <button
            className="btn-start-ops"
            onClick={async () => {
              const v = await api.voyages.create({
                deal_id: deal?.id,
                vessel_name: vesselDetails?.vessel_name ?? undefined,
                imo_number: vesselDetails?.imo_number ?? undefined,
                dwt: vesselDetails?.deadweight_tonnage ?? undefined,
                cargo_type: cargoDetails?.commodity ?? undefined,
                cargo_quantity: cargoDetails?.quantity ?? undefined,
                status: 'planned',
              });
              navigate(`/voyages/${v.id}`);
            }}
          >
            🚢 Start Operations →
          </button>
        </div>
      )}

      {/* INVITE MODAL */}
      {showInvite && (
        <div className="modal-overlay" onClick={() => { setShowInvite(false); setInviteResult(null); setInviteEmail(''); }}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Invite to Negotiate</h2>
            {!inviteResult ? (
              <>
                <div className="form-group">
                  <label>Email Address</label>
                  <input
                    type="email"
                    value={inviteEmail}
                    onChange={e => setInviteEmail(e.target.value)}
                    placeholder="counterparty@company.com"
                    autoFocus
                    onKeyDown={e => e.key === 'Enter' && handleCreateInvite()}
                  />
                </div>
                <div className="form-group">
                  <label>Their Role</label>
                  <select value={inviteRole} onChange={e => setInviteRole(e.target.value as typeof inviteRole)}>
                    <option value="charterer">Charterer</option>
                    <option value="shipowner">Ship Owner</option>
                    <option value="broker">Broker</option>
                  </select>
                </div>
                <div className="modal-actions">
                  <button className="btn-secondary" onClick={() => setShowInvite(false)}>Cancel</button>
                  <button className="btn-primary" onClick={handleCreateInvite} disabled={!inviteEmail.trim()}>
                    Send Invite
                  </button>
                </div>
              </>
            ) : (
              <div className="invite-sent-success">
                <div className="invite-sent-icon">✉️</div>
                {inviteResult.email_sent ? (
                  <>
                    <h3>Invite Sent!</h3>
                    <p>An email has been sent to <strong>{inviteEmail}</strong> with a link to join this negotiation.</p>
                  </>
                ) : (
                  <>
                    <h3>Invite Created</h3>
                    <p>Copy this link and send it to <strong>{inviteEmail}</strong>:</p>
                    <div className="invite-link-box">
                      <input type="text" readOnly value={inviteResult.link} onClick={e => (e.target as HTMLInputElement).select()} />
                      <button
                        className="btn-primary btn-sm"
                        onClick={() => { navigator.clipboard.writeText(inviteResult.link); setLinkCopied(true); setTimeout(() => setLinkCopied(false), 2000); }}
                      >
                        {linkCopied ? 'Copied!' : 'Copy'}
                      </button>
                    </div>
                    <p style={{ fontSize: '0.8rem', color: '#6b7280', marginTop: '0.75rem' }}>
                      To send invites by email automatically, configure SMTP in config.local.yaml.
                    </p>
                  </>
                )}
                <button className="btn-primary" style={{ marginTop: '1.5rem' }} onClick={() => { setShowInvite(false); setInviteResult(null); setInviteEmail(''); setLinkCopied(false); }}>
                  Done
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* FILL YOUR SIDE PROMPT */}
      {showFillPrompt && (
        <div className="modal-overlay">
          <div className="modal modal-wide">
            <div className="modal-header">
              <h2>Welcome to the Deal Room!</h2>
              <p className="modal-sub">
                {needsVessel
                  ? "You've joined as a Ship Owner. Fill in your vessel details so the counterparty can evaluate the offer."
                  : "You've joined as a Charterer. Fill in your cargo requirements so the counterparty can evaluate the inquiry."}
              </p>
            </div>
            {needsVessel && (
              <VesselEditForm
                dealId={id!}
                onSaved={d => { setVesselDetails(d); setShowFillPrompt(false); setEditingVessel(false); }}
              />
            )}
            {needsCargo && (
              <CargoEditForm
                dealId={id!}
                onSaved={d => { setCargoDetails(d); setShowFillPrompt(false); setEditingCargo(false); }}
              />
            )}
            <div style={{ textAlign: 'right', marginTop: '0.5rem' }}>
              <button className="btn-link" onClick={() => setShowFillPrompt(false)}>Skip for now</button>
            </div>
          </div>
        </div>
      )}

      {/* 3-PANEL LAYOUT */}
      <div className="deal-panels">

        {/* ── LEFT: VESSEL ──────────────────────────────── */}
        <aside className="panel panel-vessel">
          <div className="panel-heading">
            <h3>⚓ Vessel</h3>
            {canEditVessel && (
              <button className="btn-link" onClick={() => setEditingVessel(true)}>
                {vesselDetails ? 'Edit' : '+ Add'}
              </button>
            )}
          </div>

          {vesselDetails ? (
            <div className="details-view">
              <DetailRow label="Vessel" value={vesselDetails.vessel_name} />
              <DetailRow label="IMO" value={vesselDetails.imo_number} />
              <DetailRow label="Type" value={vesselDetails.vessel_type} />
              <DetailRow label="Flag" value={vesselDetails.flag_state} />
              <DetailRow label="DWT" value={vesselDetails.deadweight_tonnage ? `${vesselDetails.deadweight_tonnage.toLocaleString()} t` : undefined} />
              <DetailRow label="GRT" value={vesselDetails.gross_tonnage ? `${vesselDetails.gross_tonnage.toLocaleString()} t` : undefined} />
              <DetailRow label="Built" value={vesselDetails.build_year} />
              <DetailRow label="Class" value={vesselDetails.class_society} />
              <DetailRow label="Position" value={vesselDetails.current_position} />
              <DetailRow label="Available" value={vesselDetails.available_from?.slice(0, 10)} />
              <DetailRow
                label="Rate"
                value={vesselDetails.asking_rate
                  ? `${vesselDetails.asking_rate_currency} ${vesselDetails.asking_rate.toLocaleString()} / ${vesselDetails.asking_rate_type.replace('_', ' ')}`
                  : undefined}
              />
              {vesselDetails.notes && <div className="detail-notes">{vesselDetails.notes}</div>}
            </div>
          ) : (
            <div className="panel-empty">
              {canEditVessel
                ? 'No vessel details yet. Click + Add to fill in your vessel.'
                : 'Waiting for shipowner to add vessel details.'}
            </div>
          )}

          <div className="panel-participants">
            <h4>Participants</h4>
            {participants.length === 0 ? (
              <p className="empty-text">Just you so far</p>
            ) : (
              <ul className="participants-list">
                {participants.map(p => (
                  <li key={p.id}>
                    <span className={`role-dot role-${p.role}`} />
                    <span>{p.user?.full_name || 'Pending'}</span>
                    <span className="participant-role">{p.role}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </aside>

        {/* ── CENTER: CHARTER PARTY / CLAUSES ───────────── */}
        <main className="panel panel-cp">
          <div className="panel-heading">
            <h3>📄 Charter Party</h3>
            <span className="clause-count">{negotiations.length} negotiation{negotiations.length !== 1 ? 's' : ''}</span>
          </div>

          {/* Add clause form removed — negotiations come from the uploaded document */}

          {/* Upload zone (shown when no document yet) */}
          {cpUploadState === 'idle' && negotiations.length === 0 && (
            <div className="panel-empty center-empty">
              <input
                type="file"
                ref={cpInputRef}
                accept=".pdf,.txt"
                style={{ display: 'none' }}
                onChange={e => { const f = e.target.files?.[0]; if (f) handleUploadCharterParty(f); e.target.value = ''; }}
              />
              <div style={{ fontSize: '2.5rem', marginBottom: '0.75rem' }}>📄</div>
              <h4>Attach Charter Party</h4>
              <p className="hint">Upload the PDF or text file. The document will appear here — read it and propose edits on individual clauses to start negotiating.</p>
              <button className="btn-primary" style={{ marginTop: '1rem' }} onClick={() => cpInputRef.current?.click()}>
                Upload Document
              </button>
            </div>
          )}

          {/* Loading states */}
          {(cpUploadState === 'uploading' || cpUploadState === 'extracting') && (
            <div className="center-empty">
              <div className="loading-spinner" style={{ margin: '0 auto 1rem' }} />
              <p>{cpUploadState === 'uploading' ? 'Uploading document…' : 'Extracting text from document…'}</p>
            </div>
          )}

          {/* Document viewer frame with AI scan */}
          {cpUploadState === 'done' && cpDocText && (
            <div className="cp-doc-viewer">
              {/* Toolbar */}
              <div className="cp-doc-toolbar">
                <span className="cp-doc-label">📄 Charter Party Document</span>
                <div className="cp-doc-actions">
                  {aiScanState === 'idle' && negotiations.length === 0 && (
                    <button className="btn-scan" onClick={handleAIScan}>
                      🔍 Scan for Negotiation Points
                    </button>
                  )}
                  {aiScanState === 'scanning' && (
                    <span className="scan-status">
                      <span className="loading-spinner-sm" /> AI scanning clauses…
                    </span>
                  )}
                  {(aiScanState === 'done' || negotiations.length > 0) && (
                    <button className="btn-secondary btn-sm" onClick={handleAIScan} disabled={aiScanState === 'scanning'}>
                      ↺ Re-scan
                    </button>
                  )}
                </div>
              </div>

              {/* Document text in a paper-like frame */}
              <div className="cp-doc-paper">
                {(() => {
                  const lines = cpDocText.split('\n').map(l => l.trim()).filter(Boolean);
                  const chunks: string[] = [];
                  let buf = '';
                  for (const line of lines) {
                    const startsClause = /^\d{1,3}[\.\)](\s|$)/.test(line);
                    if (startsClause && buf) { chunks.push(buf.trim()); buf = line; }
                    else { buf = buf ? buf + ' ' + line : line; }
                  }
                  if (buf) chunks.push(buf.trim());

                  return chunks.map((chunk, i) => {
                    const isAllCaps = chunk.length > 0 && chunk === chunk.toUpperCase() && /[A-Z]/.test(chunk) && chunk.length < 160;
                    return isAllCaps
                      ? <h4 key={i} className="cp-doc-heading">{chunk}</h4>
                      : <p key={i} className="cp-doc-para">{chunk}</p>;
                  });
                })()}
              </div>

              {/* AI Analysis summary (if available) */}
              {aiAnalysis && (
                <div className="cp-ai-summary">
                  <strong>AI Summary:</strong> {aiAnalysis.summary}
                  {aiAnalysis.risk_factors && aiAnalysis.risk_factors.length > 0 && (
                    <div className="cp-ai-risks">
                      <strong>Risk Factors:</strong>
                      <ul>
                        {aiAnalysis.risk_factors.slice(0, 3).map((r, i) => <li key={i}>{r}</li>)}
                      </ul>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Clause list */}
          {negotiations.length > 0 && (
            <div className="clause-list">
              {negotiations.map(neg => {
                const isActive = activeNeg?.id === neg.id;
                const latestProposal = isActive && proposals.length > 0 ? proposals[proposals.length - 1] : null;
                const pendingCount = isActive ? proposals.filter(p => p.status === 'pending').length : 0;

                return (
                  <div
                    key={neg.id}
                    className={`clause-card ${isActive ? 'clause-card--active' : ''}`}
                    onClick={() => handleSelectNeg(neg)}
                  >
                    {/* Collapsed view */}
                    <div className="clause-card-header">
                      <span className="clause-type-tag">{neg.clause_type.replace('_', ' ')}</span>
                      <span className={`badge ${statusColor[neg.status] ?? 'badge-info'}`}>
                        {statusLabel[neg.status] ?? neg.status}
                      </span>
                    </div>
                    <h4>{neg.clause_title}</h4>
                    {!isActive && (
                      <p className="clause-excerpt">{neg.original_content.slice(0, 140)}{neg.original_content.length > 140 ? '…' : ''}</p>
                    )}

                    {/* Expanded view */}
                    {isActive && (
                      <div className="clause-expanded" onClick={e => e.stopPropagation()}>

                        {/* Original clause */}
                        <div className="clause-original">
                          <span className="clause-original-label">Original Clause</span>
                          <p>{neg.original_content}</p>
                        </div>

                        {/* ── Negotiation conversation ── */}
                        <div className="neg-conversation">
                          {loadingProposals ? (
                            <div className="proposal-loading">
                              <span className="loading-spinner-sm" /> Loading…
                            </div>
                          ) : proposals.length === 0 ? (
                            <div className="neg-empty">
                              No edits proposed yet. Use the form below to suggest changes.
                            </div>
                          ) : (
                            <>
                              {/* Counter badge */}
                              {pendingCount > 0 && (
                                <div className="neg-counter-banner">
                                  {pendingCount} pending edit{pendingCount > 1 ? 's' : ''} awaiting response
                                </div>
                              )}

                              {/* Proposal messages */}
                              {proposals.map((p, idx) => {
                                const isMe = p.proposed_by === user?.id;
                                const roleName = p.proposed_by_user?.role ?? 'unknown';
                                const isLatest = idx === proposals.length - 1;
                                const resolved = p.status !== 'pending';

                                return (
                                  <div
                                    key={p.id}
                                    className={`neg-msg ${isMe ? 'neg-msg--mine' : 'neg-msg--theirs'} ${resolved ? 'neg-msg--resolved' : ''} ${isLatest ? 'neg-msg--latest' : ''}`}
                                  >
                                    <div className="neg-msg-meta">
                                      <span className={`neg-msg-author role-tag-${roleName}`}>
                                        {p.proposed_by_user?.full_name ?? 'Unknown'}
                                      </span>
                                      <span className="neg-msg-role">{roleName}</span>
                                      <span className="neg-msg-time">{timeAgo(p.created_at)}</span>
                                      {resolved && (
                                        <span className={`badge badge-sm ${p.status === 'accepted' ? 'badge-success' : p.status === 'rejected' ? 'badge-error' : 'badge-muted'}`}>
                                          {p.status === 'accepted' ? 'Agreed' : p.status === 'rejected' ? 'Rejected' : p.status}
                                        </span>
                                      )}
                                    </div>

                                    <div className="neg-msg-body">
                                      <span className="neg-msg-label">Proposed:</span>
                                      <blockquote>{p.proposed_content}</blockquote>
                                    </div>

                                    {p.comment && (
                                      <div className="neg-msg-comment">
                                        <span className="neg-msg-label">Note:</span> {p.comment}
                                      </div>
                                    )}

                                    {/* Actions — only on others' pending proposals */}
                                    {!isMe && p.status === 'pending' && (
                                      <div className="neg-msg-actions">
                                        <button className="btn-accept btn-sm" onClick={() => handleAcceptProposal(p.id)}>Accept</button>
                                        <button className="btn-reject btn-sm" onClick={() => handleRejectProposal(p.id)}>Reject</button>
                                        <button className="btn-secondary btn-sm" onClick={() => setProposalText(p.proposed_content)}>Counter</button>
                                      </div>
                                    )}
                                  </div>
                                );
                              })}
                              <div ref={conversationEndRef} />
                            </>
                          )}
                        </div>

                        {/* ── Propose an edit ── */}
                        <div className="clause-proposal-area">
                          <label className="proposal-label">
                            {proposals.length > 0 ? 'Reply / Counter-Propose' : 'Propose an Edit'}
                          </label>
                          <textarea
                            rows={3}
                            value={proposalText}
                            onChange={e => setProposalText(e.target.value)}
                            placeholder="Type your proposed clause wording…"
                          />
                          <textarea
                            rows={2}
                            className="proposal-comment-input"
                            value={commentText}
                            onChange={e => setCommentText(e.target.value)}
                            placeholder="Add a note or reason for this change (optional)"
                          />
                          <div className="form-actions">
                            <button className="btn-secondary btn-sm" onClick={() => { setActiveNeg(null); setProposals([]); setProposalText(''); setCommentText(''); }}>
                              Close
                            </button>
                            <button className="btn-primary btn-sm" onClick={handleCreateProposal} disabled={!proposalText.trim()}>
                              {proposals.length > 0 ? 'Counter' : 'Submit Proposal'}
                            </button>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </main>

        {/* ── RIGHT: CARGO ──────────────────────────────── */}
        <aside className="panel panel-cargo">
          <div className="panel-heading">
            <h3>📦 Cargo</h3>
            {canEditCargo && (
              <button className="btn-link" onClick={() => setEditingCargo(true)}>
                {cargoDetails ? 'Edit' : '+ Add'}
              </button>
            )}
          </div>

          {cargoDetails ? (
            <div className="details-view">
              <DetailRow label="Commodity" value={cargoDetails.commodity} />
              <DetailRow label="Quantity" value={cargoDetails.quantity ? `${cargoDetails.quantity.toLocaleString()} ${cargoDetails.quantity_unit}` : undefined} />
              <DetailRow label="Load Port" value={cargoDetails.load_port} />
              <DetailRow label="Discharge" value={cargoDetails.discharge_port} />
              <DetailRow label="Laycan" value={
                cargoDetails.laycan_from
                  ? `${cargoDetails.laycan_from.slice(0, 10)} – ${cargoDetails.laycan_to?.slice(0, 10) ?? '?'}`
                  : undefined
              } />
              <DetailRow
                label="Freight Idea"
                value={cargoDetails.freight_idea
                  ? `${cargoDetails.freight_currency} ${cargoDetails.freight_idea.toLocaleString()} / ${cargoDetails.freight_type.replace('_', ' ')}`
                  : undefined}
              />
              {cargoDetails.special_requirements && (
                <div className="detail-notes"><strong>Requirements:</strong> {cargoDetails.special_requirements}</div>
              )}
              {cargoDetails.notes && (
                <div className="detail-notes">{cargoDetails.notes}</div>
              )}
            </div>
          ) : (
            <div className="panel-empty">
              {canEditCargo
                ? 'No cargo details yet. Click + Add to fill in your cargo.'
                : 'Waiting for charterer to add cargo details.'}
            </div>
          )}
        </aside>

      </div>

      {/* ── VESSEL MODAL ──────────────────────────────── */}
      {editingVessel && (
        <div className="modal-overlay" onClick={() => setEditingVessel(false)}>
          <div className="modal-dialog" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3>⚓ {vesselDetails ? 'Edit Vessel Details' : 'Add Vessel Details'}</h3>
              <button className="modal-close" onClick={() => setEditingVessel(false)}>✕</button>
            </div>
            <div className="modal-body">
              <VesselEditForm
                dealId={id!}
                initial={vesselDetails}
                onSaved={d => { setVesselDetails(d); setEditingVessel(false); }}
                onCancel={() => setEditingVessel(false)}
              />
            </div>
          </div>
        </div>
      )}

      {/* ── CARGO MODAL ───────────────────────────────── */}
      {editingCargo && (
        <div className="modal-overlay" onClick={() => setEditingCargo(false)}>
          <div className="modal-dialog" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3>📦 {cargoDetails ? 'Edit Cargo Details' : 'Add Cargo Details'}</h3>
              <button className="modal-close" onClick={() => setEditingCargo(false)}>✕</button>
            </div>
            <div className="modal-body">
              <CargoEditForm
                dealId={id!}
                initial={cargoDetails}
                onSaved={d => { setCargoDetails(d); setEditingCargo(false); }}
                onCancel={() => setEditingCargo(false)}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
