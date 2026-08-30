import { useInfiniteQuery } from '@tanstack/react-query';
import { listActivity } from '../api/activity';
import ActivityList from '../components/ActivityList';
import PageHeader from '../components/PageHeader';
import { EmptyState, ErrorState, Loading } from '../components/States';

const PAGE_SIZE = 50;

export default function ActivityPage() {
  const query = useInfiniteQuery({
    queryKey: ['activity', 'all'],
    queryFn: ({ pageParam }) => listActivity({ before: pageParam || undefined, limit: PAGE_SIZE }),
    initialPageParam: 0,
    getNextPageParam: (last) => (last.length === PAGE_SIZE ? last.at(-1)?.id : undefined),
  });

  const entries = query.data?.pages.flat() ?? [];

  return (
    <>
      <PageHeader title="Activity" subtitle="Everything that was added, changed or deleted, newest first." />
      {query.isPending && <Loading />}
      {query.error && <ErrorState error={query.error} />}
      {query.data && entries.length === 0 && <EmptyState title="Nothing has happened yet" />}
      {entries.length > 0 && (
        <section className="card">
          <ActivityList entries={entries} />
          {query.hasNextPage && (
            <div className="pager">
              <button
                type="button"
                className="button small secondary"
                disabled={query.isFetchingNextPage}
                onClick={() => void query.fetchNextPage()}
              >
                {query.isFetchingNextPage ? 'Loading…' : 'Show older'}
              </button>
            </div>
          )}
        </section>
      )}
    </>
  );
}
