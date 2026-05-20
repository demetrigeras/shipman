import { useState, useRef, useEffect } from 'react';
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
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const handleSignout = () => {
    setMenuOpen(false);
    signout();
    navigate('/login');
  };

  const isActive = (path: string) =>
    location.pathname === path || location.pathname.startsWith(path + '/');

  // Close dropdown when clicking outside
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const initials = user?.full_name
    ? user.full_name.split(' ').map(w => w[0]).slice(0, 2).join('').toUpperCase()
    : '?';

  const roleLabels: Record<string, string> = {
    shipowner: 'Ship Owner',
    charterer: 'Charterer',
    broker: 'Broker',
  };

  return (
    <nav className="app-nav">
      <div className="app-nav-left">
        <Link to="/dashboard" className="app-nav-logo">⚓ Shipman</Link>
        <div className="app-nav-links">
          <Link
            to="/voyages"
            className={`nav-link${isActive('/voyages') ? ' nav-link--active' : ''}`}
          >
            Fixed C/P
          </Link>
        </div>
      </div>
      <div className="app-nav-right">
        {backTo && backTo !== location.pathname && (
          <button className="btn-back-nav" onClick={() => navigate(backTo)}>
            ← {backLabel ?? 'Back'}
          </button>
        )}

        <div className="nav-user-menu" ref={menuRef}>
          <button
            className="nav-avatar-btn"
            onClick={() => setMenuOpen(o => !o)}
            aria-label="User menu"
          >
            <span className="nav-avatar">{initials}</span>
            <span className="nav-avatar-chevron">{menuOpen ? '▲' : '▾'}</span>
          </button>

          {menuOpen && (
            <div className="nav-dropdown">
              <div className="nav-dropdown-header">
                <span className="nav-dropdown-name">{user?.full_name}</span>
                <span className="nav-dropdown-email">{user?.email}</span>
                <span className="nav-dropdown-role">{roleLabels[user?.role ?? ''] ?? user?.role}</span>
              </div>
              <div className="nav-dropdown-divider" />
              <button className="nav-dropdown-item nav-dropdown-signout" onClick={handleSignout}>
                Sign Out
              </button>
            </div>
          )}
        </div>
      </div>
    </nav>
  );
}
