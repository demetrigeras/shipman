import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { MapContainer, TileLayer, Marker, Popup, Polyline } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { api } from '../api/client';
import type { Voyage, ShipPosition, LaytimeEntry, LaytimeSummary, Document, ExtractedTerms, VoyagePayment, User } from '../api/client';
import NavBar from '../components/NavBar';
import EmbedWallet from '../components/EmbedWallet';

// Fix leaflet default marker icons in Vite
delete (L.Icon.Default.prototype as unknown as Record<string, unknown>)._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon-2x.png',
  iconUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png',
  shadowUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-shadow.png',
});

type Tab = 'negotiations' | 'charter' | 'tracking' | 'laytime' | 'payments';

const ACTIVITIES = [
  'NOR Tendered',
  'Loading Commenced',
  'Loading Completed',
  'Discharging Commenced',
  'Discharging Completed',
  'Anchoring / Waiting',
  'Weather Delay (excluded)',
  'Breakdown (excluded)',
  'Other',
];

function fmtNum(n?: number, decimals = 2) {
  if (n == null) return '—';
  return n.toLocaleString('en-US', { minimumFractionDigits: decimals, maximumFractionDigits: decimals });
}

export default function VoyageViewer() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [tab, setTab] = useState<Tab>('negotiations');
  const [paymentTab, setPaymentTab] = useState<'received' | 'paid' | 'all'>('all');
  const [nextPort, setNextPort] = useState('');
  const [voyage, setVoyage] = useState<Voyage | null>(null);
  const [positions, setPositions] = useState<ShipPosition[]>([]);
  const [laytimeEntries, setLaytimeEntries] = useState<LaytimeEntry[]>([]);
  const [laytimeSummary, setLaytimeSummary] = useState<LaytimeSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [editForm, setEditForm] = useState<Partial<Voyage>>({});
  // Charter Party tab
  const [documents, setDocuments] = useState<Document[]>([]);
  // docsLoading kept to track fetch state, used in charter tab effect
  const [, setDocsLoading] = useState(false);
  const [cpState, setCpState] = useState<'idle' | 'uploading' | 'extracting' | 'scanning' | 'done'>('idle');
  const [extracting, setExtracting] = useState(false);
  const [extractedTerms, setExtractedTerms] = useState<ExtractedTerms | null>(null);
  const [cpDocFilename, setCpDocFilename] = useState<string | null>(null);
  const [cpScanError, setCpScanError] = useState('');
  const cpFileRef = useRef<HTMLInputElement>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showPositionForm, setShowPositionForm] = useState(false);
  const [showLaytimeForm, setShowLaytimeForm] = useState(false);

  // Current user & role
  const [currentUser, setCurrentUser] = useState<User | null>(null);

  // Payments
  const [payments, setPayments] = useState<VoyagePayment[]>([]);
  const [showPaymentForm, setShowPaymentForm] = useState(false);
  const [payForm, setPayForm] = useState({ payment_type: 'hire', name: '', amount: '', recurring: false, interval: 'Month', frequency: 'Every', duration: 'Until Cancelled', durationCount: '' });
  const [payCreating, setPayCreating] = useState(false);
  const [copiedLink, setCopiedLink] = useState<string | null>(null);
  const [checkoutModal, setCheckoutModal] = useState<{ url: string; name: string } | null>(null);

  // Invite
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState<'shipowner' | 'charterer' | 'broker'>('charterer');
  const [inviteResult, setInviteResult] = useState<{ link: string; email_sent: boolean } | null>(null);
  const [inviteError, setInviteError] = useState('');
  const [inviteLinkCopied, setInviteLinkCopied] = useState(false);
  
  const [posForm, setPosForm] = useState({ latitude: '', longitude: '', speed_knots: '', heading: '', remarks: '' });
  const [laytimeForm, setLaytimeForm] = useState({ port_name: '', activity: ACTIVITIES[0], started_at: '', ended_at: '', remarks: '' });

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    Promise.all([
      api.voyages.get(id),
      api.auth.me(),
    ])
      .then(([v, u]) => { setVoyage(v); setEditForm(v); setCurrentUser(u); })
      .catch(() => navigate('/voyages'))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    if (!id || !voyage) return;
    if (tab === 'charter' && documents.length === 0) {
      setDocsLoading(true);
      api.documents.list(100, 0)
        .then(res => setDocuments(res.data))
        .catch(() => {})
        .finally(() => setDocsLoading(false));
    }
    if (tab === 'tracking') {
      api.voyages.listPositions(id).then(setPositions).catch(() => {});
    }
    if (tab === 'laytime') {
      api.voyages.listLaytime(id).then(setLaytimeEntries).catch(() => {});
      api.voyages.getLaytimeSummary(id).then(setLaytimeSummary).catch(() => {});
    }
    if (tab === 'payments') {
      api.voyages.listPayments(id).then(setPayments).catch(() => {});
    }
  }, [tab, voyage]);

  const handleAttachDoc = async (docId: string) => {
    if (!id) return;
    await api.voyages.attachDocument(id, docId);
    setVoyage(prev => prev ? { ...prev, document_id: docId } : prev);
    setEditForm(prev => ({ ...prev, document_id: docId }));
  };

  const handleCpUpload = async (file: File) => {
    if (!id) return;
    setCpScanError('');
    setExtractedTerms(null);
    try {
      // 1. Upload
      setCpState('uploading');
      const doc = await api.documents.upload(file);
      setCpDocFilename(file.name);

      // 2. Extract text
      setCpState('extracting');
      await api.documents.process(doc.id);
      await handleAttachDoc(doc.id);
      setDocuments(prev => [doc, ...prev]);

      // 3. Auto-scan for commercial terms
      setCpState('scanning');
      const terms = await api.voyages.extractTerms(id);
      setExtractedTerms(terms);

      // 4. Auto-apply terms to overview
      const patch: Partial<Voyage> = {};
      if (terms.vessel_name) patch.vessel_name = terms.vessel_name;
      if (terms.imo_number) patch.imo_number = terms.imo_number;
      if (terms.vessel_type) patch.vessel_type = terms.vessel_type;
      if (terms.dwt) patch.dwt = terms.dwt;
      if (terms.flag_state) patch.flag_state = terms.flag_state;
      if (terms.hire_rate) patch.hire_rate = terms.hire_rate;
      if (terms.freight_rate) patch.freight_rate = terms.freight_rate;
      if (terms.cargo_type) patch.cargo_type = terms.cargo_type;
      if (terms.cargo_quantity) patch.cargo_quantity = terms.cargo_quantity;
      if (terms.laytime_allowed_hours) patch.laytime_allowed_hours = terms.laytime_allowed_hours;
      if (terms.demurrage_rate) patch.demurrage_rate = terms.demurrage_rate;
      if (terms.despatch_rate) patch.despatch_rate = terms.despatch_rate;
      if (terms.currency) patch.demurrage_currency = terms.currency;

      if (Object.keys(patch).length > 0) {
        const updated = await api.voyages.update(id, patch);
        setVoyage(updated);
        setEditForm(updated);
      }

      setCpState('done');
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Upload or scan failed';
      setCpScanError(msg);
      setCpState('done');
    }
  };

  const handleRemoveCpDoc = async () => {
    if (!id) return;
    await api.voyages.update(id, { clear_document: true });
    setVoyage(prev => prev ? { ...prev, document_id: undefined } : prev);
    setEditForm(prev => ({ ...prev, document_id: undefined }));
    setCpState('idle');
    setCpDocFilename(null);
    setExtractedTerms(null);
    setCpScanError('');
    if (cpFileRef.current) cpFileRef.current.value = '';
  };

  const handleExtractTerms = async () => {
    if (!id) return;
    setExtracting(true);
    setExtractedTerms(null);
    setCpScanError('');
    try {
      const terms = await api.voyages.extractTerms(id);
      setExtractedTerms(terms);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Extraction failed';
      setCpScanError(msg);
    } finally {
      setExtracting(false);
    }
  };

  const handleApplyTerms = async () => {
    if (!id || !extractedTerms) return;
    const patch: Partial<Voyage> = {};
    if (extractedTerms.vessel_name) patch.vessel_name = extractedTerms.vessel_name;
    if (extractedTerms.imo_number) patch.imo_number = extractedTerms.imo_number;
    if (extractedTerms.vessel_type) patch.vessel_type = extractedTerms.vessel_type;
    if (extractedTerms.dwt) patch.dwt = extractedTerms.dwt;
    if (extractedTerms.flag_state) patch.flag_state = extractedTerms.flag_state;
    if (extractedTerms.hire_rate) patch.hire_rate = extractedTerms.hire_rate;
    if (extractedTerms.freight_rate) patch.freight_rate = extractedTerms.freight_rate;
    if (extractedTerms.cargo_type) patch.cargo_type = extractedTerms.cargo_type;
    if (extractedTerms.cargo_quantity) patch.cargo_quantity = extractedTerms.cargo_quantity;
    if (extractedTerms.laytime_allowed_hours) patch.laytime_allowed_hours = extractedTerms.laytime_allowed_hours;
    if (extractedTerms.demurrage_rate) patch.demurrage_rate = extractedTerms.demurrage_rate;
    if (extractedTerms.despatch_rate) patch.despatch_rate = extractedTerms.despatch_rate;
    if (extractedTerms.currency) patch.demurrage_currency = extractedTerms.currency;

    const updated = await api.voyages.update(id, patch);
    setVoyage(updated);
    setEditForm(updated);
    setExtractedTerms(null);
  };

  const handleSaveOverview = async () => {
    if (!id) return;
    setSaving(true);
    try {
      const updated = await api.voyages.update(id, editForm);
      setVoyage(updated);
      setEditForm(updated);
    } finally {
      setSaving(false);
    }
  };

  const handleSetStatus = async (status: string) => {
    if (!id) return;
    const updated = await api.voyages.update(id, { status });
    setVoyage(updated);
    setEditForm(updated);
  };

  const handleDelete = async () => {
    if (!id) return;
    setDeleting(true);
    try {
      await api.voyages.delete(id);
      navigate('/voyages');
    } catch {
      setDeleting(false);
      setShowDeleteConfirm(false);
    }
  };

  const handleAddPosition = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id) return;
    const pos = await api.voyages.addPosition(id, {
      recorded_at: new Date().toISOString(),
      latitude: parseFloat(posForm.latitude),
      longitude: parseFloat(posForm.longitude),
      speed_knots: posForm.speed_knots ? parseFloat(posForm.speed_knots) : undefined,
      heading: posForm.heading ? parseFloat(posForm.heading) : undefined,
      remarks: posForm.remarks || undefined,
    });
    setPositions(prev => [pos, ...prev]);
    setShowPositionForm(false);
    setPosForm({ latitude: '', longitude: '', speed_knots: '', heading: '', remarks: '' });
  };

  const handleAddLaytime = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id) return;
    const entry = await api.voyages.addLaytime(id, {
      port_name: laytimeForm.port_name,
      activity: laytimeForm.activity,
      started_at: laytimeForm.started_at,
      ended_at: laytimeForm.ended_at || undefined,
      remarks: laytimeForm.remarks || undefined,
    });
    setLaytimeEntries(prev => [...prev, entry]);
    const summary = await api.voyages.getLaytimeSummary(id);
    setLaytimeSummary(summary);
    setShowLaytimeForm(false);
    setLaytimeForm({ port_name: '', activity: ACTIVITIES[0], started_at: '', ended_at: '', remarks: '' });
  };

  const handleDeleteLaytime = async (entryId: string) => {
    if (!id) return;
    await api.voyages.deleteLaytime(id, entryId);
    setLaytimeEntries(prev => prev.filter(e => e.id !== entryId));
    const summary = await api.voyages.getLaytimeSummary(id);
    setLaytimeSummary(summary);
  };

  const handleCreatePayment = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id || !payForm.amount) return;
    setPayCreating(true);
    try {
      const p = await api.voyages.createPayment(id, {
        payment_type: payForm.payment_type,
        name: payForm.name || undefined,
        amount: parseFloat(payForm.amount),
        currency: 'USD',
      });
      setPayments(prev => [p, ...prev]);
      setShowPaymentForm(false);
      setPayForm({ payment_type: 'hire', name: '', amount: '', recurring: false, interval: 'Month', frequency: 'Every', duration: 'Until Cancelled', durationCount: '' });
    } finally {
      setPayCreating(false);
    }
  };

  const handleCopyCheckoutLink = (url: string, paymentId: string) => {
    navigator.clipboard.writeText(url);
    setCopiedLink(paymentId);
    setTimeout(() => setCopiedLink(null), 2000);
  };

  const handleDeletePayment = async (paymentId: string) => {
    if (!id) return;
    await api.voyages.deletePayment(id, paymentId);
    setPayments(prev => prev.filter(p => p.id !== paymentId));
  };

  const handleMarkPaid = async (paymentId: string) => {
    if (!id) return;
    const updated = await api.voyages.markPaid(id, paymentId);
    setPayments(prev => prev.map(p => p.id === paymentId ? updated : p));
  };

  const handleCreateInvite = async () => {
    if (!id || !inviteEmail.trim()) return;
    setInviteError('');
    try {
      const result = await api.voyages.createInvite(id, inviteEmail.trim(), inviteRole);
      setInviteResult({ link: result.invite_link, email_sent: result.email_sent });
    } catch (e) {
      setInviteError(e instanceof Error ? e.message : 'Failed to create invite');
    }
  };

  const handleGenerateSchedule = async () => {
    if (!id || !editForm.hire_rate || !editForm.payment_frequency) return;
    setPayCreating(true);
    try {
      await handleSaveOverview();

      const freq = editForm.payment_frequency;
      const daysPerPeriod = freq === 'semi_monthly' ? 15 : 30;
      const rate = editForm.hire_rate;
      const amount = rate * daysPerPeriod;
      const currency = editForm.demurrage_currency || 'USD';
      const totalVal = editForm.total_contract_value;
      const periods = totalVal ? Math.ceil(totalVal / amount) : 6;

      const firstDate = editForm.first_payment_date
        ? new Date(editForm.first_payment_date)
        : new Date();

      for (let i = 0; i < periods; i++) {
        const dueDate = new Date(firstDate);
        dueDate.setDate(dueDate.getDate() + (i * daysPerPeriod));
        const label = `Hire payment ${i + 1}/${periods} — due ${dueDate.toLocaleDateString()}`;
        await api.voyages.createPayment(id, {
          payment_type: 'hire',
          name: label,
          amount: i === periods - 1 && totalVal ? totalVal - (amount * (periods - 1)) : amount,
          currency,
        });
      }

      const updated = await api.voyages.listPayments(id);
      setPayments(updated);
      setTab('payments');
    } catch (e) {
      alert(e instanceof Error ? e.message : 'Failed to generate schedule');
    } finally {
      setPayCreating(false);
    }
  };

  

  

  if (loading) return (
    <div className="page-shell">
      <NavBar backTo="/voyages" backLabel="Voyages" />
      <div style={{ textAlign: 'center', padding: '4rem' }}>
        <div className="loading-spinner" style={{ margin: '0 auto' }} />
      </div>
    </div>
  );
  if (!voyage) return null;

  const statusOptions = ['planned', 'in_progress', 'completed', 'cancelled'];
  const fixtureTitle = voyage.voyage_number ?? voyage.vessel_name ?? 'Unnamed Fixture';
  const vesselSubtitle = voyage.voyage_number && voyage.vessel_name ? voyage.vessel_name : null;
  const charterTypeLabel: Record<string, string> = { time_charter: 'Time Charter', voyage_charter: 'Voyage Charter', bareboat: 'Bareboat' };

  const isOwner = currentUser?.id === voyage.owner_user_id;
  const isCounterparty = !isOwner && currentUser?.email === voyage.counterparty_email;
  type FixtureRole = 'shipowner' | 'charterer';
  const myRole: FixtureRole = isOwner ? 'shipowner' : isCounterparty ? 'charterer' : (currentUser?.role === 'charterer' ? 'charterer' : 'shipowner');
  const otherPartyLabel = myRole === 'shipowner' ? 'Charterer' : 'Shipowner';

  const mapPositions = [...positions].sort((a, b) => new Date(a.recorded_at).getTime() - new Date(b.recorded_at).getTime());
  const latLngs: [number, number][] = mapPositions.map(p => [p.latitude, p.longitude]);
  const latest = mapPositions[mapPositions.length - 1];

  return (
    <div className="page-shell">
      <NavBar backTo="/voyages" backLabel="Fixed C/P" />
      <div className="page-content">

        {/* Header */}
        <div className="voyage-viewer-header">
          <div>
            <h1 style={{ display: 'flex', alignItems: 'center', gap: '0.6rem' }}>
              {fixtureTitle}
              {voyage.charter_type && (
                <span className="voyage-type-pill" style={{ fontSize: '0.7rem', verticalAlign: 'middle' }}>
                  {charterTypeLabel[voyage.charter_type] ?? voyage.charter_type}
                </span>
              )}
            </h1>
            {vesselSubtitle && <p className="page-subtitle">🚢 {vesselSubtitle}{voyage.imo_number ? ` — IMO ${voyage.imo_number}` : ''}</p>}
          </div>
          <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
            <button
              className="btn-secondary btn-sm"
              onClick={() => { setShowInviteModal(true); setInviteResult(null); setInviteEmail(''); setInviteError(''); }}
              title="Invite counterparty to this fixture"
            >
              + Invite Party
            </button>
            <select
              className="field-input"
              style={{ width: 'auto', fontSize: '0.875rem' }}
              value={voyage.status}
              onChange={e => handleSetStatus(e.target.value)}
            >
              {statusOptions.map(s => (
                <option key={s} value={s}>{s.replace('_', ' ').replace(/\b\w/g, c => c.toUpperCase())}</option>
              ))}
            </select>
            <button
              className="btn-danger-outline btn-sm"
              onClick={() => setShowDeleteConfirm(true)}
              title="Delete fixture"
            >
              Delete
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="voyage-tabs">
          {(['negotiations', 'charter', 'tracking', 'laytime', 'payments'] as Tab[]).map(t => (
            <button
              key={t}
              className={`voyage-tab-btn${tab === t ? ' voyage-tab-btn--active' : ''}`}
              onClick={() => setTab(t)}
            >
              {t === 'negotiations' && '📋 Negotiations'}
              {t === 'charter' && <>📄 Charter Party {voyage.document_id && <span className="tab-dot-green">●</span>}</>}
              {t === 'tracking' && '🗺 Tracking'}
              {t === 'laytime' && '⏱ Laytime'}
              {t === 'payments' && <>💰 Payments {payments.length > 0 && <span className="tab-badge">{payments.length}</span>}</>}
            </button>
          ))}
        </div>

        {/* ── Negotiations Tab ─────────────────────────────── */}
        {tab === 'negotiations' && (
          <div className="voyage-tab-content">
            <div className="negotiations-empty">
              <div className="negotiations-empty-icon">📋</div>
              <h3>Clause Negotiations</h3>
              <p>
                Negotiate charter party clauses with your counterparty — proposals, counter-offers,
                and acceptance — all in one thread per clause.
              </p>
              <p className="hint">
                Upload your C/P in the <button className="link-inline" onClick={() => setTab('charter')}>Charter Party</button> tab and
                we'll auto-extract clauses you can negotiate here.
              </p>
              <div className="negotiations-cta">
                <span className="negotiations-cta-badge">Coming soon</span>
              </div>
            </div>
          </div>
        )}

        {/* ── Charter Party Tab ────────────────────────────── */}
        {tab === 'charter' && (
          <div className="voyage-tab-content">

            {/* Document upload / scan section */}
            <div className="cp-doc-strip">
              <input ref={cpFileRef} type="file" accept=".pdf,.txt,application/pdf" style={{ display: 'none' }}
                onChange={e => { const file = e.target.files?.[0]; if (file) handleCpUpload(file); e.target.value = ''; }} />

              {cpState === 'idle' && !voyage.document_id ? (
                <div className="cp-doc-strip-empty" onClick={() => cpFileRef.current?.click()}>
                  <span style={{ fontSize: '1.5rem' }}>📄</span>
                  <div>
                    <strong>Upload Charter Party</strong>
                    <p>Upload the C/P PDF to auto-extract contract details, or enter them manually below.</p>
                  </div>
                </div>
              ) : (cpState === 'uploading' || cpState === 'extracting' || cpState === 'scanning') ? (
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', padding: '0.75rem' }}>
                  <div className="loading-spinner" />
                  <span style={{ fontSize: '0.85rem' }}>
                    {cpState === 'uploading' && 'Uploading…'}
                    {cpState === 'extracting' && 'Extracting text…'}
                    {cpState === 'scanning' && 'Scanning for contract terms…'}
                  </span>
                </div>
              ) : (
                <div className="cp-doc-strip-attached">
                  <span>📄</span>
                  <div style={{ flex: 1 }}>
                    <strong>{cpDocFilename ?? 'Charter Party'}</strong>
                    <span style={{ color: '#10b981', fontSize: '0.75rem', marginLeft: '0.5rem' }}>✓ Attached</span>
                  </div>
                  <button className="btn-secondary btn-sm" onClick={handleExtractTerms} disabled={extracting}>
                    {extracting ? 'Scanning…' : 'Re-scan'}
                  </button>
                  <button className="btn-icon-sm" title="Replace" onClick={() => cpFileRef.current?.click()}>↑</button>
                  <button className="btn-icon-sm" title="Remove" style={{ color: '#ef4444', borderColor: '#ef4444' }} onClick={handleRemoveCpDoc}>✕</button>
                </div>
              )}
              {cpScanError && <p style={{ color: '#ef4444', fontSize: '0.78rem', padding: '0 0.75rem 0.5rem' }}>{cpScanError}</p>}
            </div>

            {/* Extracted terms preview (if just scanned) */}
            {extractedTerms && !extractedTerms.raw_summary && (
              <div className="cp-scan-result">
                <span style={{ fontSize: '0.82rem', fontWeight: 600 }}>Scanned terms found</span>
                <button className="btn-primary btn-sm" onClick={handleApplyTerms}>Apply to Contract Details</button>
              </div>
            )}

            {/* ── Editable contract detail lines ── */}
            <div className="cp-details-section">
              <h3 className="cp-details-heading">Contract Details</h3>
              <p className="hint" style={{ marginBottom: '1rem' }}>Edit any field. These are auto-populated when you scan a document.</p>

              <div className="cp-detail-lines">
                <div className="cp-detail-line"><label>Vessel Name</label><input className="field-input" value={editForm.vessel_name ?? ''} placeholder="MV Pacific Star" onChange={e => setEditForm(f => ({ ...f, vessel_name: e.target.value }))} /></div>
                <div className="cp-detail-line"><label>IMO Number</label><input className="field-input" value={editForm.imo_number ?? ''} placeholder="9234567" onChange={e => setEditForm(f => ({ ...f, imo_number: e.target.value }))} /></div>
                <div className="cp-detail-line"><label>Vessel Type</label><input className="field-input" value={editForm.vessel_type ?? ''} placeholder="Bulk Carrier" onChange={e => setEditForm(f => ({ ...f, vessel_type: e.target.value }))} /></div>
                <div className="cp-detail-line"><label>DWT (MT)</label><input className="field-input" type="number" value={editForm.dwt ?? ''} placeholder="75000" onChange={e => setEditForm(f => ({ ...f, dwt: e.target.value ? parseFloat(e.target.value) : undefined }))} /></div>
                <div className="cp-detail-line"><label>Flag State</label><input className="field-input" value={editForm.flag_state ?? ''} placeholder="Panama" onChange={e => setEditForm(f => ({ ...f, flag_state: e.target.value }))} /></div>
                <div className="cp-detail-line cp-line-divider" />
                <div className="cp-detail-line"><label>Counterparty</label><input className="field-input" value={editForm.counterparty_name ?? ''} placeholder="ABC Shipping Co." onChange={e => setEditForm(f => ({ ...f, counterparty_name: e.target.value }))} /></div>
                <div className="cp-detail-line"><label>Counterparty Email</label><input className="field-input" value={editForm.counterparty_email ?? ''} placeholder="ops@abc.com" onChange={e => setEditForm(f => ({ ...f, counterparty_email: e.target.value }))} /></div>
                <div className="cp-detail-line cp-line-divider" />
                <div className="cp-detail-line"><label>Load Port</label><input className="field-input" value={editForm.departure_port ?? ''} placeholder="Rotterdam" onChange={e => setEditForm(f => ({ ...f, departure_port: e.target.value }))} /></div>
                <div className="cp-detail-line"><label>Discharge Port</label><input className="field-input" value={editForm.arrival_port ?? ''} placeholder="Singapore" onChange={e => setEditForm(f => ({ ...f, arrival_port: e.target.value }))} /></div>
                <div className="cp-detail-line"><label>Cargo Type</label><input className="field-input" value={editForm.cargo_type ?? ''} placeholder="Iron Ore" onChange={e => setEditForm(f => ({ ...f, cargo_type: e.target.value }))} /></div>
                <div className="cp-detail-line"><label>Cargo Quantity (MT)</label><input className="field-input" type="number" value={editForm.cargo_quantity ?? ''} placeholder="50000" onChange={e => setEditForm(f => ({ ...f, cargo_quantity: e.target.value ? parseFloat(e.target.value) : undefined }))} /></div>
                <div className="cp-detail-line cp-line-divider" />
                <div className="cp-detail-line"><label>Currency</label>
                  <select className="field-input" value={editForm.demurrage_currency || 'USD'} onChange={e => setEditForm(f => ({ ...f, demurrage_currency: e.target.value }))}>
                    <option value="USD">USD</option><option value="EUR">EUR</option><option value="GBP">GBP</option>
                  </select>
                </div>
                <div className="cp-detail-line"><label>Hire Rate (/day)</label><input className="field-input" type="number" value={editForm.hire_rate ?? ''} placeholder="25000" onChange={e => setEditForm(f => ({ ...f, hire_rate: e.target.value ? parseFloat(e.target.value) : undefined }))} /></div>
                <div className="cp-detail-line"><label>Freight Rate (/MT)</label><input className="field-input" type="number" value={editForm.freight_rate ?? ''} placeholder="15.50" onChange={e => setEditForm(f => ({ ...f, freight_rate: e.target.value ? parseFloat(e.target.value) : undefined }))} /></div>
                <div className="cp-detail-line"><label>Total Contract Value</label><input className="field-input" type="number" value={editForm.total_contract_value ?? ''} placeholder="500000" onChange={e => setEditForm(f => ({ ...f, total_contract_value: e.target.value ? parseFloat(e.target.value) : undefined }))} /></div>
                <div className="cp-detail-line"><label>Payment Frequency</label>
                  <select className="field-input" value={editForm.payment_frequency ?? ''} onChange={e => setEditForm(f => ({ ...f, payment_frequency: e.target.value || undefined }))}>
                    <option value="">—</option><option value="monthly">Monthly</option><option value="semi_monthly">Semi-Monthly</option><option value="lump_sum">Lump Sum</option><option value="on_completion">On Completion</option>
                  </select>
                </div>
                <div className="cp-detail-line"><label>Commission (%)</label><input className="field-input" type="number" step="0.25" value={editForm.commission_rate ?? ''} placeholder="3.75" onChange={e => setEditForm(f => ({ ...f, commission_rate: e.target.value ? parseFloat(e.target.value) : undefined }))} /></div>
                <div className="cp-detail-line cp-line-divider" />
                <div className="cp-detail-line"><label>Laytime Allowed (hrs)</label><input className="field-input" type="number" value={editForm.laytime_allowed_hours ?? ''} placeholder="96" onChange={e => setEditForm(f => ({ ...f, laytime_allowed_hours: e.target.value ? parseFloat(e.target.value) : undefined }))} /></div>
                <div className="cp-detail-line"><label>Demurrage Rate (/day)</label><input className="field-input" type="number" value={editForm.demurrage_rate ?? ''} placeholder="25000" onChange={e => setEditForm(f => ({ ...f, demurrage_rate: e.target.value ? parseFloat(e.target.value) : undefined }))} /></div>
                <div className="cp-detail-line"><label>Despatch Rate (/day)</label><input className="field-input" type="number" value={editForm.despatch_rate ?? ''} placeholder="12500" onChange={e => setEditForm(f => ({ ...f, despatch_rate: e.target.value ? parseFloat(e.target.value) : undefined }))} /></div>
                <div className="cp-detail-line cp-line-divider" />
                <div className="cp-detail-line"><label>Notes</label><input className="field-input" value={editForm.notes ?? ''} placeholder="Additional notes…" onChange={e => setEditForm(f => ({ ...f, notes: e.target.value }))} /></div>
              </div>

              <button className="btn-primary" style={{ marginTop: '1.25rem' }} onClick={handleSaveOverview} disabled={saving}>
                {saving ? 'Saving…' : 'Save Contract Details'}
              </button>
            </div>
          </div>
        )}

        {/* ── Tracking Tab ─────────────────────────────────── */}
        {tab === 'tracking' && (
          <div className="voyage-tab-content">

            {/* IMO lookup strip */}
            {!voyage.imo_number ? (
              <div className="tracking-imo-strip">
                <span style={{ fontSize: '1.2rem' }}>🚢</span>
                <div style={{ flex: 1 }}>
                  <strong>Track your vessel</strong>
                  <p style={{ fontSize: '0.8rem', color: 'var(--color-text-secondary)', margin: 0 }}>
                    Enter the IMO number to view the vessel on MarineTraffic.
                  </p>
                </div>
                <input className="field-input" style={{ width: '160px' }} placeholder="IMO number" value={editForm.imo_number ?? ''}
                  onChange={e => setEditForm(f => ({ ...f, imo_number: e.target.value }))} />
                <button className="btn-primary btn-sm" onClick={async () => {
                  if (!id || !editForm.imo_number) return;
                  const updated = await api.voyages.update(id, { imo_number: editForm.imo_number });
                  setVoyage(updated); setEditForm(updated);
                }}>Save IMO</button>
              </div>
            ) : (
              <div className="tracking-imo-strip tracking-imo-strip--linked">
                <span style={{ fontSize: '1.2rem' }}>🚢</span>
                <div style={{ flex: 1 }}>
                  <strong>IMO {voyage.imo_number}</strong>
                  <a href={`https://www.marinetraffic.com/en/ais/details/ships/imo:${voyage.imo_number}`}
                    target="_blank" rel="noopener noreferrer" style={{ marginLeft: '0.75rem', fontSize: '0.82rem' }}>
                    View on MarineTraffic ↗
                  </a>
                </div>
                <button className="btn-secondary btn-sm" onClick={() => setShowPositionForm(true)}>+ Log Position</button>
              </div>
            )}

            {/* Map */}
            <div className="voyage-map-container">
              {latLngs.length === 0 ? (
                <div className="voyage-map-empty">
                  <div>🗺</div>
                  <p>No positions logged yet.<br />Log a position to see the vessel on the map.</p>
                </div>
              ) : (
                <MapContainer
                  center={latest ? [latest.latitude, latest.longitude] : [0, 0]}
                  zoom={5}
                  style={{ height: '100%', width: '100%', borderRadius: '8px' }}
                >
                  <TileLayer
                    attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
                    url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                  />
                  {latLngs.length > 1 && (
                    <Polyline positions={latLngs} color="#2563eb" weight={2} opacity={0.7} />
                  )}
                  {latest && (
                    <Marker position={[latest.latitude, latest.longitude]}>
                      <Popup>
                        <strong>{fixtureTitle}</strong><br />
                        {latest.speed_knots != null && <>Speed: {latest.speed_knots} kn<br /></>}
                        {latest.heading != null && <>Heading: {latest.heading}°<br /></>}
                        {new Date(latest.recorded_at).toLocaleString()}
                      </Popup>
                    </Marker>
                  )}
                </MapContainer>
              )}
            </div>

            {/* Port section under map */}
            <div className="tracking-port-section">
              <h4 style={{ margin: '0 0 0.5rem' }}>Ports</h4>
              <div className="tracking-port-grid">
                <div className="tracking-port-card">
                  <span className="tracking-port-label">Load Port</span>
                  <span className="tracking-port-value">{voyage.departure_port || '—'}</span>
                </div>
                <div className="tracking-port-card">
                  <span className="tracking-port-label">Discharge Port</span>
                  <span className="tracking-port-value">{voyage.arrival_port || '—'}</span>
                </div>
                <div className="tracking-port-card tracking-port-card--next">
                  <span className="tracking-port-label">Next Port</span>
                  <input className="field-input" value={nextPort} placeholder="Enter next port…" style={{ fontSize: '0.85rem' }}
                    onChange={e => setNextPort(e.target.value)} />
                </div>
              </div>
            </div>

            {/* Position history */}
            {positions.length > 0 && (
              <div style={{ marginTop: '1rem' }}>
                <h4 style={{ marginBottom: '0.5rem', fontSize: '0.875rem', color: 'var(--color-text-secondary)' }}>
                  Position History ({positions.length})
                </h4>
                <table className="voyage-table">
                  <thead>
                    <tr>
                      <th>Date/Time</th>
                      <th>Latitude</th>
                      <th>Longitude</th>
                      <th>Speed (kn)</th>
                      <th>Heading (°)</th>
                      <th>Source</th>
                    </tr>
                  </thead>
                  <tbody>
                    {positions.map(p => (
                      <tr key={p.id}>
                        <td>{new Date(p.recorded_at).toLocaleString()}</td>
                        <td>{p.latitude.toFixed(4)}</td>
                        <td>{p.longitude.toFixed(4)}</td>
                        <td>{p.speed_knots ?? '—'}</td>
                        <td>{p.heading ?? '—'}</td>
                        <td><span className="badge-source">{p.source}</span></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* ── Laytime Tab ─────────────────────────────────── */}
        {tab === 'laytime' && (
          <div className="voyage-tab-content">

            {/* Summary box */}
            {laytimeSummary && (
              <div className={`laytime-summary-box ${laytimeSummary.demurrage_hours > 0 ? 'laytime-summary--demurrage' : laytimeSummary.despatch_hours > 0 ? 'laytime-summary--despatch' : ''}`}>
                <div className="laytime-kpi-row">
                  <div className="laytime-kpi">
                    <span className="laytime-kpi-val">{fmtNum(laytimeSummary.total_hours_used, 2)} h</span>
                    <span className="laytime-kpi-label">Used</span>
                  </div>
                  <div className="laytime-kpi">
                    <span className="laytime-kpi-val">{fmtNum(laytimeSummary.total_hours_allowed, 2)} h</span>
                    <span className="laytime-kpi-label">Allowed</span>
                  </div>
                  <div className="laytime-kpi">
                    {laytimeSummary.demurrage_hours > 0 ? (
                      <>
                        <span className="laytime-kpi-val laytime-kpi--red">+{fmtNum(laytimeSummary.demurrage_hours, 2)} h</span>
                        <span className="laytime-kpi-label">Demurrage Time</span>
                      </>
                    ) : laytimeSummary.despatch_hours > 0 ? (
                      <>
                        <span className="laytime-kpi-val laytime-kpi--green">-{fmtNum(laytimeSummary.despatch_hours, 2)} h</span>
                        <span className="laytime-kpi-label">Despatch Time</span>
                      </>
                    ) : (
                      <>
                        <span className="laytime-kpi-val">0 h</span>
                        <span className="laytime-kpi-label">Balance</span>
                      </>
                    )}
                  </div>
                  {laytimeSummary.demurrage_amount != null && (
                    <div className="laytime-kpi">
                      <span className="laytime-kpi-val laytime-kpi--red">
                        {laytimeSummary.currency} {fmtNum(laytimeSummary.demurrage_amount, 0)}
                      </span>
                      <span className="laytime-kpi-label">Demurrage Owed</span>
                    </div>
                  )}
                  {laytimeSummary.despatch_amount != null && (
                    <div className="laytime-kpi">
                      <span className="laytime-kpi-val laytime-kpi--green">
                        {laytimeSummary.currency} {fmtNum(laytimeSummary.despatch_amount, 0)}
                      </span>
                      <span className="laytime-kpi-label">Despatch Earned</span>
                    </div>
                  )}
                </div>
                {voyage.laytime_allowed_hours == null && (
                  <p className="laytime-no-terms">⚠ Set laytime allowed hours and demurrage rate in Overview to see financial calculations.</p>
                )}
              </div>
            )}

            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', margin: '1.25rem 0 0.75rem' }}>
              <h3 style={{ margin: 0 }}>Time Log</h3>
              <button className="btn-secondary" onClick={() => setShowLaytimeForm(true)}>+ Log Event</button>
            </div>

            {laytimeEntries.length === 0 ? (
              <div className="laytime-empty">
                <p>No events logged. Start by logging the NOR tendering time.</p>
              </div>
            ) : (
              <table className="voyage-table">
                <thead>
                  <tr>
                    <th>Port</th>
                    <th>Activity</th>
                    <th>Start</th>
                    <th>End</th>
                    <th>Hours</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {laytimeEntries.map(e => (
                    <tr key={e.id}>
                      <td>{e.port_name}</td>
                      <td>
                        <span className={`laytime-activity ${e.activity.includes('excluded') ? 'laytime-activity--excluded' : ''}`}>
                          {e.activity}
                        </span>
                      </td>
                      <td>{new Date(e.started_at).toLocaleString()}</td>
                      <td>{e.ended_at ? new Date(e.ended_at).toLocaleString() : '—'}</td>
                      <td>{e.hours_counted != null ? fmtNum(e.hours_counted, 2) : '—'}</td>
                      <td>
                        <button className="btn-icon-danger" title="Delete" onClick={() => handleDeleteLaytime(e.id)}>✕</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}

        {/* ── Payments Tab ─────────────────────────────────── */}
        {tab === 'payments' && (() => {
          const typeLabels: Record<string, string> = {
            hire: 'Hire', freight: 'Freight', demurrage: 'Demurrage', despatch: 'Despatch',
            bunker: 'Bunker', port_charges: 'Port Charges', other: 'Other',
          };
          const statusColor: Record<string, string> = {
            draft: '#6366f1', pending: '#f59e0b', completed: '#10b981', failed: '#ef4444', cancelled: '#64748b',
          };

          const isMine = (p: VoyagePayment) => currentUser && p.created_by === currentUser.id;
          const sentInvoices = payments.filter(p => isMine(p));
          const receivedInvoices = payments.filter(p => !isMine(p));
          const completedPayments = payments.filter(p => p.status === 'completed' || p.status === 'failed');

          const totalReceived = completedPayments.filter(p => !isMine(p)).reduce((s, p) => s + p.amount, 0);
          const totalPaid = completedPayments.filter(p => isMine(p)).reduce((s, p) => s + p.amount, 0);
          const historyFiltered = paymentTab === 'received' ? completedPayments.filter(p => !isMine(p))
            : paymentTab === 'paid' ? completedPayments.filter(p => isMine(p))
            : completedPayments;

          const renderInvoiceCard = (p: VoyagePayment, isSent: boolean) => {
            const color = statusColor[p.status] ?? '#64748b';
            const recipientName = isSent ? (voyage.counterparty_name || null) : (myRole === 'charterer' ? voyage.vessel_name || null : null);
            const recipientEmail = isSent ? (p.recipient_email || voyage.counterparty_email || null) : null;
            return (
              <div key={p.id} className={`pay-invoice-card${isSent ? '' : ' pay-invoice-card--received'}`}>
                <div className="pay-invoice-left">
                  <span className="payment-type-tag">{typeLabels[p.payment_type] ?? p.payment_type}</span>
                  <div className="pay-invoice-details">
                    <strong>{p.description || 'Untitled'}</strong>
                    <span className="pay-invoice-meta">
                      USD {p.amount.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                      <span className="pay-invoice-dot">·</span>
                      {new Date(p.created_at).toLocaleDateString()}
                    </span>
                    {isSent && (recipientName || recipientEmail) && (
                      <span className="pay-invoice-recipient">
                        {recipientName && <span className="pay-recipient-name">{recipientName}</span>}
                        {recipientEmail && <span className="pay-recipient-email">{recipientEmail}</span>}
                      </span>
                    )}
                  </div>
                </div>
                <div className="pay-invoice-right">
                  <span className="payment-status-badge" style={{ background: color + '1a', color }}>{p.status}</span>
                  <div className="pay-invoice-actions">
                    {isSent && p.coinsub_checkout_url && (
                      <button className="btn-secondary btn-sm"
                        onClick={() => handleCopyCheckoutLink(p.coinsub_checkout_url!, p.id)}>
                        {copiedLink === p.id ? '✓ Copied' : 'Share Link'}
                      </button>
                    )}
                    {isSent && p.status !== 'completed' && (
                      <button className="btn-secondary btn-sm" title="Mark as paid manually"
                        onClick={() => handleMarkPaid(p.id)}>
                        Mark Paid
                      </button>
                    )}
                    {isSent && (
                      <button className="btn-icon-danger" title="Delete" onClick={() => handleDeletePayment(p.id)}>✕</button>
                    )}
                    {!isSent && p.coinsub_checkout_url && p.status !== 'completed' && (
                      <button className="btn-primary btn-sm"
                        onClick={() => setCheckoutModal({ url: p.coinsub_checkout_url!, name: p.description || 'Payment' })}>
                        Pay
                      </button>
                    )}
                  </div>
                </div>
              </div>
            );
          };

          return (
          <div className="voyage-tab-content">

            {/* ━━ EMBEDDED WALLET (RocketRamp) ━━━━━━━━━━━━━━━━━━ */}
            {voyage.counterparty_email ? (
              <div className="pay-wallet-card">
                <div className="pay-wallet-header">
                  <div>
                    <div className="pay-wallet-label">Send Funds</div>
                    <div className="pay-wallet-desc">
                      Wallet prefilled for <strong>{voyage.counterparty_name || voyage.counterparty_email}</strong>. Sign in to your RocketRamp wallet below to send.
                    </div>
                  </div>
                </div>
                <EmbedWallet
                  recipientEmail={voyage.counterparty_email}
                  memo={`Voyage ${voyage.voyage_number || voyage.id.slice(0, 8)}`}
                  height={760}
                />
              </div>
            ) : (
              <div className="pay-wallet-card pay-wallet-card--empty">
                <div className="pay-wallet-label">Send Funds</div>
                <p className="pay-wallet-desc">
                  Add a counterparty to this voyage to enable the embedded wallet for direct payments.
                </p>
              </div>
            )}

            {/* ━━ TWO-COLUMN INVOICES ━━━━━━━━━━━━━━━━━━━━━━━━━━ */}
            <div className="pay-invoices-grid">
              {/* Sent by you */}
              <div className="pay-col">
                <div className="pay-col-header">
                  <div className="pay-col-identity">
                    <h3>Sent</h3>
                    <span className="pay-col-count">{sentInvoices.length}</span>
                  </div>
                  <div className="pay-col-actions">
                    {voyage.hire_rate && voyage.payment_frequency && voyage.payment_frequency !== 'lump_sum' && voyage.payment_frequency !== 'on_completion' && (
                      <button className="btn-secondary btn-sm" onClick={handleGenerateSchedule} disabled={payCreating}>
                        Generate
                      </button>
                    )}
                    <button className="btn-primary btn-sm" onClick={() => setShowPaymentForm(true)}>+ Invoice</button>
                  </div>
                </div>
                <div className="pay-col-who">
                  <span className="pay-col-role">{myRole === 'shipowner' ? 'Shipowner' : 'Charterer'}</span>
                  <span className="pay-col-email">{currentUser?.email}</span>
                </div>
                {sentInvoices.length === 0 ? (
                  <div className="pay-section-empty">No invoices sent yet</div>
                ) : (
                  <div className="pay-invoice-list">
                    {sentInvoices.map(p => renderInvoiceCard(p, true))}
                  </div>
                )}
              </div>

              {/* Received */}
              <div className="pay-col">
                <div className="pay-col-header">
                  <div className="pay-col-identity">
                    <h3>Received</h3>
                    <span className="pay-col-count">{receivedInvoices.length}</span>
                  </div>
                </div>
                <div className="pay-col-who">
                  <span className="pay-col-role">{otherPartyLabel}</span>
                  <span className="pay-col-email">{myRole === 'shipowner' ? (voyage.counterparty_email || '—') : (voyage.owner_user_id ? 'Owner' : '—')}</span>
                </div>
                {receivedInvoices.length === 0 ? (
                  <div className="pay-section-empty">No invoices received yet</div>
                ) : (
                  <div className="pay-invoice-list">
                    {receivedInvoices.map(p => renderInvoiceCard(p, false))}
                  </div>
                )}
              </div>
            </div>

            {/* ━━ PAYMENT HISTORY ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ */}
            <div className="pay-section" style={{ marginTop: '1.5rem' }}>
              <div className="pay-section-header">
                <h3>Payment History</h3>
              </div>

              <div className="pay-summary-row">
                <div className="pay-summary-card pay-summary-card--in">
                  <span className="pay-summary-label">Received</span>
                  <strong>USD {totalReceived.toLocaleString(undefined, { minimumFractionDigits: 2 })}</strong>
                </div>
                <div className="pay-summary-card pay-summary-card--out">
                  <span className="pay-summary-label">Paid</span>
                  <strong>USD {totalPaid.toLocaleString(undefined, { minimumFractionDigits: 2 })}</strong>
                </div>
              </div>

              <div className="pay-tabs">
                {([
                  ['all', 'All', completedPayments.length],
                  ['received', 'Received', completedPayments.filter(p => !isMine(p)).length],
                  ['paid', 'Paid', completedPayments.filter(p => isMine(p)).length],
                ] as ['all' | 'received' | 'paid', string, number][]).map(([key, label, count]) => (
                  <button key={key} className={`pay-tab${paymentTab === key ? ' pay-tab--active' : ''}`}
                    onClick={() => setPaymentTab(key)}>
                    {label}{count > 0 && <span className="pay-tab-count">{count}</span>}
                  </button>
                ))}
              </div>

              {historyFiltered.length === 0 ? (
                <div className="pay-section-empty">No completed payments yet</div>
              ) : (
                <table className="voyage-table pay-table">
                  <thead>
                    <tr>
                      <th>Type</th>
                      <th>Name</th>
                      <th style={{ textAlign: 'right' }}>Amount</th>
                      <th>From</th>
                      <th>Date</th>
                      <th>TX</th>
                    </tr>
                  </thead>
                  <tbody>
                    {historyFiltered.map(p => {
                      const incoming = !isMine(p);
                      return (
                        <tr key={p.id}>
                          <td><span className="payment-type-tag">{typeLabels[p.payment_type] ?? p.payment_type}</span></td>
                          <td style={{ maxWidth: '200px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{p.description || '—'}</td>
                          <td style={{ textAlign: 'right', fontWeight: 600, color: incoming ? '#10b981' : '#ef4444' }}>
                            {incoming ? '+' : '−'} USD {p.amount.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                          </td>
                          <td style={{ fontSize: '0.78rem', color: 'var(--color-text-secondary)' }}>
                            {incoming
                              ? (voyage.counterparty_name || otherPartyLabel)
                              : (voyage.counterparty_name || otherPartyLabel)}
                            {incoming && voyage.counterparty_email && (
                              <span style={{ display: 'block', fontSize: '0.72rem', opacity: 0.7 }}>{voyage.counterparty_email}</span>
                            )}
                          </td>
                          <td style={{ fontSize: '0.78rem' }}>{p.paid_at ? new Date(p.paid_at).toLocaleDateString() : new Date(p.created_at).toLocaleDateString()}</td>
                          <td style={{ fontSize: '0.75rem' }}>
                            {p.coinsub_tx_hash ? (
                              <a href={`https://polygonscan.com/tx/${p.coinsub_tx_hash}`} target="_blank" rel="noopener noreferrer">
                                {p.coinsub_tx_hash.slice(0, 8)}…
                              </a>
                            ) : '—'}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              )}
            </div>
          </div>
          );
        })()}
      </div>

      {/* Payment Form Modal */}
      {showPaymentForm && (
        <div className="modal-backdrop" onClick={() => setShowPaymentForm(false)}>
          <div className="modal-box" onClick={e => e.stopPropagation()} style={{ maxWidth: 440 }}>
            <h3 style={{ marginBottom: '1rem' }}>Create Invoice</h3>
            <form onSubmit={handleCreatePayment}>
              <label className="field-label">Payment Type *</label>
              <select className="field-input" value={payForm.payment_type}
                onChange={e => setPayForm(f => ({ ...f, payment_type: e.target.value }))}>
                <option value="hire">Hire</option>
                <option value="freight">Freight</option>
                <option value="demurrage">Demurrage</option>
                <option value="despatch">Despatch</option>
                <option value="bunker">Bunker</option>
                <option value="port_charges">Port Charges</option>
                <option value="other">Other</option>
              </select>

              <label className="field-label" style={{ marginTop: '0.75rem' }}>Name *</label>
              <input className="field-input" required placeholder="e.g. Monthly hire — MV Pacific Star"
                value={payForm.name}
                onChange={e => setPayForm(f => ({ ...f, name: e.target.value }))} />
              <p className="hint" style={{ fontSize: '0.72rem', marginTop: '0.2rem' }}>Shown to the payer on the checkout screen.</p>

              <label className="field-label" style={{ marginTop: '0.75rem' }}>Amount (USD) *</label>
              <input className="field-input" type="number" step="0.01" required placeholder="25000.00"
                value={payForm.amount}
                onChange={e => setPayForm(f => ({ ...f, amount: e.target.value }))} />

              {/* Recurring payment toggle */}
              <div className="pay-recurring-section" style={{ marginTop: '1rem' }}>
                <label className="pay-recurring-toggle">
                  <input type="checkbox" checked={payForm.recurring}
                    onChange={e => setPayForm(f => ({ ...f, recurring: e.target.checked }))} />
                  <span>Recurring payment</span>
                </label>
                <p className="hint" style={{ fontSize: '0.72rem', marginTop: '0.15rem' }}>
                  Enable for subscriptions like monthly hire payments. The payer will be charged automatically.
                </p>

                {payForm.recurring && (
                  <div className="pay-schedule-row" style={{ marginTop: '0.6rem' }}>
                    <span className="pay-schedule-label">Charge</span>
                    <select className="field-input pay-schedule-select" value={payForm.frequency}
                      onChange={e => setPayForm(f => ({ ...f, frequency: e.target.value }))}>
                      <option value="Every">every</option>
                      <option value="Every Other">every other</option>
                      <option value="Every Third">every 3rd</option>
                      <option value="Every Fourth">every 4th</option>
                      <option value="Every Fifth">every 5th</option>
                      <option value="Every Sixth">every 6th</option>
                      <option value="Every Seventh">every 7th</option>
                    </select>
                    <select className="field-input pay-schedule-select" value={payForm.interval}
                      onChange={e => setPayForm(f => ({ ...f, interval: e.target.value }))}>
                      <option value="Day">day</option>
                      <option value="Week">week</option>
                      <option value="Month">month</option>
                      <option value="Year">year</option>
                    </select>
                    <span className="pay-schedule-label">,</span>
                    <select className="field-input pay-schedule-select" value={payForm.duration}
                      onChange={e => setPayForm(f => ({ ...f, duration: e.target.value, durationCount: e.target.value === 'Until Cancelled' ? '' : f.durationCount }))}>
                      <option value="Until Cancelled">until cancelled</option>
                      <option value="fixed">for</option>
                    </select>
                    {payForm.duration === 'fixed' && (
                      <>
                        <input className="field-input pay-schedule-count" type="number" min="1" placeholder="#"
                          value={payForm.durationCount}
                          onChange={e => setPayForm(f => ({ ...f, durationCount: e.target.value }))} />
                        <span className="pay-schedule-label">payments</span>
                      </>
                    )}
                  </div>
                )}
              </div>

              <div className="modal-actions" style={{ marginTop: '1.25rem' }}>
                <button type="button" className="btn-secondary" onClick={() => setShowPaymentForm(false)}>Cancel</button>
                <button type="submit" className="btn-primary" disabled={payCreating || !payForm.amount || !payForm.name}>
                  {payCreating ? 'Creating…' : payForm.recurring ? 'Create Recurring' : 'Create'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Invite Party Modal */}
      {showInviteModal && (
        <div className="modal-backdrop" onClick={() => setShowInviteModal(false)}>
          <div className="modal-box" onClick={e => e.stopPropagation()} style={{ maxWidth: 460 }}>
            <h3 style={{ marginBottom: '1rem' }}>Invite Party to Fixture</h3>

            {!inviteResult ? (
              <>
                <label className="field-label">Their Email Address</label>
                <input
                  className="field-input"
                  type="email"
                  placeholder="counterparty@company.com"
                  value={inviteEmail}
                  autoFocus
                  onChange={e => setInviteEmail(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleCreateInvite()}
                />

                <label className="field-label" style={{ marginTop: '0.75rem' }}>Their Role</label>
                <select
                  className="field-input"
                  value={inviteRole}
                  onChange={e => setInviteRole(e.target.value as typeof inviteRole)}
                >
                  <option value="charterer">Charterer</option>
                  <option value="shipowner">Ship Owner</option>
                  <option value="broker">Broker</option>
                </select>

                {inviteError && (
                  <p style={{ color: '#ef4444', fontSize: '0.8rem', marginTop: '0.5rem' }}>{inviteError}</p>
                )}

                <div className="modal-actions" style={{ marginTop: '1.25rem' }}>
                  <button className="btn-secondary" onClick={() => setShowInviteModal(false)}>Cancel</button>
                  <button className="btn-primary" onClick={handleCreateInvite} disabled={!inviteEmail.trim()}>
                    Send Invite
                  </button>
                </div>
              </>
            ) : (
              <div style={{ textAlign: 'center', padding: '0.5rem 0' }}>
                <div style={{ fontSize: '2.5rem', marginBottom: '0.75rem' }}>✉️</div>
                {inviteResult.email_sent ? (
                  <>
                    <h4 style={{ marginBottom: '0.4rem' }}>Invite Sent!</h4>
                    <p style={{ color: 'var(--color-text-secondary)', fontSize: '0.875rem' }}>
                      An email with a join link has been sent to <strong>{inviteEmail}</strong>.
                    </p>
                  </>
                ) : (
                  <>
                    <h4 style={{ marginBottom: '0.4rem' }}>Invite Created</h4>
                    <p style={{ color: 'var(--color-text-secondary)', fontSize: '0.875rem', marginBottom: '0.75rem' }}>
                      Copy this link and send it to <strong>{inviteEmail}</strong>:
                    </p>
                    <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'stretch' }}>
                      <input
                        className="field-input"
                        readOnly
                        value={inviteResult.link}
                        style={{ fontSize: '0.75rem', flex: 1 }}
                        onClick={e => (e.target as HTMLInputElement).select()}
                      />
                      <button
                        className="btn-primary btn-sm"
                        onClick={() => {
                          navigator.clipboard.writeText(inviteResult!.link);
                          setInviteLinkCopied(true);
                          setTimeout(() => setInviteLinkCopied(false), 2000);
                        }}
                      >
                        {inviteLinkCopied ? '✓ Copied' : 'Copy'}
                      </button>
                    </div>
                    <p style={{ fontSize: '0.72rem', color: 'var(--color-text-secondary)', marginTop: '0.6rem' }}>
                      Set <code>SENDGRID_API_KEY</code> and <code>SENDGRID_TEMPLATE_ID</code> on the backend to send invites by email automatically.
                    </p>
                  </>
                )}
                <button
                  className="btn-primary"
                  style={{ marginTop: '1.25rem', width: '100%' }}
                  onClick={() => { setShowInviteModal(false); setInviteResult(null); setInviteEmail(''); setInviteLinkCopied(false); }}
                >
                  Done
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <div className="modal-backdrop" onClick={() => setShowDeleteConfirm(false)}>
          <div className="modal-box" onClick={e => e.stopPropagation()} style={{ maxWidth: 400 }}>
            <h3 style={{ marginBottom: '0.75rem' }}>Delete Fixture?</h3>
            <p style={{ color: 'var(--color-text-secondary)', fontSize: '0.9rem', marginBottom: '1.25rem' }}>
              This will permanently delete <strong>{fixtureTitle}</strong> and all its tracking data, laytime entries, and payment records. This cannot be undone.
            </p>
            <div className="modal-actions">
              <button className="btn-secondary" onClick={() => setShowDeleteConfirm(false)}>Cancel</button>
              <button className="btn-danger" onClick={handleDelete} disabled={deleting}>
                {deleting ? 'Deleting…' : 'Delete Fixture'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Coinsub Checkout Modal */}
      {checkoutModal && (
        <div className="modal-backdrop" onClick={() => setCheckoutModal(null)}>
          <div className="checkout-modal" onClick={e => e.stopPropagation()}>
            <div className="checkout-modal-header">
              <strong>{checkoutModal.name}</strong>
              <button className="btn-icon" onClick={() => setCheckoutModal(null)}>✕</button>
            </div>
            <iframe
              src={checkoutModal.url}
              title="Coinsub Checkout"
              className="checkout-iframe"
            />
          </div>
        </div>
      )}

      {/* Position Modal */}
      {showPositionForm && (
        <div className="modal-backdrop" onClick={() => setShowPositionForm(false)}>
          <div className="modal-box" onClick={e => e.stopPropagation()} style={{ maxWidth: 440 }}>
            <h3 style={{ marginBottom: '1rem' }}>Log Position</h3>
            <form onSubmit={handleAddPosition}>
              <div className="form-row-2">
                <div>
                  <label className="field-label">Latitude *</label>
                  <input className="field-input" type="number" step="any" required
                    placeholder="e.g. 51.5074" value={posForm.latitude}
                    onChange={e => setPosForm(f => ({ ...f, latitude: e.target.value }))} />
                </div>
                <div>
                  <label className="field-label">Longitude *</label>
                  <input className="field-input" type="number" step="any" required
                    placeholder="e.g. -0.1278" value={posForm.longitude}
                    onChange={e => setPosForm(f => ({ ...f, longitude: e.target.value }))} />
                </div>
              </div>
              <div className="form-row-2" style={{ marginTop: '0.75rem' }}>
                <div>
                  <label className="field-label">Speed (knots)</label>
                  <input className="field-input" type="number" step="any" value={posForm.speed_knots}
                    onChange={e => setPosForm(f => ({ ...f, speed_knots: e.target.value }))} />
                </div>
                <div>
                  <label className="field-label">Heading (°)</label>
                  <input className="field-input" type="number" step="any" value={posForm.heading}
                    onChange={e => setPosForm(f => ({ ...f, heading: e.target.value }))} />
                </div>
              </div>
              <label className="field-label" style={{ marginTop: '0.75rem' }}>Remarks</label>
              <input className="field-input" value={posForm.remarks}
                onChange={e => setPosForm(f => ({ ...f, remarks: e.target.value }))} />
              <div className="modal-actions" style={{ marginTop: '1rem' }}>
                <button type="button" className="btn-secondary" onClick={() => setShowPositionForm(false)}>Cancel</button>
                <button type="submit" className="btn-primary">Save Position</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Laytime Event Modal */}
      {showLaytimeForm && (
        <div className="modal-backdrop" onClick={() => setShowLaytimeForm(false)}>
          <div className="modal-box" onClick={e => e.stopPropagation()} style={{ maxWidth: 480 }}>
            <h3 style={{ marginBottom: '1rem' }}>Log Laytime Event</h3>
            <form onSubmit={handleAddLaytime}>
              <label className="field-label">Port *</label>
              <input className="field-input" required placeholder="e.g. Port Elizabeth"
                value={laytimeForm.port_name}
                onChange={e => setLaytimeForm(f => ({ ...f, port_name: e.target.value }))} />

              <label className="field-label" style={{ marginTop: '0.75rem' }}>Activity *</label>
              <select className="field-input" value={laytimeForm.activity}
                onChange={e => setLaytimeForm(f => ({ ...f, activity: e.target.value }))}>
                {ACTIVITIES.map(a => <option key={a}>{a}</option>)}
              </select>

              <div className="form-row-2" style={{ marginTop: '0.75rem' }}>
                <div>
                  <label className="field-label">Start *</label>
                  <input className="field-input" type="datetime-local" required
                    value={laytimeForm.started_at}
                    onChange={e => setLaytimeForm(f => ({ ...f, started_at: e.target.value }))} />
                </div>
                <div>
                  <label className="field-label">End (leave blank if ongoing)</label>
                  <input className="field-input" type="datetime-local"
                    value={laytimeForm.ended_at}
                    onChange={e => setLaytimeForm(f => ({ ...f, ended_at: e.target.value }))} />
                </div>
              </div>

              <label className="field-label" style={{ marginTop: '0.75rem' }}>Remarks</label>
              <input className="field-input" value={laytimeForm.remarks}
                onChange={e => setLaytimeForm(f => ({ ...f, remarks: e.target.value }))} />

              <p style={{ fontSize: '0.78rem', color: 'var(--color-text-secondary)', marginTop: '0.5rem' }}>
                Hours will be auto-calculated from start/end. Activities marked "(excluded)" don't count toward laytime.
              </p>

              <div className="modal-actions" style={{ marginTop: '1rem' }}>
                <button type="button" className="btn-secondary" onClick={() => setShowLaytimeForm(false)}>Cancel</button>
                <button type="submit" className="btn-primary">Log Event</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
