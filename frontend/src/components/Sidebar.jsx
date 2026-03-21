import { Menu, Plus, Newspaper, X, Rss } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'

export default function Sidebar({ stories, selectedStory, onSelectStory, onNewStory, onBrowseHeadlines, isOpen, onToggle, loading, currentMode }) {
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
        w-80 bg-white border-r border-gray-200
        transform transition-transform duration-300 ease-in-out
        ${isOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
        flex flex-col
      `}>
        {/* Header */}
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <Newspaper className="w-6 h-6 text-primary-600" />
              <h1 className="text-xl font-bold text-gray-900">FollowUpMedium</h1>
            </div>
            <button
              onClick={onToggle}
              className="lg:hidden p-2 hover:bg-gray-100 rounded-lg transition-colors"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
          
          <button
            onClick={onNewStory}
            className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors font-medium mb-2"
          >
            <Plus className="w-5 h-5" />
            New Story
          </button>
          
          <button
            onClick={onBrowseHeadlines}
            className={`w-full flex items-center justify-center gap-2 px-4 py-3 rounded-lg transition-colors font-medium ${
              currentMode === 'headlines' || currentMode === 'editor'
                ? 'bg-green-600 text-white hover:bg-green-700'
                : 'bg-green-50 text-green-700 hover:bg-green-100'
            }`}
          >
            <Rss className="w-5 h-5" />
            Browse RSS Headlines
          </button>
        </div>

        {/* Stories List */}
        <div className="flex-1 overflow-y-auto p-4 space-y-2">
          {loading ? (
            <div className="space-y-2">
              {[1, 2, 3].map(i => (
                <div key={i} className="h-20 bg-gray-100 rounded-lg animate-pulse" />
              ))}
            </div>
          ) : stories && stories.length > 0 ? (
            stories.map(story => (
              <button
                key={story._id}
                onClick={() => onSelectStory(story)}
                className={`
                  w-full text-left p-3 rounded-lg transition-all
                  ${selectedStory?._id === story._id 
                    ? 'bg-primary-50 border-primary-200 border' 
                    : 'hover:bg-gray-50 border border-transparent'
                  }
                `}
              >
                <h3 className="font-medium text-gray-900 line-clamp-2 mb-1">
                  {story.title}
                </h3>
                <p className="text-xs text-gray-500">
                  {formatDistanceToNow(new Date(story.created_at), { addSuffix: true })}
                </p>
              </button>
            ))
          ) : (
            <div className="text-center py-8 text-gray-500">
              <Newspaper className="w-12 h-12 mx-auto mb-2 opacity-50" />
              <p className="text-sm">No stories yet</p>
              <p className="text-xs mt-1">Create your first story to get started</p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-gray-200">
          <div className="text-xs text-gray-500 text-center">
            AI-Powered News Tracking
          </div>
        </div>
      </div>

      {/* Mobile menu button */}
      {!isOpen && (
        <button
          onClick={onToggle}
          className="fixed top-4 left-4 z-30 lg:hidden p-2 bg-white rounded-lg shadow-lg border border-gray-200"
        >
          <Menu className="w-6 h-6" />
        </button>
      )}
    </>
  )
}
