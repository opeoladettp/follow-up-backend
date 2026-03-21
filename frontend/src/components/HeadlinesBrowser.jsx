import { useState, useEffect } from 'react'
import { Newspaper, RefreshCw, Sparkles, ChevronRight, Filter, Search, X, Calendar, Tag } from 'lucide-react'
import { api } from '../services/api'

export default function HeadlinesBrowser({ onSelectHeadline, cachedHeadlines, onRefresh }) {
  const [headlines, setHeadlines] = useState(cachedHeadlines || [])
  const [filteredHeadlines, setFilteredHeadlines] = useState([])
  const [loading, setLoading] = useState(!cachedHeadlines)
  const [selectedHeadline, setSelectedHeadline] = useState(null)
  
  // Filter states
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedSource, setSelectedSource] = useState('all')
  const [sortBy, setSortBy] = useState('date-desc') // date-desc, date-asc, title-asc, title-desc
  const [showFilters, setShowFilters] = useState(false)
  const [dateFilter, setDateFilter] = useState('all') // all, today, week, month

  // Decode HTML entities
  const decodeHTML = (html) => {
    const txt = document.createElement('textarea')
    txt.innerHTML = html
    return txt.value
  }

  useEffect(() => {
    if (cachedHeadlines) {
      setHeadlines(cachedHeadlines)
      setLoading(false)
    } else {
      loadHeadlines()
    }
  }, [cachedHeadlines])

  useEffect(() => {
    applyFilters()
  }, [headlines, searchQuery, selectedSource, sortBy, dateFilter])

  const loadHeadlines = async () => {
    try {
      setLoading(true)
      const data = await api.getRSSHeadlines()
      setHeadlines(data.headlines || [])
      if (onRefresh) {
        onRefresh()
      }
    } catch (error) {
      console.error('Failed to load headlines:', error)
    } finally {
      setLoading(false)
    }
  }

  const applyFilters = () => {
    let filtered = [...headlines]

    // Search filter
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase()
      filtered = filtered.filter(h => 
        h.title.toLowerCase().includes(query) ||
        (h.description && h.description.toLowerCase().includes(query)) ||
        h.source.toLowerCase().includes(query)
      )
    }

    // Source filter
    if (selectedSource !== 'all') {
      filtered = filtered.filter(h => h.source === selectedSource)
    }

    // Date filter
    if (dateFilter !== 'all') {
      const now = new Date()
      const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      
      filtered = filtered.filter(h => {
        const headlineDate = new Date(h.published_at)
        
        switch(dateFilter) {
          case 'today':
            return headlineDate >= today
          case 'week':
            const weekAgo = new Date(today)
            weekAgo.setDate(weekAgo.getDate() - 7)
            return headlineDate >= weekAgo
          case 'month':
            const monthAgo = new Date(today)
            monthAgo.setMonth(monthAgo.getMonth() - 1)
            return headlineDate >= monthAgo
          default:
            return true
        }
      })
    }

    // Sort
    filtered.sort((a, b) => {
      switch(sortBy) {
        case 'date-desc':
          return new Date(b.published_at) - new Date(a.published_at)
        case 'date-asc':
          return new Date(a.published_at) - new Date(b.published_at)
        case 'title-asc':
          return a.title.localeCompare(b.title)
        case 'title-desc':
          return b.title.localeCompare(a.title)
        default:
          return 0
      }
    })

    setFilteredHeadlines(filtered)
  }

  const handleSelectHeadline = (headline) => {
    setSelectedHeadline(headline)
    onSelectHeadline(headline)
  }

  const clearFilters = () => {
    setSearchQuery('')
    setSelectedSource('all')
    setSortBy('date-desc')
    setDateFilter('all')
  }

  const hasActiveFilters = searchQuery || selectedSource !== 'all' || dateFilter !== 'all' || sortBy !== 'date-desc'

  // Get unique sources
  const sources = ['all', ...new Set(headlines.map(h => h.source))]

  return (
    <div className="flex-1 flex flex-col h-full bg-white dark:bg-gray-900">
      {/* Header */}
      <div className="border-b border-gray-200 dark:border-gray-800 px-4 sm:px-6 py-4">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
              <Newspaper className="w-5 h-5 text-gray-900 dark:text-white" />
              RSS Headlines
            </h2>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {filteredHeadlines.length} {filteredHeadlines.length === 1 ? 'headline' : 'headlines'}
              {hasActiveFilters && ` (filtered from ${headlines.length})`}
            </p>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setShowFilters(!showFilters)}
              className={`flex items-center gap-2 px-3 sm:px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
                showFilters || hasActiveFilters
                  ? 'bg-brand-primary text-white shadow-md'
                  : 'text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 hover:border-brand-primary'
              }`}
            >
              <Filter className="w-4 h-4" />
              <span className="hidden sm:inline">Filters</span>
            </button>
            <button
              onClick={loadHeadlines}
              disabled={loading}
              className="flex items-center gap-2 px-3 sm:px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              <span className="hidden sm:inline">Refresh</span>
            </button>
          </div>
        </div>

        {/* Filters Panel */}
        {showFilters && (
          <div className="space-y-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
            {/* Search */}
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                <Search className="w-4 h-4 inline mr-1" />
                Search
              </label>
              <div className="relative">
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search headlines..."
                  className="w-full px-4 py-2 pr-10 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-white text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400"
                />
                {searchQuery && (
                  <button
                    onClick={() => setSearchQuery('')}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                  >
                    <X className="w-4 h-4" />
                  </button>
                )}
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              {/* Source Filter */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  <Tag className="w-4 h-4 inline mr-1" />
                  Source
                </label>
                <select
                  value={selectedSource}
                  onChange={(e) => setSelectedSource(e.target.value)}
                  className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-white text-gray-900 dark:text-white"
                >
                  {sources.map(source => (
                    <option key={source} value={source}>
                      {source === 'all' ? 'All Sources' : source}
                    </option>
                  ))}
                </select>
              </div>

              {/* Date Filter */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  <Calendar className="w-4 h-4 inline mr-1" />
                  Date Range
                </label>
                <select
                  value={dateFilter}
                  onChange={(e) => setDateFilter(e.target.value)}
                  className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-white text-gray-900 dark:text-white"
                >
                  <option value="all">All Time</option>
                  <option value="today">Today</option>
                  <option value="week">Past Week</option>
                  <option value="month">Past Month</option>
                </select>
              </div>

              {/* Sort */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Sort By
                </label>
                <select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value)}
                  className="w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-white text-gray-900 dark:text-white"
                >
                  <option value="date-desc">Newest First</option>
                  <option value="date-asc">Oldest First</option>
                  <option value="title-asc">Title (A-Z)</option>
                  <option value="title-desc">Title (Z-A)</option>
                </select>
              </div>
            </div>

            {/* Clear Filters */}
            {hasActiveFilters && (
              <div className="flex justify-end">
                <button
                  onClick={clearFilters}
                  className="flex items-center gap-2 px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white"
                >
                  <X className="w-4 h-4" />
                  Clear All Filters
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Headlines List */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-6">
        {loading ? (
          <div className="space-y-4">
            {[1, 2, 3, 4, 5].map(i => (
              <div key={i} className="h-32 bg-gray-100 dark:bg-gray-800 rounded-lg animate-pulse" />
            ))}
          </div>
        ) : filteredHeadlines.length > 0 ? (
          <div className="grid gap-4 max-w-5xl mx-auto">
            {filteredHeadlines.map((headline, index) => (
              <button
                key={headline.id || index}
                onClick={() => handleSelectHeadline(headline)}
                className={`
                  text-left p-4 rounded-lg border-2 transition-all
                  ${selectedHeadline?.id === headline.id
                    ? 'border-brand-primary bg-orange-50 dark:bg-gray-800'
                    : 'border-gray-200 dark:border-gray-700 hover:border-brand-primary hover:shadow-md'
                  }
                `}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-2 flex-wrap">
                      <span className="px-2 py-1 bg-brand-primary text-white text-xs font-medium rounded">
                        {headline.source}
                      </span>
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        {new Date(headline.published_at).toLocaleDateString('en-US', { 
                          month: 'short', 
                          day: 'numeric',
                          year: 'numeric',
                          hour: '2-digit',
                          minute: '2-digit'
                        })}
                      </span>
                    </div>
                    <h3 className="font-semibold text-gray-900 dark:text-white mb-2 line-clamp-2">
                      {decodeHTML(headline.title)}
                    </h3>
                    {headline.description && (
                      <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-2">
                        {decodeHTML(headline.description)}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    {selectedHeadline?.id === headline.id ? (
                      <div className="flex items-center gap-2 text-gray-900 dark:text-white">
                        <Sparkles className="w-5 h-5" />
                        <span className="text-sm font-medium hidden sm:inline">Selected</span>
                      </div>
                    ) : (
                      <ChevronRight className="w-5 h-5 text-gray-400 dark:text-gray-500" />
                    )}
                  </div>
                </div>
              </button>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <Newspaper className="w-16 h-16 mx-auto mb-4 text-gray-400 dark:text-gray-600" />
            <p className="text-gray-600 dark:text-gray-400 font-medium mb-2">
              {hasActiveFilters ? 'No headlines match your filters' : 'No headlines available'}
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-500">
              {hasActiveFilters ? 'Try adjusting your filters' : 'Check your RSS feed configuration'}
            </p>
            {hasActiveFilters && (
              <button
                onClick={clearFilters}
                className="mt-4 px-4 py-2 bg-gray-900 dark:bg-white text-white dark:text-gray-900 rounded-lg hover:bg-gray-800 dark:hover:bg-gray-100"
              >
                Clear Filters
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
