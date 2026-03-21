import { useState, useEffect } from 'react'
import { Newspaper, RefreshCw, Sparkles, ChevronRight, Plus, X, Settings } from 'lucide-react'
import { api } from '../services/api'

export default function HeadlinesBrowser({ onSelectHeadline }) {
  const [headlines, setHeadlines] = useState([])
  const [loading, setLoading] = useState(true)
  const [selectedHeadline, setSelectedHeadline] = useState(null)
  const [showAddFeed, setShowAddFeed] = useState(false)
  const [newFeedUrl, setNewFeedUrl] = useState('')
  const [newFeedName, setNewFeedName] = useState('')
  const [feeds, setFeeds] = useState([])
  const [showManageFeeds, setShowManageFeeds] = useState(false)

  useEffect(() => {
    loadHeadlines()
    loadFeeds()
  }, [])

  const loadFeeds = async () => {
    try {
      const data = await api.getRSSFeeds()
      setFeeds(data.feeds || [])
    } catch (error) {
      console.error('Failed to load feeds:', error)
    }
  }

  const loadHeadlines = async () => {
    try {
      setLoading(true)
      const data = await api.getRSSHeadlines()
      setHeadlines(data.headlines || [])
    } catch (error) {
      console.error('Failed to load headlines:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSelectHeadline = (headline) => {
    setSelectedHeadline(headline)
    onSelectHeadline(headline)
  }

  const handleAddFeed = async () => {
    if (!newFeedUrl.trim() || !newFeedName.trim()) {
      alert('Please enter both feed URL and name')
      return
    }

    try {
      await api.addRSSFeed(newFeedUrl, newFeedName)
      setNewFeedUrl('')
      setNewFeedName('')
      setShowAddFeed(false)
      loadFeeds()
      loadHeadlines()
      alert('RSS feed added successfully!')
    } catch (error) {
      console.error('Failed to add feed:', error)
      alert('Failed to add feed. Please check the URL and try again.')
    }
  }

  const handleRemoveFeed = async (feedUrl) => {
    if (!confirm('Are you sure you want to remove this feed?')) {
      return
    }

    try {
      await api.removeRSSFeed(feedUrl)
      loadFeeds()
      loadHeadlines()
      alert('RSS feed removed successfully!')
    } catch (error) {
      console.error('Failed to remove feed:', error)
      alert('Failed to remove feed.')
    }
  }

  return (
    <div className="flex-1 flex flex-col h-full bg-white">
      {/* Header */}
      <div className="border-b border-gray-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
              <Newspaper className="w-5 h-5 text-primary-600" />
              RSS Headlines
            </h2>
            <p className="text-sm text-gray-500">Select a story to create your report</p>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setShowManageFeeds(!showManageFeeds)}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
            >
              <Settings className="w-4 h-4" />
              Manage Feeds
            </button>
            <button
              onClick={loadHeadlines}
              disabled={loading}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>
        </div>

        {/* Manage Feeds Panel */}
        {showManageFeeds && (
          <div className="mt-4 p-4 bg-gray-50 rounded-lg border border-gray-200">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-medium text-gray-900">RSS Feed Sources ({feeds.length})</h3>
              <button
                onClick={() => setShowAddFeed(!showAddFeed)}
                className="flex items-center gap-1 px-3 py-1 text-sm bg-primary-600 text-white rounded hover:bg-primary-700"
              >
                <Plus className="w-4 h-4" />
                Add Feed
              </button>
            </div>

            {showAddFeed && (
              <div className="mb-4 p-3 bg-white rounded border border-gray-300">
                <div className="space-y-2">
                  <input
                    type="text"
                    value={newFeedName}
                    onChange={(e) => setNewFeedName(e.target.value)}
                    placeholder="Feed Name (e.g., TechCrunch)"
                    className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                  />
                  <input
                    type="url"
                    value={newFeedUrl}
                    onChange={(e) => setNewFeedUrl(e.target.value)}
                    placeholder="Feed URL (e.g., https://techcrunch.com/feed/)"
                    className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                  />
                  <div className="flex gap-2">
                    <button
                      onClick={handleAddFeed}
                      className="flex-1 px-3 py-2 bg-primary-600 text-white rounded text-sm hover:bg-primary-700"
                    >
                      Add Feed
                    </button>
                    <button
                      onClick={() => setShowAddFeed(false)}
                      className="px-3 py-2 bg-gray-200 text-gray-700 rounded text-sm hover:bg-gray-300"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </div>
            )}

            <div className="space-y-2 max-h-60 overflow-y-auto">
              {feeds.map((feed, index) => (
                <div key={index} className="flex items-center justify-between p-2 bg-white rounded border border-gray-200">
                  <div className="flex-1">
                    <p className="text-sm font-medium text-gray-900">{feed.name}</p>
                    <p className="text-xs text-gray-500 truncate">{feed.url}</p>
                  </div>
                  <button
                    onClick={() => handleRemoveFeed(feed.url)}
                    className="p-1 text-red-600 hover:bg-red-50 rounded"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Headlines List */}
      <div className="flex-1 overflow-y-auto p-6">
        {loading ? (
          <div className="space-y-4">
            {[1, 2, 3, 4, 5].map(i => (
              <div key={i} className="h-32 bg-gray-100 rounded-lg animate-pulse" />
            ))}
          </div>
        ) : headlines.length > 0 ? (
          <div className="grid gap-4 max-w-5xl mx-auto">
            {headlines.map((headline, index) => (
              <button
                key={headline.id || index}
                onClick={() => handleSelectHeadline(headline)}
                className={`
                  text-left p-4 rounded-lg border-2 transition-all
                  ${selectedHeadline?.id === headline.id
                    ? 'border-primary-500 bg-primary-50'
                    : 'border-gray-200 hover:border-primary-300 hover:shadow-md'
                  }
                `}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <span className="px-2 py-1 bg-blue-100 text-blue-700 text-xs font-medium rounded">
                        {headline.source}
                      </span>
                      <span className="text-xs text-gray-500">
                        {new Date(headline.published_at).toLocaleDateString()}
                      </span>
                    </div>
                    <h3 className="font-semibold text-gray-900 mb-2 line-clamp-2">
                      {headline.title}
                    </h3>
                    {headline.description && (
                      <p className="text-sm text-gray-600 line-clamp-2">
                        {headline.description}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    {selectedHeadline?.id === headline.id ? (
                      <div className="flex items-center gap-2 text-primary-600">
                        <Sparkles className="w-5 h-5" />
                        <span className="text-sm font-medium">Selected</span>
                      </div>
                    ) : (
                      <ChevronRight className="w-5 h-5 text-gray-400" />
                    )}
                  </div>
                </div>
              </button>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <Newspaper className="w-16 h-16 mx-auto mb-4 text-gray-400" />
            <p className="text-gray-600 font-medium mb-2">No headlines available</p>
            <p className="text-sm text-gray-500">Check your RSS feed configuration</p>
          </div>
        )}
      </div>
    </div>
  )
}
