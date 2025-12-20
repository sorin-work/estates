import { Link } from 'react-router';

export default function NotFoundPage() {
  return (
    <div className="empty">
      <h3>Page not found</h3>
      <p className="muted">
        <Link to="/">Back to the dashboard</Link>
      </p>
    </div>
  );
}
