/**
 * Global loading skeleton — shows during route transitions instead of blank screen.
 * Uses animated shimmer placeholders that match the typical page layout.
 */
export default function Loading() {
  return (
    <div className="flex min-h-screen bg-gray-50 dark:bg-gray-950">
      {/* Sidebar skeleton */}
      <div className="hidden md:flex flex-col w-[260px] border-r border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-4 gap-3">
        <div className="h-8 w-8 rounded-lg bg-gray-200 dark:bg-gray-800 animate-pulse" />
        <div className="h-8 rounded-lg bg-gray-100 dark:bg-gray-800/50 animate-pulse" />
        <div className="space-y-2 mt-4">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="h-8 rounded-lg bg-gray-100 dark:bg-gray-800/50 animate-pulse" style={{ animationDelay: `${i * 100}ms` }} />
          ))}
        </div>
        <div className="space-y-2 mt-6">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-6 rounded-lg bg-gray-100 dark:bg-gray-800/50 animate-pulse" style={{ animationDelay: `${(i + 5) * 100}ms` }} />
          ))}
        </div>
      </div>

      {/* Main content skeleton */}
      <div className="flex-1 p-8">
        {/* Title */}
        <div className="h-7 w-48 rounded-lg bg-gray-200 dark:bg-gray-800 animate-pulse mb-2" />
        <div className="h-4 w-72 rounded bg-gray-100 dark:bg-gray-800/50 animate-pulse mb-8" />

        {/* Cards row */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-24 rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-4">
              <div className="h-4 w-20 rounded bg-gray-100 dark:bg-gray-800 animate-pulse mb-3" style={{ animationDelay: `${i * 80}ms` }} />
              <div className="h-7 w-16 rounded bg-gray-200 dark:bg-gray-800 animate-pulse" style={{ animationDelay: `${i * 80 + 100}ms` }} />
            </div>
          ))}
        </div>

        {/* Table skeleton */}
        <div className="rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-4">
          <div className="h-5 w-32 rounded bg-gray-100 dark:bg-gray-800/50 animate-pulse mb-4" />
          <div className="space-y-3">
            {[...Array(6)].map((_, i) => (
              <div key={i} className="flex items-center gap-4">
                <div className="h-4 w-24 rounded bg-gray-100 dark:bg-gray-800/50 animate-pulse" style={{ animationDelay: `${i * 60}ms` }} />
                <div className="h-4 flex-1 rounded bg-gray-100 dark:bg-gray-800/50 animate-pulse" style={{ animationDelay: `${i * 60 + 30}ms` }} />
                <div className="h-4 w-16 rounded bg-gray-100 dark:bg-gray-800/50 animate-pulse" style={{ animationDelay: `${i * 60 + 60}ms` }} />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
