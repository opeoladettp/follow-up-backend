import { useState, useEffect } from 'react'
import { Plus, Trash2, Edit2, Check, X, Rss, AlertCircle } from 'lucide-react'
import { api } from '../services/api'

export default function RSSFeedManager({ user }) {
  const [feeds, setFeeds] = useState([])
  const [loading, setLoading] = useState(true)
  const [showAddForm, setShowAddForm] = useState(false)
  const [editingId, setEditingId] = useState(null)
  const [formData, setFormData] = useState({
    name: '',
    url: '',
    category: ''
  })

  useEffect(() => {
    loadFeeds()
  }, [])

  const loadFeeds = async () => {
    try {
      setLoading(true)
      const data = await api.getRSSFeeds()
      setFeeds(data.feeds || [])
    } catch (error) {
      console.error('Failed to load feeds:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    try {
      if (editingId) {
        await api.updateRSSFeed(editingId, formData)
      } else {
        const result = await api.addRSSFeed(formData)
        if (result.resolved_url && result.resolved_url !== formData.url) {
          alert(`Feed added. X handle resolved to:\n${result.resolved_url}`)
        }
      }
      await loadFeeds()
      resetForm()
    } catch (error) {
      console.error('Failed to save feed:', error)
      const msg = error?.response?.data?.error || 'Failed to save feed. Please try again.'
      alert(msg)
    }
  }

  const handleDelete = async (id) => {
    if (!confirm('Are you sure you want to delete this feed?')) return
    try {
      await api.deleteRSSFeed(id)
      await loadFeeds()
    } catch (error) {
      console.error('Failed to delete feed:', error)
      alert('Failed to delete feed. Please try again.')
    }
  }

  const handleEdit = (feed) => {
    setEditingId(feed.id || feed._id)
    setFormData({
      name: feed.name,
      url: feed.url,
      category: feed.category || ''
    })
    setShowAddForm(true)
  }

  const resetForm = () => {
    setFormData({ name: '', url: '', category: '' })
    setEditingId(null)
    setShowAddForm(false)
  }

  // Check if user is editor or admin
  const canManage = user?.role === 'editor' || user?.role === 'admin'

  if (!canManage) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center p-8">
          <AlertCircle className="w-16 h-16 text-gray-400 mx-auto mb-4" />
          <h2 className="text-xl font-semibold text-gray-900 mb-2">Access Restricted</h2>
          <p className="text-gray-600">Only Editors and Admins can manage RSS feeds.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <div className="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800 p-4 sm:p-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">RSS Feed Management</h1>
            <p className="text-gray-600 dark:text-gray-400 mt-1">Manage news sources for your newsroom</p>
          </div>
          {!showAddForm && (
            <button
              onClick={() => setShowAddForm(true)}
              className="flex items-center gap-2 px-4 py-2 bg-gray-900 dark:bg-white text-white dark:text-gray-900 rounded-lg hover:bg-gray-800 dark:hover:bg-gray-100 transition-colors"
            >
              <Plus className="w-5 h-5" />
              <span className="hidden sm:inline">Add Feed</span>
            </button>
          )}
        </div>
      </div>

      {/* Add/Edit Form */}
      {showAddForm && (
        <div className="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800 p-4 sm:p-6">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Feed Name
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg focus:ring-2 focus:ring-gray-900 dark:focus:ring-white focus:border-transparent text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400"
                  placeholder="e.g., BBC News"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Feed URL or X/Twitter Handle
                </label>
                <input
                  type="text"
                  value={formData.url}
                  onChange={(e) => setFormData({ ...formData, url: e.target.value })}
                  className="w-full px-3 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg focus:ring-2 focus:ring-gray-900 dark:focus:ring-white focus:border-transparent text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400"
                  placeholder="https://feeds.bbci.co.uk/... or @BBCBreaking"
                  required
                />
                <p className="text-xs text-gray-400 mt-1">Enter an RSS URL or an X/Twitter handle (e.g. @BBCBreaking)</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Category
                </label>
                <input
                  type="text"
                  value={formData.category}
                  onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                  className="w-full px-3 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg focus:ring-2 focus:ring-gray-900 dark:focus:ring-white focus:border-transparent text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400"
                  placeholder="e.g., World News"
                  required
                />
              </div>
            </div>
            <div className="flex gap-2">
              <button
                type="submit"
                className="flex items-center gap-2 px-4 py-2 bg-gray-900 dark:bg-white text-white dark:text-gray-900 rounded-lg hover:bg-gray-800 dark:hover:bg-gray-100 transition-colors"
              >
                <Check className="w-4 h-4" />
                {editingId ? 'Update' : 'Add'} Feed
              </button>
              <button
                type="button"
                onClick={resetForm}
                className="flex items-center gap-2 px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
              >
                <X className="w-4 h-4" />
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Feeds List */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-6">
        {loading ? (
          <div className="space-y-4">
            {[1, 2, 3].map(i => (
              <div key={i} className="h-24 bg-white dark:bg-gray-800 rounded-lg animate-pulse" />
            ))}
          </div>
        ) : feeds.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {feeds.map((feed, index) => (
              <div
                key={feed.id || feed.url || index}
                className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 hover:shadow-md transition-shadow"
              >
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <Rss className="w-5 h-5 text-gray-900 dark:text-white" />
                    <h3 className="font-semibold text-gray-900 dark:text-white">{feed.name}</h3>
                  </div>
                  <div className="flex gap-1">
                    <button
                      onClick={() => handleEdit(feed)}
                      className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
                      title="Edit"
                    >
                      <Edit2 className="w-4 h-4 text-gray-600 dark:text-gray-400" />
                    </button>
                    <button
                      onClick={() => handleDelete(feed.id || feed._id)}
                      className="p-1.5 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
                      title="Delete"
                    >
                      <Trash2 className="w-4 h-4 text-red-600 dark:text-red-400" />
                    </button>
                  </div>
                </div>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2 truncate">
                  {feed.url.startsWith('twitter://') ? `@${feed.url.replace('twitter://', '')}` : feed.url}
                </p>
                <span className="inline-block px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 text-xs rounded-full">
                  {feed.category}
                </span>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <Rss className="w-16 h-16 text-gray-400 dark:text-gray-600 mx-auto mb-4" />
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">No RSS Feeds</h3>
            <p className="text-gray-600 dark:text-gray-400 mb-4">Add your first RSS feed to get started</p>
            <button
              onClick={() => setShowAddForm(true)}
              className="inline-flex items-center gap-2 px-4 py-2 bg-gray-900 dark:bg-white text-white dark:text-gray-900 rounded-lg hover:bg-gray-800 dark:hover:bg-gray-100 transition-colors"
            >
              <Plus className="w-5 h-5" />
              Add Feed
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
