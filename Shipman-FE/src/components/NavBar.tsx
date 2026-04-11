import { useNavigate, useLocation, Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

interface NavBarProps {
  backTo?: string;
  backLabel?: string;
}

export default function NavBar({ backTo, backLabel }: NavBarProps) {
  const { user, signout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const handleSignout = () => {
    signout();
    navigate('/login');
  };

  const isActive = (path: string) =>
    location.pathname === path || location.pathname.startsWith(path + '/');

  return (
    <nav className="app-nav">
      <div className="app-nav-left">
        <Link to="/dashboard" className="app-nav-logo">⚓ Shipman</Link>
        <div className="app-nav-links">
          <Link
            to="/documents"
            className={`nav-link${isActive('/documents') ? ' nav-link--active' : ''}`}
          >
            Documents
          </Link>
          <Link
            to="/deals"
            className={`nav-link${isActive('/deals') ? ' nav-link--active' : ''}`}
          >
            Negotiations
          </Link>
        </div>
      </div>
      <div className="app-nav-right">
        {backTo && backTo !== location.pathname && (
          <button className="btn-back-nav" onClick={() => navigate(backTo)}>
            ← {backLabel ?? 'Back'}
          </button>
        )}
        <span className="nav-user">{user?.full_name}</span>
        <button className="btn-signout" onClick={handleSignout}>Sign Out</button>
      </div>
    </nav>
  );
}
