import { useState } from 'react';
import { NavLink, Outlet } from 'react-router';
import { useAuth } from '../auth/AuthContext';

const nav = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/properties', label: 'Properties' },
  { to: '/tenants', label: 'Tenants' },
  { to: '/leases', label: 'Leases' },
  { to: '/rent', label: 'Rent' },
  { to: '/maintenance', label: 'Maintenance' },
  { to: '/vendors', label: 'Vendors' },
  { to: '/expenses', label: 'Expenses' },
  { to: '/reports', label: 'Reports' },
];

export default function Layout() {
  const { user, logout } = useAuth();
  const [open, setOpen] = useState(false);

  return (
    <div className="app">
      <aside className={open ? 'sidebar open' : 'sidebar'}>
        <div className="brand">
          <img src="/favicon.svg" alt="" width={28} height={28} />
          <span>Estates</span>
        </div>
        <nav>
          {nav.map((item) => (
            <NavLink key={item.to} to={item.to} end={item.end} onClick={() => setOpen(false)}>
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          <div className="user-name" title={user?.email}>
            {user?.name}
          </div>
          <div className="sidebar-links">
            <NavLink to="/settings" onClick={() => setOpen(false)}>
              Settings
            </NavLink>
            <button type="button" className="link-button" onClick={logout}>
              Sign out
            </button>
          </div>
        </div>
      </aside>
      <div className="main">
        <header className="topbar">
          <button type="button" className="menu-button" aria-label="Toggle navigation" onClick={() => setOpen(!open)}>
            ☰
          </button>
          <span className="topbar-title">Estates</span>
        </header>
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
