import { NavLink, Outlet } from 'react-router';
import { useAuth } from '../auth/AuthContext';

const nav = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/properties', label: 'Properties' },
];

export default function Layout() {
  const { user, logout } = useAuth();

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="brand">
          <img src="/favicon.svg" alt="" width={28} height={28} />
          <span>Estates</span>
        </div>
        <nav>
          {nav.map((item) => (
            <NavLink key={item.to} to={item.to} end={item.end}>
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          <div className="user-name" title={user?.email}>
            {user?.name}
          </div>
          <button type="button" className="link-button" onClick={logout}>
            Sign out
          </button>
        </div>
      </aside>
      <div className="main">
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
