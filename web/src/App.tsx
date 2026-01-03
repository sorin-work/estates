import { Navigate, Route, Routes } from 'react-router';
import { GuestOnly, RequireAuth } from './auth/guards';
import Layout from './components/Layout';
import LeaseDetailPage from './pages/LeaseDetailPage';
import LeasesPage from './pages/LeasesPage';
import LoginPage from './pages/LoginPage';
import NotFoundPage from './pages/NotFoundPage';
import PropertiesPage from './pages/PropertiesPage';
import PropertyDetailPage from './pages/PropertyDetailPage';
import RegisterPage from './pages/RegisterPage';
import RentPage from './pages/RentPage';
import TenantDetailPage from './pages/TenantDetailPage';
import TenantsPage from './pages/TenantsPage';

export default function App() {
  return (
    <Routes>
      <Route element={<GuestOnly />}>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
      </Route>
      <Route element={<RequireAuth />}>
        <Route element={<Layout />}>
          <Route index element={<Navigate to="/properties" replace />} />
          <Route path="properties" element={<PropertiesPage />} />
          <Route path="properties/:id" element={<PropertyDetailPage />} />
          <Route path="tenants" element={<TenantsPage />} />
          <Route path="tenants/:id" element={<TenantDetailPage />} />
          <Route path="leases" element={<LeasesPage />} />
          <Route path="leases/:id" element={<LeaseDetailPage />} />
          <Route path="rent" element={<RentPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Route>
    </Routes>
  );
}
