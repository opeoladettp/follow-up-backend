import { useState } from 'react'
import { Send, Sparkles, Loader2 } from 'lucide-react'
import { api } from '../services/api'

export default function ChatInterface({ onCreateStory, onStoryCreated, onOpenEditor }) {
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [thinkingSteps, setThinkingSteps] = useState([])
  const [showThinking, setShowThinking] = useState(false)

  const addThinkingStep = (step) => {
    setThinkingSteps(prev => [...prev, { ...step, timestamp: new Date() }])
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!input.trim() || loading) return

    const userQuery = input.trim()
    setInput('')
    setLoading(true)
    setThinkingSteps([])
    setShowThinking(true)

    try {
      addThinkingStep({ type: 'info', message: 'Searching the internet for information...' })
      
      // Call the new search endpoint
      const result = await api.searchAndGenerateStory(userQuery, '')
      
      addThinkingStep({ type: 'info', message: 'Gathering world reactions...' })
      addThinkingStep({ type: 'info', message: 'Generating comprehensive story...' })
      
      // Create a headline object that can be passed to the editor
      const headline = {
        id: `search-${Date.now()}`,
        title: result.title || userQuery,
        description: result.description || result.report || '',
        url: result.url || '',
        source: 'search',
        report: result.report || result.script || '',
        reactions: result.reactions || []
      }
      
      addThinkingStep({ type: 'success', message: 'Story created successfully!' })
      
      // Open the editor with the search result
      if (onOpenEditor) {
        setTimeout(() => {
          onOpenEditor(headline)
        }, 500)
      }
      
      // Keep thinking steps visible for 2 seconds then hide
      setTimeout(() => {
        setShowThinking(false)
      }, 2000)
      
      onStoryCreated()
    } catch (error) {
      console.error('Failed to create story:', error)
      addThinkingStep({ type: 'error', message: 'Failed to create story. Please try again.' })
      alert('Failed to create story. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex-1 flex flex-col h-full bg-white dark:bg-gray-900">
      {/* Center Content */}
      <div className="flex-1 flex items-center justify-center px-4 sm:px-6">
        <div className="w-full max-w-3xl">
          {/* Welcome Message */}
          <div className="text-center mb-8 sm:mb-12">
            <div className="inline-flex items-center justify-center w-16 h-16 sm:w-20 sm:h-20 bg-gradient-to-br from-brand-primary to-brand-dark rounded-full mb-4 sm:mb-6 shadow-lg">
              <Sparkles className="w-8 h-8 sm:w-10 sm:h-10 text-white" />
            </div>
            <h1 className="text-2xl sm:text-3xl md:text-4xl font-bold text-gray-900 dark:text-white mb-2 sm:mb-3 px-4">
              Welcome to FollowUp News AI
            </h1>
            <p className="text-base sm:text-lg text-gray-600 dark:text-gray-400 px-4">
              Search for any story and get AI-powered insights with world reactions
            </p>
          </div>

          {/* Input Form */}
          <form onSubmit={handleSubmit} className="relative px-2 sm:px-0 mb-6">
            <div className="flex items-center gap-2 sm:gap-3 px-4 sm:px-6 py-3 sm:py-4 bg-white dark:bg-gray-800 border-2 border-gray-200 dark:border-gray-700 rounded-full shadow-lg hover:shadow-xl focus-within:border-brand-primary dark:focus-within:border-brand-primary focus-within:shadow-xl transition-all">
              <input
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="Search for any story topic..."
                className="flex-1 text-sm sm:text-base focus:outline-none bg-transparent text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400"
                disabled={loading}
              />
              <button
                type="submit"
                disabled={!input.trim() || loading}
                className="p-2 sm:p-3 bg-brand-primary hover:bg-brand-primary-dark text-white rounded-full disabled:opacity-50 disabled:cursor-not-allowed transition-all hover:scale-105 flex-shrink-0 shadow-md"
              >
                {loading ? (
                  <Loader2 className="w-4 h-4 sm:w-5 sm:h-5 animate-spin" />
                ) : (
                  <Send className="w-4 h-4 sm:w-5 sm:h-5" />
                )}
              </button>
            </div>
          </form>

          {/* Thinking Process */}
          {showThinking && thinkingSteps.length > 0 && (
            <div className="bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4 max-w-2xl mx-auto">
              <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
                <Sparkles className="w-4 h-4" />
                Gemini Thinking Process
              </h3>
              <div className="space-y-2">
                {thinkingSteps.map((step, index) => (
                  <div 
                    key={index}
                    className={`flex items-start gap-2 text-sm p-2 rounded ${
                      step.type === 'error' ? 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400' :
                      step.type === 'success' ? 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400' :
                      'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400'
                    }`}
                  >
                    {loading && step === thinkingSteps[thinkingSteps.length - 1] ? (
                      <Loader2 className="w-4 h-4 mt-0.5 flex-shrink-0 animate-spin" />
                    ) : (
                      <div className="w-4 h-4 mt-0.5 flex-shrink-0 rounded-full bg-current opacity-50" />
                    )}
                    <span>{step.message}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Example Queries */}
          {!loading && thinkingSteps.length === 0 && (
            <div className="mt-8 text-center">
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">Try searching for:</p>
              <div className="flex flex-wrap gap-2 justify-center">
                {[
                  'Latest AI breakthroughs',
                  'Climate change summit',
                  'Tech industry news',
                  'Space exploration',
                  'Global economy trends'
                ].map((example, idx) => (
                  <button
                    key={idx}
                    onClick={() => setInput(example)}
                    className="px-3 py-1 text-xs sm:text-sm bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 rounded-full hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                  >
                    {example}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
