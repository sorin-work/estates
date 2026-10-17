import { lazy, Suspense } from 'react';
import { Route, Routes } from 'react-router';
import { GuestOnly, RequireAuth } from './auth/guards';
import Layout from './components/Layout';
import { Loading } from './components/States';
import DashboardPage from './pages/DashboardPage';
import LoginPage from './pages/LoginPage';
import NotFoundPage from './pages/NotFoundPage';
import RegisterPage from './pages/RegisterPage';

// The dashboard and sign-in pages are part of the main bundle; the rest are
// loaded when first visited.
const ActivityPage = lazy(() => import('./pages/ActivityPage'));
const ExpensesPage = lazy(() => import('./pages/ExpensesPage'));
const LeaseDetailPage = lazy(() => import('./pages/LeaseDetailPage'));
const LeasesPage = lazy(() => import('./pages/LeasesPage'));
const MaintenancePage = lazy(() => import('./pages/MaintenancePage'));
const PropertiesPage = lazy(() => import('./pages/PropertiesPage'));
const PropertyDetailPage = lazy(() => import('./pages/PropertyDetailPage'));
const ReceiptPage = lazy(() => import('./pages/ReceiptPage'));
const RentPage = lazy(() => import('./pages/RentPage'));
const ReportsPage = lazy(() => import('./pages/ReportsPage'));
const SettingsPage = lazy(() => import('./pages/SettingsPage'));
const TenantDetailPage = lazy(() => import('./pages/TenantDetailPage'));
const TenantsPage = lazy(() => import('./pages/TenantsPage'));
const VendorDetailPage = lazy(() => import('./pages/VendorDetailPage'));
const VendorsPage = lazy(() => import('./pages/VendorsPage'));

export default function App() {
  return (
    <Suspense fallback={<Loading />}>
      <Routes>
        <Route element={<GuestOnly />}>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
        </Route>
        <Route element={<RequireAuth />}>
          <Route path="payments/:id/receipt" element={<ReceiptPage />} />
          <Route element={<Layout />}>
            <Route index element={<DashboardPage />} />
            <Route path="properties" element={<PropertiesPage />} />
            <Route path="properties/:id" element={<PropertyDetailPage />} />
            <Route path="tenants" element={<TenantsPage />} />
            <Route path="tenants/:id" element={<TenantDetailPage />} />
            <Route path="leases" element={<LeasesPage />} />
            <Route path="leases/:id" element={<LeaseDetailPage />} />
            <Route path="rent" element={<RentPage />} />
            <Route path="maintenance" element={<MaintenancePage />} />
            <Route path="vendors" element={<VendorsPage />} />
            <Route path="vendors/:id" element={<VendorDetailPage />} />
            <Route path="expenses" element={<ExpensesPage />} />
            <Route path="reports" element={<ReportsPage />} />
            <Route path="activity" element={<ActivityPage />} />
            <Route path="settings" element={<SettingsPage />} />
            <Route path="*" element={<NotFoundPage />} />
          </Route>
        </Route>
      </Routes>
    </Suspense>
  );
}
