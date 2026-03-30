import axios from 'axios'

const API_BASE = import.meta.env.VITE_API_URL
  ? `${import.meta.env.VITE_API_URL}/api/v1`
  : '/api/v1'

const apiClient = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add auth token to requests
apiClient.interceptors.request.use((config) => {
  const userId = localStorage.getItem('userId')
  const user = JSON.parse(localStorage.getItem('user') || '{}')
  
  if (userId) {
    config.headers['X-User-ID'] = userId
  }
  if (user.role) {
    config.headers['X-User-Role'] = user.role
  }
  
  return config
})

export const api = {
  // Authentication
  googleAuth: async (userData) => {
    const response = await apiClient.post('/auth/google', userData)
    return response.data
  },

  getCurrentUser: async () => {
    const response = await apiClient.get('/auth/me')
    return response.data
  },

  logout: async () => {
    const response = await apiClient.post('/auth/logout')
    localStorage.removeItem('user')
    localStorage.removeItem('userId')
    return response.data
  },

  // Stories
  getStories: async () => {
    const response = await apiClient.get('/stories')
    return response.data
  },

  createStory: async (storyData) => {
    const response = await apiClient.post('/stories', storyData)
    return response.data
  },

  getStoryContext: async (storyId) => {
    const response = await apiClient.get(`/stories/${storyId}/context`)
    return response.data
  },

  // RSS Feed Management (Editor/Admin only)
  getRSSFeeds: async () => {
    const response = await apiClient.get('/rss/feeds')
    return response.data
  },

  addRSSFeed: async (feedData) => {
    const response = await apiClient.post('/rss/feeds', feedData)
    return response.data
  },

  updateRSSFeed: async (feedId, feedData) => {
    const response = await apiClient.put(`/rss/feeds/${feedId}`, feedData)
    return response.data
  },

  deleteRSSFeed: async (feedId) => {
    const response = await apiClient.delete(`/rss/feeds/${feedId}`)
    return response.data
  },

  removeRSSFeed: async (feedUrl) => {
    const response = await apiClient.delete('/rss/feeds', {
      data: { feed_url: feedUrl },
    })
    return response.data
  },

  // RSS Headlines
  getRSSHeadlines: async () => {
    const response = await apiClient.get('/rss/headlines')
    return response.data
  },

  getRSSHeadlinesBySource: async (source) => {
    const response = await apiClient.get(`/rss/headlines/${source}`)
    return response.data
  },

  // News Reports
  generateNewsReport: async (data) => {
    const response = await apiClient.post('/rss/generate-report', data)
    return response.data
  },

  // Search and generate story from user query
  searchAndGenerateStory: async (query, authorName = '') => {
    const response = await apiClient.post('/rss/search-story', {
      query: query,
      author_name: authorName
    })
    return response.data
  },

  refineNewsReport: async (currentScript, userFeedback, authorName) => {
    const response = await apiClient.post('/rss/refine-report', {
      current_script: currentScript,
      user_feedback: userFeedback,
      author_name: authorName,
    })
    return response.data
  },

  saveReport: async (data) => {
    const response = await apiClient.post('/rss/save-report', data)
    return response.data
  },

  getReportByTitle: async (title) => {
    const response = await apiClient.get('/rss/report-by-title', { params: { title } })
    return response.data
  },

  refreshHeadlines: async () => {
    const response = await apiClient.post('/rss/refresh-headlines')
    return response.data
  },

  updateReportImages: async (reportId, images) => {
    const response = await apiClient.patch(`/rss/report/${reportId}/images`, { images })
    return response.data
  },

  generateVideo: async (data) => {
    const response = await apiClient.post('/rss/generate-video', data)
    return response.data
  },

  cloneVoice: async (formData) => {
    const response = await apiClient.post('/rss/clone-voice', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    return response.data
  },

  generateStoryMedia: async (data) => {
    const response = await apiClient.post('/rss/generate-media', data)
    return response.data
  },

  getReportStatus: async (reportId) => {
    const response = await apiClient.get(`/rss/report-status/${reportId}`)
    return response.data
  },

  // Admin - User Management
  getAllUsers: async () => {
    const response = await apiClient.get('/admin/users')
    return response.data
  },

  updateUserRole: async (userId, role) => {
    const response = await apiClient.put(`/admin/users/${userId}/role`, { role })
    return response.data
  },

  // Health
  getHealth: async () => {
    const response = await apiClient.get('/health')
    return response.data
  },

  // KPI
  updateKPI: async () => {
    const response = await apiClient.post('/kpi/update')
    return response.data
  },
}
