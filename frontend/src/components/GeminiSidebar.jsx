import { useState, useEffect, useRef } from 'react'
import { 
  Menu, 
  Plus, 
  Newspaper, 
  Rss, 
  Clock,
  ChevronLeft,
  LogOut,
  Shield
} from 'lucide-react'

export default function GeminiSidebar({ 
  stories, 
  selectedStory, 
  onSelectStory, 
  onNewStory, 
  onBrowseHeadlines,
  onFollowUpStories,
  onManageUsers,
  isOpen, 
  onToggle, 
  loading, 
  currentMode,
  user,
  onLogout 
}) {
  const [pastStories, setPastStories] = useState([])
  const [loadingMore, setLoadingMore] = useState(false)
  const [page, setPage] = useState(1)
  const scrollRef = useRef(null)

  useEffect(() => {
    if (stories && stories.length > 0) {
      setPastStories(stories)
    }
  }, [stories])

  const handleScroll = (e) => {
    const { scrollTop, scrollHeight, clientHeight } = e.target
    if (scrollHeight - scrollTop <= clientHeight * 1.5 && !loadingMore) {
      loadMoreStories()
    }
  }

  const loadMoreStories = async () => {
    setLoadingMore(true)
    setTimeout(() => {
      setPage(prev => prev + 1)
      setLoadingMore(false)
    }, 1000)
  }

  return (
    <>
      {/* Mobile overlay */}
      {isOpen && (
        <div 
          className="fixed inset-0 bg-black bg-opacity-50 z-40 lg:hidden"
          onClick={onToggle}
        />
      )}

      {/* Sidebar */}
      <div className={`
        fixed lg:relative inset-y-0 left-0 z-50
        ${isOpen ? 'w-72' : 'w-0 lg:w-16'}
        bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800
        transform transition-all duration-300 ease-in-out
        flex flex-col overflow-hidden
      `}>
        {/* Header */}
        <div className="p-4 border-b border-gray-200 dark:border-gray-800 flex-shrink-0 flex items-center justify-between">
          {isOpen && (
            <img src="/logo.png" alt="FollowUp" className="h-24 object-contain" />
          )}
          <button
            onClick={onToggle}
            className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
            title={isOpen ? "Collapse sidebar" : "Expand sidebar"}
          >
            {isOpen
              ? <ChevronLeft className="w-5 h-5 text-gray-700 dark:text-gray-300" />
              : <Menu className="w-5 h-5 text-gray-700 dark:text-gray-300" />
            }
          </button>
        </div>

        {/* Navigation */}
        {isOpen && (
          <div className="p-3 border-b border-gray-200 dark:border-gray-800 flex-shrink-0 space-y-2">
            <a
              href="#"
              onClick={(e) => { e.preventDefault(); onNewStory(); }}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
                currentMode === 'chat'
                  ? 'bg-brand-primary text-white font-medium shadow-md'
                  : 'text-gray-700 dark:text-gray-300 hover:text-white hover:bg-brand-primary'
              }`}
            >
              <Plus className="w-5 h-5" />
              <span>New story</span>
            </a>

            <a
              href="#"
              onClick={(e) => { e.preventDefault(); onBrowseHeadlines(); }}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
                currentMode === 'headlines' || currentMode === 'editor'
                  ? 'bg-brand-primary text-white font-medium shadow-md'
                  : 'text-gray-700 dark:text-gray-300 hover:text-white hover:bg-brand-primary'
              }`}
            >
              <Rss className="w-5 h-5" />
              <span>Headlines</span>
            </a>

            <a
              href="#"
              onClick={(e) => { e.preventDefault(); onFollowUpStories(); }}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
                currentMode === 'followup'
                  ? 'bg-brand-primary text-white font-medium shadow-md'
                  : 'text-gray-700 dark:text-gray-300 hover:text-white hover:bg-brand-primary'
              }`}
            >
              <Clock className="w-5 h-5" />
              <span>Follow-up stories</span>
            </a>

            {user?.role === 'admin' && (
              <a
                href="#"
                onClick={(e) => { e.preventDefault(); onManageUsers(); }}
                className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
                  currentMode === 'admin'
                    ? 'bg-brand-primary text-white font-medium shadow-md'
                    : 'text-gray-700 dark:text-gray-300 hover:text-white hover:bg-brand-primary'
                }`}
              >
                <Shield className="w-5 h-5" />
                <span>Manage Users</span>
              </a>
            )}
          </div>
        )}

        {/* Past Stories - Scrollable */}
        {isOpen && (
          <div 
            ref={scrollRef}
            onScroll={handleScroll}
            className="flex-1 overflow-y-auto p-0 space-y-0"
          >
            {loading ? (
              <div className="space-y-1 p-1">
                {[1, 2, 3, 4, 5].map(i => (
                  <div key={i} className="h-6 bg-gray-100 dark:bg-gray-800 rounded animate-pulse" />
                ))}
              </div>
            ) : pastStories && pastStories.length > 0 ? (
              <>
                {pastStories.map(story => (
                  <button
                    key={story._id || story.id}
                    onClick={() => onSelectStory(story)}
                    className={`
                      w-full text-left px-2 py-1 transition-all text-xs flex items-center gap-1 min-w-0
                      ${(selectedStory?.id && selectedStory.id === story.id)
                        ? 'text-brand-primary font-medium'
                        : 'text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white'
                      }
                    `}
                    title={story.title}
                  >
                    <span className="truncate flex-1">{story.title}</span>
                    {story.created_at && (
                      <span className="flex-shrink-0 text-gray-400 dark:text-gray-500 text-[10px]">
                        {new Date(story.created_at).toLocaleDateString('en-GB', { day: '2-digit', month: 'short' })}
                      </span>
                    )}
                  </button>
                ))}
                {loadingMore && (
                  <div className="h-6 bg-gray-100 dark:bg-gray-800 rounded animate-pulse m-1" />
                )}
              </>
            ) : (
              <div className="text-center py-4 text-gray-500 dark:text-gray-400 text-xs">
                <Newspaper className="w-8 h-8 mx-auto mb-1 opacity-50" />
                <p>No stories yet</p>
              </div>
            )}
          </div>
        )}

        {/* User Account - Stuck to bottom */}
        {isOpen && user && (
          <div className="p-3 border-t border-gray-200 dark:border-gray-800 flex-shrink-0">
            <div className="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer group">
              <img 
                src={user.picture || `https://ui-avatars.com/api/?name=${encodeURIComponent(user.name)}`}
                alt={user.name}
                className="w-10 h-10 rounded-full"
              />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 dark:text-white truncate">{user.name}</p>
                <div className="flex items-center gap-1">
                  {user.role === 'admin' && <Shield className="w-3 h-3 text-gray-600 dark:text-gray-400" />}
                  <p className="text-xs text-gray-500 dark:text-gray-400 capitalize">{user.role}</p>
                </div>
              </div>
              <div className="opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  onClick={onLogout}
                  className="p-1.5 hover:bg-gray-200 dark:hover:bg-gray-700 rounded transition-colors"
                  title="Logout"
                >
                  <LogOut className="w-4 h-4 text-gray-600 dark:text-gray-400" />
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Collapsed state icons */}
        {!isOpen && (
          <div className="flex flex-col items-center gap-4 py-4">
            <a
              href="#"
              onClick={(e) => { e.preventDefault(); onNewStory(); }}
              className="p-3 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
              title="New story"
            >
              <Plus className="w-5 h-5 text-gray-700 dark:text-gray-300" />
            </a>
            <a
              href="#"
              onClick={(e) => { e.preventDefault(); onBrowseHeadlines(); }}
              className="p-3 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
              title="Headlines"
            >
              <Rss className="w-5 h-5 text-gray-700 dark:text-gray-300" />
            </a>
            <a
              href="#"
              onClick={(e) => { e.preventDefault(); onFollowUpStories(); }}
              className="p-3 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
              title="Follow-up stories"
            >
              <Clock className="w-5 h-5 text-gray-700 dark:text-gray-300" />
            </a>
          </div>
        )}
      </div>
    </>
  )
}
