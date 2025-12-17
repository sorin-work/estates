import { Route, Routes } from 'react-router';
import { useAuth } from './auth/AuthContext';
import { GuestOnly, RequireAuth } from './auth/guards';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';

function Home() {
  const { user, logout } = useAuth();
  return (
    <div className="auth-page">
      <p>
        Signed in as {user?.name}.{' '}
        <button type="button" className="link-button" onClick={logout}>
          Sign out
        </button>
      </p>
    </div>
  );
}

export default function App() {
  return (
    <Routes>
      <Route element={<GuestOnly />}>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
      </Route>
      <Route element={<RequireAuth />}>
        <Route index element={<Home />} />
      </Route>
    </Routes>
  );
}
