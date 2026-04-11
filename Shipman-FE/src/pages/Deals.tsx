import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, ApiError } from '../api/client';
import type { Deal } from '../api/client';
import { useAuth } from '../context/AuthContext';
import NavBar from '../components/NavBar';

type CreateStep = 'basics' | 'vessel' | 'cargo';

interface VesselForm {
  vessel_name: string; imo_number: string; vessel_type: string; flag_state: string;
  deadweight_tonnage: string; gross_tonnage: string; build_year: string; class_society: string;
  current_position: string; available_from: string; asking_rate: string;
  asking_rate_currency: string; asking_rate_type: string; notes: string;
}

interface CargoForm {
  commodity: string; quantity: string; quantity_unit: string;
  load_port: string; discharge_port: string;
  laycan_from: string; laycan_to: string;
  freight_idea: string; freight_currency: string; freight_type: string;
  special_requirements: string; notes: string;
}

const emptyVessel: VesselForm = {
  vessel_name: '', imo_number: '', vessel_type: '', flag_state: '',
  deadweight_tonnage: '', gross_tonnage: '', build_year: '', class_society: '',
  current_position: '', available_from: '', asking_rate: '',
  asking_rate_currency: 'USD', asking_rate_type: 'per_day', notes: '',
};

const emptyCargo: CargoForm = {
  commodity: '', quantity: '', quantity_unit: 'MT',
  load_port: '', discharge_port: '',
  laycan_from: '', laycan_to: '',
  freight_idea: '', freight_currency: 'USD', freight_type: 'per_mt',
  special_requirements: '', notes: '',
};

export default function Deals() {
  const [deals, setDeals] = useState<Deal[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [createStep, setCreateStep] = useState<CreateStep>('basics');
  const [newDeal, setNewDeal] = useState<Deal | null>(null);
  const [title, setTitle] = useState('');
  const [vesselForm, setVesselForm] = useState<VesselForm>(emptyVessel);
  const [cargoForm, setCargoForm] = useState<CargoForm>(emptyCargo);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const { user } = useAuth();

  useEffect(() => { loadDeals(); }, []);

  const loadDeals = async () => {
    try {
      const response = await api.deals.list();
      setDeals(response.data);
    } catch {
      setError('Failed to load deals');
    } finally {
      setIsLoading(false);
    }
  };

  const resetCreateModal = () => {
    setShowCreate(false);
    setCreateStep('basics');
    setNewDeal(null);
    setTitle('');
    setVesselForm(emptyVessel);
    setCargoForm(emptyCargo);
  };

  const handleCreateBasics = async () => {
    if (!title.trim()) return;
    try {
      const deal = await api.deals.create({ title });
      setNewDeal(deal);
      if (user?.role === 'shipowner') {
        setCreateStep('vessel');
      } else if (user?.role === 'charterer') {
        setCreateStep('cargo');
      } else {
        setDeals([deal, ...deals]);
        resetCreateModal();
        navigate(`/deals/${deal.id}`);
      }
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to create deal');
    }
  };

  const handleSaveVessel = async () => {
    if (!newDeal) return;
    try {
      const payload: Record<string, string | number | undefined> = {};
      if (vesselForm.vessel_name) payload.vessel_name = vesselForm.vessel_name;
      if (vesselForm.imo_number) payload.imo_number = vesselForm.imo_number;
      if (vesselForm.vessel_type) payload.vessel_type = vesselForm.vessel_type;
      if (vesselForm.flag_state) payload.flag_state = vesselForm.flag_state;
      if (vesselForm.deadweight_tonnage) payload.deadweight_tonnage = parseFloat(vesselForm.deadweight_tonnage);
      if (vesselForm.gross_tonnage) payload.gross_tonnage = parseFloat(vesselForm.gross_tonnage);
      if (vesselForm.build_year) payload.build_year = parseInt(vesselForm.build_year);
      if (vesselForm.class_society) payload.class_society = vesselForm.class_society;
      if (vesselForm.current_position) payload.current_position = vesselForm.current_position;
      if (vesselForm.available_from) payload.available_from = vesselForm.available_from;
      if (vesselForm.asking_rate) payload.asking_rate = parseFloat(vesselForm.asking_rate);
      payload.asking_rate_currency = vesselForm.asking_rate_currency;
      payload.asking_rate_type = vesselForm.asking_rate_type;
      if (vesselForm.notes) payload.notes = vesselForm.notes;
      await api.deals.upsertVesselDetails(newDeal.id, payload);
      setDeals([newDeal, ...deals]);
      resetCreateModal();
      navigate(`/deals/${newDeal.id}`);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to save vessel details');
    }
  };

  const handleSaveCargo = async () => {
    if (!newDeal) return;
    try {
      const payload: Record<string, string | number | undefined> = {};
      if (cargoForm.commodity) payload.commodity = cargoForm.commodity;
      if (cargoForm.quantity) payload.quantity = parseFloat(cargoForm.quantity);
      payload.quantity_unit = cargoForm.quantity_unit;
      if (cargoForm.load_port) payload.load_port = cargoForm.load_port;
      if (cargoForm.discharge_port) payload.discharge_port = cargoForm.discharge_port;
      if (cargoForm.laycan_from) payload.laycan_from = cargoForm.laycan_from;
      if (cargoForm.laycan_to) payload.laycan_to = cargoForm.laycan_to;
      if (cargoForm.freight_idea) payload.freight_idea = parseFloat(cargoForm.freight_idea);
      payload.freight_currency = cargoForm.freight_currency;
      payload.freight_type = cargoForm.freight_type;
      if (cargoForm.special_requirements) payload.special_requirements = cargoForm.special_requirements;
      if (cargoForm.notes) payload.notes = cargoForm.notes;
      await api.deals.upsertCargoDetails(newDeal.id, payload);
      setDeals([newDeal, ...deals]);
      resetCreateModal();
      navigate(`/deals/${newDeal.id}`);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to save cargo details');
    }
  };

  const vf = (field: keyof VesselForm) => ({
    value: vesselForm[field],
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
      setVesselForm(prev => ({ ...prev, [field]: e.target.value })),
  });

  const cf = (field: keyof CargoForm) => ({
    value: cargoForm[field],
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
      setCargoForm(prev => ({ ...prev, [field]: e.target.value })),
  });

  const getStatusBadge = (status: Deal['status']) => {
    const styles: Record<Deal['status'], string> = {
      active: 'badge-success',
      completed: 'badge-info',
      cancelled: 'badge-error',
    };
    return `badge ${styles[status]}`;
  };

  return (
    <div className="deals-page">
      <NavBar />
      <div className="page-content">
      <header className="page-header">
        <div>
          <h1>Negotiations</h1>
          <p>Collaborate with counterparties on charter terms in real-time.</p>
        </div>
        <div className="header-actions">
          <button className="btn-primary" onClick={() => setShowCreate(true)}>
            New Negotiation
          </button>
        </div>
      </header>

      {error && (
        <div className="error-banner" onClick={() => setError(null)}>{error}</div>
      )}

      {/* ---- CREATE MODAL ---- */}
      {showCreate && (
        <div className="modal-overlay" onClick={resetCreateModal}>
          <div className="modal modal-wide" onClick={e => e.stopPropagation()}>

            {createStep === 'basics' && (
              <>
                <div className="modal-header">
                  <h2>Start a New Negotiation</h2>
                  <p className="modal-sub">Give the negotiation a title, then fill in your side of the deal.</p>
                </div>
                <div className="form-group">
                  <label>Deal Title</label>
                  <input
                    type="text"
                    value={title}
                    onChange={e => setTitle(e.target.value)}
                    placeholder="e.g., MV Ocean Star — Time Charter Q3 2026"
                    autoFocus
                    onKeyDown={e => e.key === 'Enter' && handleCreateBasics()}
                  />
                </div>
                <div className="modal-actions">
                  <button className="btn-secondary" onClick={resetCreateModal}>Cancel</button>
                  <button className="btn-primary" onClick={handleCreateBasics} disabled={!title.trim()}>
                    Next →
                  </button>
                </div>
              </>
            )}

            {createStep === 'vessel' && (
              <>
                <div className="modal-header">
                  <h2>Vessel Details</h2>
                  <p className="modal-sub">Fill in as much as you know. You can update this later in the deal room.</p>
                </div>
                <div className="form-grid">
                  <div className="form-group"><label>Vessel Name</label><input type="text" placeholder="MV Ocean Star" {...vf('vessel_name')} /></div>
                  <div className="form-group"><label>IMO Number</label><input type="text" placeholder="9876543" {...vf('imo_number')} /></div>
                  <div className="form-group"><label>Vessel Type</label><input type="text" placeholder="Bulk Carrier, Tanker..." {...vf('vessel_type')} /></div>
                  <div className="form-group"><label>Flag State</label><input type="text" placeholder="Panama" {...vf('flag_state')} /></div>
                  <div className="form-group"><label>DWT (tonnes)</label><input type="number" {...vf('deadweight_tonnage')} /></div>
                  <div className="form-group"><label>GRT</label><input type="number" {...vf('gross_tonnage')} /></div>
                  <div className="form-group"><label>Build Year</label><input type="number" placeholder="2018" {...vf('build_year')} /></div>
                  <div className="form-group"><label>Class Society</label><input type="text" placeholder="Lloyd's Register" {...vf('class_society')} /></div>
                  <div className="form-group"><label>Current Position</label><input type="text" placeholder="Rotterdam" {...vf('current_position')} /></div>
                  <div className="form-group"><label>Available From</label><input type="date" {...vf('available_from')} /></div>
                  <div className="form-group"><label>Asking Rate</label><input type="number" placeholder="0.00" {...vf('asking_rate')} /></div>
                  <div className="form-group"><label>Rate Type</label>
                    <select {...vf('asking_rate_type')}>
                      <option value="per_day">Per Day</option>
                      <option value="lumpsum">Lump Sum</option>
                    </select>
                  </div>
                </div>
                <div className="form-group"><label>Notes</label><textarea rows={2} {...vf('notes')} /></div>
                <div className="modal-actions">
                  <button className="btn-secondary" onClick={() => {
                    if (newDeal) { setDeals([newDeal, ...deals]); resetCreateModal(); navigate(`/deals/${newDeal.id}`); }
                  }}>Skip for now</button>
                  <button className="btn-primary" onClick={handleSaveVessel}>Save & Open Deal Room →</button>
                </div>
              </>
            )}

            {createStep === 'cargo' && (
              <>
                <div className="modal-header">
                  <h2>Cargo Details</h2>
                  <p className="modal-sub">Fill in your cargo requirements. You can update this later in the deal room.</p>
                </div>
                <div className="form-grid">
                  <div className="form-group"><label>Commodity</label><input type="text" placeholder="Iron Ore, Grain, Coal" {...cf('commodity')} /></div>
                  <div className="form-group">
                    <label>Quantity</label>
                    <div className="input-with-addon">
                      <input type="number" placeholder="50000" {...cf('quantity')} />
                      <select style={{ width: 90 }} {...cf('quantity_unit')}>
                        <option value="MT">MT</option>
                        <option value="CBM">CBM</option>
                        <option value="TEU">TEU</option>
                      </select>
                    </div>
                  </div>
                  <div className="form-group"><label>Load Port</label><input type="text" placeholder="Rotterdam" {...cf('load_port')} /></div>
                  <div className="form-group"><label>Discharge Port</label><input type="text" placeholder="Singapore" {...cf('discharge_port')} /></div>
                  <div className="form-group"><label>Laycan From</label><input type="date" {...cf('laycan_from')} /></div>
                  <div className="form-group"><label>Laycan To</label><input type="date" {...cf('laycan_to')} /></div>
                  <div className="form-group"><label>Freight Idea</label><input type="number" placeholder="0.00" {...cf('freight_idea')} /></div>
                  <div className="form-group"><label>Freight Type</label>
                    <select {...cf('freight_type')}>
                      <option value="per_mt">Per MT</option>
                      <option value="lumpsum">Lump Sum</option>
                      <option value="per_day">Per Day</option>
                    </select>
                  </div>
                </div>
                <div className="form-group"><label>Special Requirements</label><textarea rows={2} {...cf('special_requirements')} /></div>
                <div className="form-group"><label>Notes</label><textarea rows={2} {...cf('notes')} /></div>
                <div className="modal-actions">
                  <button className="btn-secondary" onClick={() => {
                    if (newDeal) { setDeals([newDeal, ...deals]); resetCreateModal(); navigate(`/deals/${newDeal.id}`); }
                  }}>Skip for now</button>
                  <button className="btn-primary" onClick={handleSaveCargo}>Save & Open Deal Room →</button>
                </div>
              </>
            )}

          </div>
        </div>
      )}

      {isLoading ? (
        <div className="loading-container">
          <div className="loading-spinner" />
          <p>Loading negotiations...</p>
        </div>
      ) : deals.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">⚓</div>
          <h3>No negotiations yet</h3>
          <p>Start a new negotiation, or accept an email invite from a counterparty to join theirs.</p>
          <button className="btn-primary" onClick={() => setShowCreate(true)}>
            Start Your First Negotiation
          </button>
        </div>
      ) : (
        <div className="deals-list">
          {deals.map(deal => (
            <div key={deal.id} className="deal-card" onClick={() => navigate(`/deals/${deal.id}`)}>
              <div className="deal-info">
                <h3>{deal.title}</h3>
                <div className="deal-meta">
                  <span className={getStatusBadge(deal.status)}>{deal.status}</span>
                  <span>{new Date(deal.created_at).toLocaleDateString()}</span>
                </div>
              </div>
              <div className="deal-arrow">→</div>
            </div>
          ))}
        </div>
      )}
      </div>
    </div>
  );
}
