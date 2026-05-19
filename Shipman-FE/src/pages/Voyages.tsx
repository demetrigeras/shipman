import { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import type { Voyage, Document, ExtractedTerms } from '../api/client';
import NavBar from '../components/NavBar';

const STATUS_COLOR: Record<string, string> = {
  planned:     '#6366f1',
  in_progress: '#10b981',
  completed:   '#64748b',
  cancelled:   '#ef4444',
};
const STATUS_LABEL: Record<string, string> = {
  planned: 'Planned', in_progress: 'Active', completed: 'Completed', cancelled: 'Cancelled',
};
const CHARTER_TYPE_LABEL: Record<string, string> = {
  time_charter: 'TC', voyage_charter: 'VC', bareboat: 'BB',
};
const CHARTER_TYPE_FULL: Record<string, string> = {
  time_charter: 'Time Charter', voyage_charter: 'Voyage Charter', bareboat: 'Bareboat',
};
const CHARTER_TYPE_ICON: Record<string, string> = {
  time_charter: '\u23F1', voyage_charter: '\u26F5', bareboat: '\uD83D\uDEF3\uFE0F',
};
const CHARTER_TYPES = ['time_charter', 'voyage_charter', 'bareboat'] as const;

type CreateStep = 'upload' | 'scanning' | 'review';

export default function Voyages() {
  const navigate = useNavigate();
  const fileRef = useRef<HTMLInputElement>(null);

  const [fixtures, setFixtures] = useState<Voyage[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);

  const [step, setStep] = useState<CreateStep>('upload');
  const [name, setName] = useState('');
  const [charterType, setCharterType] = useState('time_charter');
  const [vesselName, setVesselName] = useState('');
  const [imoNumber, setImoNumber] = useState('');
  const [hireRate, setHireRate] = useState('');
  const [freightRate, setFreightRate] = useState('');
  const [cargoType, setCargoType] = useState('');
  const [cargoQty, setCargoQty] = useState('');
  const [loadPort, setLoadPort] = useState('');
  const [dischargePort, setDischargePort] = useState('');
  const [laytimeHours, setLaytimeHours] = useState('');
  const [demurrageRate, setDemurrageRate] = useState('');
  const [despatchRate, setDespatchRate] = useState('');
  const [currency, setCurrency] = useState('USD');
  const [uploadedDoc, setUploadedDoc] = useState<Document | null>(null);
  const [uploadProgress, setUploadProgress] = useState(false);
  const [scanError, setScanError] = useState('');
  const [createError, setCreateError] = useState('');
  const [extractedTerms, setExtractedTerms] = useState<ExtractedTerms | null>(null);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    api.voyages.list()
      .then(setFixtures)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const openCreate = () => {
    setStep('upload');
    setName('');
    setCharterType('time_charter');
    setVesselName('');
    setImoNumber('');
    setHireRate('');
    setFreightRate('');
    setCargoType('');
    setCargoQty('');
    setLoadPort('');
    setDischargePort('');
    setLaytimeHours('');
    setDemurrageRate('');
    setDespatchRate('');
    setCurrency('USD');
    setUploadedDoc(null);
    setExtractedTerms(null);
    setScanError('');
    setCreateError('');
    setShowCreate(true);
  };

  /** Upload + process PDF, then scan terms — no temporary voyage (avoids create/delete races). */
  const handleUpload = async (file: File) => {
    setScanError('');
    setUploadProgress(true);
    try {
      const doc = await api.documents.upload(file);
      await api.documents.process(doc.id);
      setUploadedDoc(doc);
      setStep('scanning');
      try {
        const terms = await api.voyages.extractTermsPreview(doc.id);
        setExtractedTerms(terms);
        if (terms.vessel_name && !vesselName) setVesselName(terms.vessel_name);
        if (terms.imo_number && !imoNumber) setImoNumber(terms.imo_number);
        if (terms.hire_rate) setHireRate(String(terms.hire_rate));
        if (terms.freight_rate) setFreightRate(String(terms.freight_rate));
        if (terms.cargo_type) setCargoType(terms.cargo_type);
        if (terms.cargo_quantity) setCargoQty(String(terms.cargo_quantity));
        if (terms.load_port) setLoadPort(terms.load_port);
        if (terms.discharge_port) setDischargePort(terms.discharge_port);
        if (terms.laytime_allowed_hours) setLaytimeHours(String(terms.laytime_allowed_hours));
        if (terms.demurrage_rate) setDemurrageRate(String(terms.demurrage_rate));
        if (terms.despatch_rate) setDespatchRate(String(terms.despatch_rate));
        if (terms.currency) setCurrency(terms.currency.toUpperCase());
        if (!name.trim() && terms.vessel_name) {
          setName(`${terms.vessel_name} — Fixture`);
        }
      } catch (e: unknown) {
        const msg = e instanceof Error ? e.message : 'Scan failed';
        setScanError(msg);
        setExtractedTerms(null);
      }
      setStep('review');
    } catch {
      setScanError('Upload or processing failed. Try again.');
      setStep('upload');
    } finally {
      setUploadProgress(false);
    }
  };

  const handleCreate = async () => {
    if (!name.trim()) return;
    setCreateError('');
    setCreating(true);
    try {
      const terms = extractedTerms ?? {};
      const num = (s: string) => { const n = parseFloat(s); return isNaN(n) ? undefined : n; };
      const v = await api.voyages.create({
        voyage_number: name.trim(),
        charter_type: charterType,
        vessel_name: vesselName || terms.vessel_name || undefined,
        imo_number: imoNumber || terms.imo_number || undefined,
        vessel_type: terms.vessel_type || undefined,
        dwt: terms.dwt || undefined,
        flag_state: terms.flag_state || undefined,
        hire_rate: num(hireRate) ?? terms.hire_rate ?? undefined,
        freight_rate: num(freightRate) ?? terms.freight_rate ?? undefined,
        cargo_type: cargoType || terms.cargo_type || undefined,
        cargo_quantity: num(cargoQty) ?? terms.cargo_quantity ?? undefined,
        departure_port: loadPort || terms.load_port || undefined,
        arrival_port: dischargePort || terms.discharge_port || undefined,
        laytime_allowed_hours: num(laytimeHours) ?? terms.laytime_allowed_hours ?? undefined,
        demurrage_rate: num(demurrageRate) ?? terms.demurrage_rate ?? undefined,
        despatch_rate: num(despatchRate) ?? terms.despatch_rate ?? undefined,
        demurrage_currency: currency || 'USD',
        status: 'planned',
      });
      if (uploadedDoc) {
        await api.voyages.attachDocument(v.id, uploadedDoc.id);
      }
      navigate(`/voyages/${v.id}`);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to create fixture';
      setCreateError(msg);
    } finally {
      setCreating(false);
    }
  };

  const byType = (ct: string) => fixtures.filter(v => v.charter_type === ct);
  const uncategorized = fixtures.filter(v => !v.charter_type || !CHARTER_TYPES.includes(v.charter_type as typeof CHARTER_TYPES[number]));

  return (
    <div className="page-shell">
      <NavBar backTo="/dashboard" backLabel="Dashboard" />
      <div className="page-content">
        <div className="page-header">
          <div>
            <h1>Fixed Charter Parties</h1>
            <p className="page-subtitle">Upload a charter party, scan terms, then track laytime and demurrage</p>
          </div>
          <button className="btn-primary" onClick={openCreate}>+ New Fixture</button>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: '3rem' }}>
            <div className="loading-spinner" style={{ margin: '0 auto' }} />
          </div>
        ) : fixtures.length === 0 ? (
          <div className="empty-state">
            <div className="empty-icon">📄</div>
            <h3>No fixtures yet</h3>
            <p>Upload your charter party PDF. We&apos;ll read hire, laytime, and demurrage where possible — you can always edit in Overview.</p>
            <button className="btn-primary" onClick={openCreate}>Create your first fixture</button>
          </div>
        ) : (
          <>
            {CHARTER_TYPES.map(ct => {
              const items = byType(ct);
              if (items.length === 0) return null;
              const active  = items.filter(v => v.status === 'in_progress');
              const planned = items.filter(v => v.status === 'planned');
              const done    = items.filter(v => v.status === 'completed' || v.status === 'cancelled');
              return (
                <div key={ct} className="charter-type-group">
                  <div className="charter-type-group-header">
                    <span className="charter-type-group-icon">{CHARTER_TYPE_ICON[ct]}</span>
                    <h2>{CHARTER_TYPE_FULL[ct]}</h2>
                    <span className="charter-type-group-count">{items.length}</span>
                  </div>
                  {active.length > 0  && <FixtureSection title="Active"    items={active}  onOpen={id => navigate(`/voyages/${id}`)} />}
                  {planned.length > 0 && <FixtureSection title="Planned"   items={planned} onOpen={id => navigate(`/voyages/${id}`)} />}
                  {done.length > 0    && <FixtureSection title="Completed" items={done}    onOpen={id => navigate(`/voyages/${id}`)} />}
                </div>
              );
            })}
            {uncategorized.length > 0 && (
              <div className="charter-type-group">
                <div className="charter-type-group-header">
                  <span className="charter-type-group-icon">📄</span>
                  <h2>Other</h2>
                  <span className="charter-type-group-count">{uncategorized.length}</span>
                </div>
                <FixtureSection title="All" items={uncategorized} onOpen={id => navigate(`/voyages/${id}`)} />
              </div>
            )}
          </>
        )}
      </div>

      {showCreate && (
        <div className="modal-backdrop" onClick={() => setShowCreate(false)}>
          <div className="modal-box" onClick={e => e.stopPropagation()} style={{ maxWidth: 520 }}>

            <div className="create-steps">
              {(['upload', 'review'] as const).map((s, i) => (
                <div
                  key={s}
                  className={`create-step ${
                    step === s || (step === 'scanning' && s === 'upload') ? 'create-step--active' : ''
                  } ${s === 'upload' && step === 'review' ? 'create-step--done' : ''}`}
                >
                  <span className="create-step-num">{i + 1}</span>
                  <span className="create-step-label">
                    {s === 'upload' ? 'Charter Party' : 'Name & create'}
                  </span>
                </div>
              ))}
            </div>

            {(step === 'upload' || step === 'scanning') && (
              <>
                <h3 style={{ margin: '1rem 0 0.5rem' }}>Attach Charter Party</h3>
                <p className="hint" style={{ fontSize: '0.82rem', marginBottom: '1rem' }}>
                  Same idea as the negotiation room: upload the PDF, we extract text for free, then run a term scan (AI) to pre-fill the fixture. You can skip the scan and enter everything manually on the next step.
                </p>

                {step === 'scanning' ? (
                  <div className="cp-scanning-box">
                    <div className="loading-spinner" style={{ margin: '0 auto 1rem' }} />
                    <p>Uploading &amp; scanning for terms…</p>
                    <p style={{ fontSize: '0.78rem', color: 'var(--color-text-secondary)' }}>
                      Extracting hire rate, laytime, demurrage, vessel…
                    </p>
                  </div>
                ) : (
                  <div
                    className="cp-upload-dropzone"
                    onClick={() => !uploadProgress && fileRef.current?.click()}
                    onDragOver={e => e.preventDefault()}
                    onDrop={e => {
                      e.preventDefault();
                      const f = e.dataTransfer.files[0];
                      if (f) handleUpload(f);
                    }}
                  >
                    {uploadProgress ? (
                      <>
                        <div className="loading-spinner" style={{ margin: '0 auto 0.5rem' }} />
                        <p>Working…</p>
                      </>
                    ) : (
                      <>
                        <div style={{ fontSize: '2rem' }}>📄</div>
                        <p><strong>Click to upload</strong> or drag &amp; drop</p>
                        <p style={{ fontSize: '0.78rem', color: 'var(--color-text-secondary)' }}>PDF (and processed text for scan)</p>
                      </>
                    )}
                    <input
                      ref={fileRef}
                      type="file"
                      accept=".pdf,application/pdf"
                      style={{ display: 'none' }}
                      onChange={e => {
                        const f = e.target.files?.[0];
                        if (f) handleUpload(f);
                        e.target.value = '';
                      }}
                    />
                  </div>
                )}

                {scanError && (
                  <p style={{ color: '#ef4444', fontSize: '0.82rem', marginTop: '0.5rem' }}>{scanError}</p>
                )}

                <div className="modal-actions" style={{ marginTop: '1rem' }}>
                  <button className="btn-secondary" onClick={() => setShowCreate(false)}>Cancel</button>
                  <button
                    className="btn-secondary"
                    onClick={() => { setExtractedTerms(null); setUploadedDoc(null); setStep('review'); }}
                    disabled={step === 'scanning'}
                  >
                    Skip upload — manual entry →
                  </button>
                </div>
              </>
            )}

            {step === 'review' && (
              <>
                <h3 style={{ margin: '1rem 0 0.25rem' }}>Fixture name &amp; details</h3>
                <p style={{ fontSize: '0.82rem', color: 'var(--color-text-secondary)', marginBottom: '0.75rem' }}>
                  {uploadedDoc
                    ? 'Adjust anything below, then create. The charter party PDF stays linked to this fixture.'
                    : 'Enter a name and charter type. You can add a charter party PDF later in the fixture.'}
                </p>

                <label className="field-label">Fixture name *</label>
                <input
                  className="field-input"
                  placeholder="e.g. MV Pacific Star — TC Q1 2026"
                  value={name}
                  onChange={e => setName(e.target.value)}
                  autoFocus
                />

                <label className="field-label" style={{ marginTop: '0.65rem' }}>Charter type</label>
                <div className="charter-type-selector">
                  {CHARTER_TYPES.map(ct => (
                    <button
                      key={ct}
                      type="button"
                      className={`charter-type-btn ${charterType === ct ? 'charter-type-btn--active' : ''}`}
                      onClick={() => setCharterType(ct)}
                    >
                      <span className="charter-type-btn-icon">{CHARTER_TYPE_ICON[ct]}</span>
                      <span className="charter-type-btn-label">{CHARTER_TYPE_FULL[ct]}</span>
                      <span className="charter-type-btn-abbr">{CHARTER_TYPE_LABEL[ct]}</span>
                    </button>
                  ))}
                </div>

                <div className="form-row-2" style={{ marginTop: '0.65rem' }}>
                  <div>
                    <label className="field-label">Vessel name</label>
                    <input className="field-input" placeholder="MV Pacific Star" value={vesselName} onChange={e => setVesselName(e.target.value)} />
                  </div>
                  <div>
                    <label className="field-label">IMO</label>
                    <input className="field-input" placeholder="9234567" value={imoNumber} onChange={e => setImoNumber(e.target.value)} />
                  </div>
                </div>

                {/* Time Charter fields */}
                {charterType === 'time_charter' && (
                  <div className="charter-fields-section">
                    <div className="charter-fields-heading">Time Charter Terms</div>
                    <div className="form-row-2">
                      <div>
                        <label className="field-label">Hire Rate (per day)</label>
                        <input className="field-input" type="number" placeholder="25000" value={hireRate} onChange={e => setHireRate(e.target.value)} />
                      </div>
                      <div>
                        <label className="field-label">Currency</label>
                        <select className="field-input" value={currency} onChange={e => setCurrency(e.target.value)}>
                          <option value="USD">USD</option>
                          <option value="EUR">EUR</option>
                          <option value="GBP">GBP</option>
                        </select>
                      </div>
                    </div>
                  </div>
                )}

                {/* Voyage Charter fields */}
                {charterType === 'voyage_charter' && (
                  <div className="charter-fields-section">
                    <div className="charter-fields-heading">Voyage Charter Terms</div>
                    <div className="form-row-2">
                      <div>
                        <label className="field-label">Freight Rate</label>
                        <input className="field-input" type="number" placeholder="15.50" value={freightRate} onChange={e => setFreightRate(e.target.value)} />
                      </div>
                      <div>
                        <label className="field-label">Currency</label>
                        <select className="field-input" value={currency} onChange={e => setCurrency(e.target.value)}>
                          <option value="USD">USD</option>
                          <option value="EUR">EUR</option>
                          <option value="GBP">GBP</option>
                        </select>
                      </div>
                    </div>
                    <div className="form-row-2">
                      <div>
                        <label className="field-label">Cargo Type</label>
                        <input className="field-input" placeholder="Iron Ore" value={cargoType} onChange={e => setCargoType(e.target.value)} />
                      </div>
                      <div>
                        <label className="field-label">Cargo Quantity (MT)</label>
                        <input className="field-input" type="number" placeholder="50000" value={cargoQty} onChange={e => setCargoQty(e.target.value)} />
                      </div>
                    </div>
                    <div className="form-row-2">
                      <div>
                        <label className="field-label">Load Port</label>
                        <input className="field-input" placeholder="Rotterdam" value={loadPort} onChange={e => setLoadPort(e.target.value)} />
                      </div>
                      <div>
                        <label className="field-label">Discharge Port</label>
                        <input className="field-input" placeholder="Singapore" value={dischargePort} onChange={e => setDischargePort(e.target.value)} />
                      </div>
                    </div>
                    <div className="form-row-3">
                      <div>
                        <label className="field-label">Laytime (hours)</label>
                        <input className="field-input" type="number" placeholder="72" value={laytimeHours} onChange={e => setLaytimeHours(e.target.value)} />
                      </div>
                      <div>
                        <label className="field-label">Demurrage Rate/day</label>
                        <input className="field-input" type="number" placeholder="25000" value={demurrageRate} onChange={e => setDemurrageRate(e.target.value)} />
                      </div>
                      <div>
                        <label className="field-label">Despatch Rate/day</label>
                        <input className="field-input" type="number" placeholder="12500" value={despatchRate} onChange={e => setDespatchRate(e.target.value)} />
                      </div>
                    </div>
                  </div>
                )}

                {/* Bareboat fields */}
                {charterType === 'bareboat' && (
                  <div className="charter-fields-section">
                    <div className="charter-fields-heading">Bareboat Terms</div>
                    <div className="form-row-2">
                      <div>
                        <label className="field-label">Hire Rate (per day)</label>
                        <input className="field-input" type="number" placeholder="15000" value={hireRate} onChange={e => setHireRate(e.target.value)} />
                      </div>
                      <div>
                        <label className="field-label">Currency</label>
                        <select className="field-input" value={currency} onChange={e => setCurrency(e.target.value)}>
                          <option value="USD">USD</option>
                          <option value="EUR">EUR</option>
                          <option value="GBP">GBP</option>
                        </select>
                      </div>
                    </div>
                  </div>
                )}

                {extractedTerms?.raw_summary && (
                  <div style={{ marginTop: '0.75rem', fontSize: '0.78rem', color: 'var(--color-text-secondary)', maxHeight: '6rem', overflow: 'auto', whiteSpace: 'pre-wrap' }}>
                    {extractedTerms.raw_summary}
                  </div>
                )}

                {uploadedDoc && (
                  <p style={{ fontSize: '0.78rem', color: '#10b981', marginTop: '0.5rem' }}>
                    ✓ {uploadedDoc.original_filename}
                  </p>
                )}

                {createError && (
                  <p style={{ color: '#ef4444', fontSize: '0.82rem', marginTop: '0.5rem' }}>{createError}</p>
                )}

                <div className="modal-actions" style={{ marginTop: '1.25rem' }}>
                  <button className="btn-secondary" onClick={() => setStep('upload')}>← Back</button>
                  <button className="btn-primary" onClick={handleCreate} disabled={creating || !name.trim()}>
                    {creating ? 'Creating…' : 'Create fixture'}
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function FixtureSection({ title, items, onOpen }: { title: string; items: Voyage[]; onOpen: (id: string) => void }) {
  return (
    <section className="voyage-section">
      <h2 className="voyage-section-title">{title}</h2>
      <div className="voyage-grid">
        {items.map(v => <FixtureCard key={v.id} voyage={v} onClick={() => onOpen(v.id)} />)}
      </div>
    </section>
  );
}

function FixtureCard({ voyage, onClick }: { voyage: Voyage; onClick: () => void }) {
  const color = STATUS_COLOR[voyage.status] ?? '#64748b';
  const label = STATUS_LABEL[voyage.status] ?? voyage.status;
  const typeLabel = voyage.charter_type ? CHARTER_TYPE_LABEL[voyage.charter_type] ?? voyage.charter_type : null;
  const title = voyage.voyage_number ?? voyage.vessel_name ?? 'Unnamed Fixture';

  return (
    <div className="voyage-card" onClick={onClick}>
      <div className="voyage-card-header">
        <span className="voyage-vessel">{title}</span>
        <div style={{ display: 'flex', gap: '0.35rem', alignItems: 'center' }}>
          {typeLabel && <span className="voyage-type-pill">{typeLabel}</span>}
          <span className="voyage-status-badge" style={{ background: color + '1a', color }}>{label}</span>
        </div>
      </div>
      {voyage.vessel_name && voyage.voyage_number && (
        <div className="voyage-meta">🚢 {voyage.vessel_name}</div>
      )}
      {voyage.imo_number && <div className="voyage-meta">IMO {voyage.imo_number}</div>}
      {voyage.hire_rate && (
        <div className="voyage-meta">
          Hire: {voyage.demurrage_currency ?? 'USD'} {voyage.hire_rate.toLocaleString()}/day
        </div>
      )}
      {voyage.demurrage_rate && (
        <div className="voyage-meta">
          Dem: {voyage.demurrage_currency ?? 'USD'} {voyage.demurrage_rate.toLocaleString()}/day
        </div>
      )}
      {voyage.document_id && <div className="voyage-meta" style={{ color: '#10b981' }}>📄 Charter Party attached</div>}
    </div>
  );
}
