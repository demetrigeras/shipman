import { useState, useEffect } from 'react';
import { api, ApiError } from '../api/client';
import type { Vessel, VesselCreateData } from '../api/client';
import NavBar from '../components/NavBar';

export default function Marketplace() {
  const [vessels, setVessels] = useState<Vessel[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [selectedVessel, setSelectedVessel] = useState<Vessel | null>(null);
  const [error, setError] = useState<string | null>(null);
  
  const [formData, setFormData] = useState<VesselCreateData>({
    name: '',
    imo_number: '',
    flag_state: '',
    vessel_type: '',
    deadweight_tonnage: undefined,
    gross_tonnage: undefined,
    build_year: undefined,
    owner: '',
  });

  useEffect(() => {
    loadVessels();
  }, []);

  const loadVessels = async () => {
    try {
      const response = await api.marketplace.listVessels();
      setVessels(response.data);
    } catch (e) {
      setError('Failed to load vessels');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreate = async () => {
    if (!formData.name.trim()) return;
    
    try {
      const vessel = await api.marketplace.createVessel(formData);
      setVessels([vessel, ...vessels]);
      setShowCreate(false);
      resetForm();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to create vessel');
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this vessel?')) return;
    
    try {
      await api.marketplace.deleteVessel(id);
      setVessels(vessels.filter(v => v.id !== id));
      if (selectedVessel?.id === id) {
        setSelectedVessel(null);
      }
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to delete vessel');
    }
  };

  const resetForm = () => {
    setFormData({
      name: '',
      imo_number: '',
      flag_state: '',
      vessel_type: '',
      deadweight_tonnage: undefined,
      gross_tonnage: undefined,
      build_year: undefined,
      owner: '',
    });
  };

  const formatNumber = (num?: number) => {
    if (num === undefined || num === null) return '-';
    return num.toLocaleString();
  };

  return (
    <div className="marketplace-page">
      <NavBar />
      <div className="page-content">
      <header className="page-header">
        <div>
          <h1>Vessel Marketplace</h1>
          <p>Browse vessels available for charter or sale, or list your own.</p>
        </div>
        <div className="header-actions">
          <button className="btn-primary" onClick={() => setShowCreate(true)}>
            List a Vessel
          </button>
        </div>
      </header>

      {error && (
        <div className="error-banner" onClick={() => setError(null)}>
          {error}
        </div>
      )}

      {showCreate && (
        <div className="modal-overlay" onClick={() => setShowCreate(false)}>
          <div className="modal modal-lg" onClick={e => e.stopPropagation()}>
            <h2>List a New Vessel</h2>
            <div className="form-grid">
              <div className="form-group">
                <label>Vessel Name *</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={e => setFormData({ ...formData, name: e.target.value })}
                  placeholder="e.g., MV Ocean Star"
                />
              </div>
              <div className="form-group">
                <label>IMO Number</label>
                <input
                  type="text"
                  value={formData.imo_number || ''}
                  onChange={e => setFormData({ ...formData, imo_number: e.target.value })}
                  placeholder="e.g., 9123456"
                />
              </div>
              <div className="form-group">
                <label>Flag State</label>
                <input
                  type="text"
                  value={formData.flag_state || ''}
                  onChange={e => setFormData({ ...formData, flag_state: e.target.value })}
                  placeholder="e.g., Panama"
                />
              </div>
              <div className="form-group">
                <label>Vessel Type</label>
                <select
                  value={formData.vessel_type || ''}
                  onChange={e => setFormData({ ...formData, vessel_type: e.target.value })}
                >
                  <option value="">Select type</option>
                  <option value="bulk_carrier">Bulk Carrier</option>
                  <option value="tanker">Tanker</option>
                  <option value="container">Container Ship</option>
                  <option value="general_cargo">General Cargo</option>
                  <option value="ro_ro">Ro-Ro</option>
                  <option value="lpg">LPG Carrier</option>
                  <option value="lng">LNG Carrier</option>
                  <option value="other">Other</option>
                </select>
              </div>
              <div className="form-group">
                <label>DWT (tonnes)</label>
                <input
                  type="number"
                  value={formData.deadweight_tonnage || ''}
                  onChange={e => setFormData({ ...formData, deadweight_tonnage: parseFloat(e.target.value) || undefined })}
                  placeholder="e.g., 50000"
                />
              </div>
              <div className="form-group">
                <label>Gross Tonnage</label>
                <input
                  type="number"
                  value={formData.gross_tonnage || ''}
                  onChange={e => setFormData({ ...formData, gross_tonnage: parseFloat(e.target.value) || undefined })}
                  placeholder="e.g., 30000"
                />
              </div>
              <div className="form-group">
                <label>Build Year</label>
                <input
                  type="number"
                  value={formData.build_year || ''}
                  onChange={e => setFormData({ ...formData, build_year: parseInt(e.target.value) || undefined })}
                  placeholder="e.g., 2015"
                />
              </div>
              <div className="form-group">
                <label>Owner</label>
                <input
                  type="text"
                  value={formData.owner || ''}
                  onChange={e => setFormData({ ...formData, owner: e.target.value })}
                  placeholder="Owner company name"
                />
              </div>
            </div>
            <div className="modal-actions">
              <button className="btn-secondary" onClick={() => { setShowCreate(false); resetForm(); }}>Cancel</button>
              <button className="btn-primary" onClick={handleCreate}>List Vessel</button>
            </div>
          </div>
        </div>
      )}

      {selectedVessel && (
        <div className="modal-overlay" onClick={() => setSelectedVessel(null)}>
          <div className="modal modal-lg" onClick={e => e.stopPropagation()}>
            <h2>{selectedVessel.name}</h2>
            <div className="vessel-details-grid">
              <div className="detail-item">
                <span className="detail-label">IMO Number</span>
                <span className="detail-value">{selectedVessel.imo_number || '-'}</span>
              </div>
              <div className="detail-item">
                <span className="detail-label">Flag State</span>
                <span className="detail-value">{selectedVessel.flag_state || '-'}</span>
              </div>
              <div className="detail-item">
                <span className="detail-label">Vessel Type</span>
                <span className="detail-value">{selectedVessel.vessel_type?.replace('_', ' ') || '-'}</span>
              </div>
              <div className="detail-item">
                <span className="detail-label">DWT</span>
                <span className="detail-value">{formatNumber(selectedVessel.deadweight_tonnage)} tonnes</span>
              </div>
              <div className="detail-item">
                <span className="detail-label">Gross Tonnage</span>
                <span className="detail-value">{formatNumber(selectedVessel.gross_tonnage)}</span>
              </div>
              <div className="detail-item">
                <span className="detail-label">Build Year</span>
                <span className="detail-value">{selectedVessel.build_year || '-'}</span>
              </div>
              <div className="detail-item">
                <span className="detail-label">Owner</span>
                <span className="detail-value">{selectedVessel.owner || '-'}</span>
              </div>
            </div>
            <div className="modal-actions">
              <button className="btn-danger" onClick={() => handleDelete(selectedVessel.id)}>Delete</button>
              <button className="btn-primary" onClick={() => setSelectedVessel(null)}>Close</button>
            </div>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="loading-container">
          <div className="loading-spinner" />
          <p>Loading vessels...</p>
        </div>
      ) : vessels.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">🚢</div>
          <h3>No vessels listed yet</h3>
          <p>Be the first to list a vessel on the marketplace.</p>
          <button className="btn-primary" onClick={() => setShowCreate(true)}>
            List Your First Vessel
          </button>
        </div>
      ) : (
        <div className="vessels-grid">
          {vessels.map(vessel => (
            <div
              key={vessel.id}
              className="vessel-card"
              onClick={() => setSelectedVessel(vessel)}
            >
              <div className="vessel-card-header">
                <span className="vessel-type">{vessel.vessel_type?.replace('_', ' ') || 'Vessel'}</span>
                {vessel.flag_state && <span className="vessel-flag">{vessel.flag_state}</span>}
              </div>
              <h3>{vessel.name}</h3>
              <div className="vessel-specs">
                {vessel.deadweight_tonnage && (
                  <div className="spec">
                    <span className="spec-label">DWT</span>
                    <span className="spec-value">{formatNumber(vessel.deadweight_tonnage)} t</span>
                  </div>
                )}
                {vessel.build_year && (
                  <div className="spec">
                    <span className="spec-label">Built</span>
                    <span className="spec-value">{vessel.build_year}</span>
                  </div>
                )}
              </div>
              {vessel.imo_number && (
                <div className="vessel-imo">IMO: {vessel.imo_number}</div>
              )}
            </div>
          ))}
        </div>
      )}
      </div>
    </div>
  );
}
