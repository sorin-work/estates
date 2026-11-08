import { useQuery } from '@tanstack/react-query';
import { Suspense, useState } from 'react';
import { NavLink, Outlet } from 'react-router';
import { getReminders } from '../api/dashboard';
import { useAuth } from '../auth/AuthContext';
import SearchBox from './SearchBox';
import { Loading } from './States';

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
  { to: '/activity', label: 'Activity' },
];

export default function Layout() {
  const { user, logout } = useAuth();
  const [open, setOpen] = useState(false);
  const reminders = useQuery({ queryKey: ['reminders'], queryFn: getReminders });
  const pressing = reminders.data?.filter((r) => r.severity !== 'info').length ?? 0;

  return (
    <div className="app">
      <a className="skip-link" href="#content">
        Skip to content
      </a>
      <aside id="sidebar" className={open ? 'sidebar open' : 'sidebar'}>
        <div className="brand">
          <img src="/favicon.svg" alt="" width={28} height={28} />
          <span>Estates</span>
        </div>
        <SearchBox onNavigate={() => setOpen(false)} />
        <nav aria-label="Main">
          {nav.map((item) => (
            <NavLink key={item.to} to={item.to} end={item.end} onClick={() => setOpen(false)}>
              {item.label}
              {item.to === '/' && pressing > 0 && (
                <span className="nav-count" aria-label={`${pressing} things need attention`}>
                  {pressing}
                </span>
              )}
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
          <button
            type="button"
            className="menu-button"
            aria-label="Toggle navigation"
            aria-expanded={open}
            aria-controls="sidebar"
            onClick={() => setOpen(!open)}
          >
            ☰
          </button>
          <span className="topbar-title">Estates</span>
        </header>
        <main className="content" id="content" tabIndex={-1}>
          {/* Pages load on demand; keep the sidebar while one does. */}
          <Suspense fallback={<Loading />}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
