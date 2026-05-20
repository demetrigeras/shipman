import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import NavBar from '../components/NavBar';

export default function Dashboard() {
  const { user } = useAuth();
  const navigate = useNavigate();

  const roleLabels: Record<string, string> = {
    shipowner: 'Ship Owner',
    charterer: 'Charterer',
    broker: 'Broker',
  };

  return (
    <div className="dashboard">
      <NavBar />

      <main className="dashboard-main">
        <div className="welcome-section">
          <h2>Welcome back, {user?.full_name}</h2>
          <p>Signed in as <strong>{roleLabels[user?.role ?? 'charterer']}</strong></p>
        </div>

        <div className="action-cards">
          <div className="action-card" onClick={() => navigate('/voyages')}>
            <div className="card-icon">⚓</div>
            <h3>Fixed C/P</h3>
            <p>Manage active fixtures: negotiate clauses, track laytime, monitor positions, and settle payments — all in one voyage room.</p>
            <button className="btn-primary">Open Fixtures →</button>
          </div>

          <div className="action-card" onClick={() => navigate('/marketplace')}>
            <div className="card-icon">🚢</div>
            <h3>Vessel Marketplace</h3>
            <p>Browse vessels available for sale or list your own. Connect buyers and sellers directly.</p>
            <button className="btn-primary">Browse Marketplace →</button>
          </div>
        </div>
      </main>
    </div>
  );
}
