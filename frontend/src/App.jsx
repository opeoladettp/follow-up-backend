import { useState, useEffect } from 'react'
import GeminiSidebar from './components/GeminiSidebar'
import GoogleAuth from './components/GoogleAuth'
import ChatInterface from './components/ChatInterface'
import StoryPanel from './components/StoryPanel'
import HeadlinesBrowser from './components/HeadlinesBrowser'
import ReportEditor from './components/ReportEditor'
import RSSFeedManager from './components/RSSFeedManager'
import { api } from './services/api'
import './styles/brand.css'

function App() {
  const [user, setUser] = useState(null)
  const [stories, setStories] = useState([])
  const [selectedStory, setSelectedStory] = useState(null)
  const [loading, setLoading] = useState(true)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [mode, setMode] = useState('chat') // 'chat', 'headlines', 'editor', 'followup', 'rss-manager'
  const [selectedHeadline, setSelectedHeadline] = useState(null)
  const [headlinesCache, setHeadlinesCache] = useState(null)
  const [headlinesLoading, setHeadlinesLoading] = useState(false)

  useEffect(() => {
    // Check if user is already logged in
    const storedUser = localStorage.getItem('user')
    if (storedUser) {
      try {
        const parsedUser = JSON.parse(storedUser)
        setUser(parsedUser)
        loadStories()
        // Preload headlines on app start
        loadHeadlines()
      } catch (error) {
        console.error('Failed to parse stored user:', error)
        localStorage.removeItem('user')
        localStorage.removeItem('userId')
        setLoading(false)
      }
    } else {
      setLoading(false)
    }
  }, [])

  const loadHeadlines = async () => {
    if (headlinesCache) return // Already cached
    
    try {
      setHeadlinesLoading(true)
      const data = await api.getRSSHeadlines()
      setHeadlinesCache(data.headlines || [])
    } catch (error) {
      console.error('Failed to preload headlines:', error)
    } finally {
      setHeadlinesLoading(false)
    }
  }

  const loadStories = async () => {
    try {
      setLoading(true)
      const data = await api.getStories()
      setStories(data.stories || [])
    } catch (error) {
      console.error('Failed to load stories:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleLoginSuccess = (userData) => {
    setUser(userData)
    loadStories()
  }

  const handleLogout = async () => {
    try {
      await api.logout()
      setUser(null)
      setStories([])
      setSelectedStory(null)
      setMode('chat')
    } catch (error) {
      console.error('Logout failed:', error)
    }
  }

  const handleCreateStory = async (storyData) => {
    try {
      const result = await api.createStory(storyData)
      await loadStories()
      return result
    } catch (error) {
      console.error('Failed to create story:', error)
      throw error
    }
  }

  const handleSelectStory = async (story) => {
    try {
      const context = await api.getStoryContext(story._id || story.id)
      setSelectedStory({ ...story, context })
      setMode('story')
    } catch (error) {
      console.error('Failed to load story context:', error)
      setSelectedStory(story)
      setMode('story')
    }
  }

  const handleSelectHeadline = (headline) => {
    setSelectedHeadline(headline)
    setMode('editor')
  }

  const handleBackToHeadlines = () => {
    setSelectedHeadline(null)
    setMode('headlines')
  }

  const handleOpenEditor = (headline) => {
    setSelectedHeadline(headline)
    setMode('editor')
  }

  const handleNewStory = () => {
    setSelectedStory(null)
    setMode('chat')
  }

  const handleBrowseHeadlines = () => {
    setMode('headlines')
  }

  const handleFollowUpStories = () => {
    setMode('followup')
  }

  const handleManageRSSFeeds = () => {
    setMode('rss-manager')
  }

  // Show login screen if not authenticated
  if (!user) {
    return <GoogleAuth onLoginSuccess={handleLoginSuccess} />
  }

  return (
    <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
      {/* Gemini-style Sidebar */}
      <GeminiSidebar
        stories={stories}
        selectedStory={selectedStory}
        onSelectStory={handleSelectStory}
        onNewStory={handleNewStory}
        onBrowseHeadlines={handleBrowseHeadlines}
        onFollowUpStories={handleFollowUpStories}
        isOpen={sidebarOpen}
        onToggle={() => setSidebarOpen(!sidebarOpen)}
        loading={loading}
        currentMode={mode}
        user={user}
        onLogout={handleLogout}
      />

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Top Navigation Bar */}
        <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-4 sm:px-6 py-3 flex items-center justify-between">
          <h2 className="text-base sm:text-lg font-semibold text-gray-900 dark:text-white truncate">
            {mode === 'chat' && 'New Story'}
            {mode === 'story' && 'Story Details'}
            {mode === 'headlines' && 'RSS Headlines'}
            {mode === 'editor' && 'Report Editor'}
            {mode === 'followup' && 'Follow-up Stories'}
            {mode === 'rss-manager' && 'RSS Feed Management'}
          </h2>
          
          {(user.role === 'editor' || user.role === 'admin') && mode === 'headlines' && (
            <button
              onClick={handleManageRSSFeeds}
              className="px-3 sm:px-4 py-2 bg-gray-900 dark:bg-white text-white dark:text-gray-900 rounded-lg hover:bg-gray-800 dark:hover:bg-gray-100 transition-colors text-sm"
            >
              <span className="hidden sm:inline">Manage RSS Feeds</span>
              <span className="sm:hidden">Manage</span>
            </button>
          )}
          
          {mode === 'rss-manager' && (
            <button
              onClick={handleBrowseHeadlines}
              className="px-3 sm:px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors text-sm"
            >
              Back
            </button>
          )}
        </div>

        {/* Content Area */}
        <div className="flex-1 overflow-hidden">
          {mode === 'story' && selectedStory ? (
            <StoryPanel
              story={selectedStory}
              onClose={handleNewStory}
            />
          ) : mode === 'headlines' ? (
            <HeadlinesBrowser
              onSelectHeadline={handleSelectHeadline}
              cachedHeadlines={headlinesCache}
              onRefresh={loadHeadlines}
            />
          ) : mode === 'editor' && selectedHeadline ? (
            <ReportEditor
              headline={selectedHeadline}
              onBack={handleBackToHeadlines}
              user={user}
              onStoryCreated={loadStories}
            />
          ) : mode === 'followup' ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-center">
                <h3 className="text-xl font-semibold text-gray-900 mb-2">Follow-up Stories</h3>
                <p className="text-gray-600">Track ongoing stories and their developments</p>
                <p className="text-sm text-gray-500 mt-4">Coming soon...</p>
              </div>
            </div>
          ) : mode === 'rss-manager' ? (
            <RSSFeedManager user={user} />
          ) : (
            <ChatInterface
              onCreateStory={handleCreateStory}
              onStoryCreated={loadStories}
              onOpenEditor={handleOpenEditor}
            />
          )}
        </div>
      </div>
    </div>
  )
}

export default App
