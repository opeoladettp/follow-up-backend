import { X, Calendar, ExternalLink, TrendingUp, Clock, Video } from 'lucide-react'
import { format } from 'date-fns'

export default function StoryPanel({ story, onClose }) {
  const developments = story.context?.developments || []

  return (
    <div className="flex-1 flex flex-col h-full bg-white">
      {/* Header */}
      <div className="border-b border-gray-200 px-6 py-4">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-green-100 text-green-700 text-xs font-medium rounded">
                {story.status || 'Active'}
              </span>
              <span className="text-xs text-gray-500">
                {format(new Date(story.created_at), 'MMM d, yyyy')}
              </span>
            </div>
            <h1 className="text-2xl font-bold text-gray-900 mb-2">
              {story.title}
            </h1>
            {story.description && (
              <p className="text-gray-600">{story.description}</p>
            )}
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-6 py-6">
        <div className="max-w-4xl mx-auto">
          {/* Stats */}
          <div className="grid grid-cols-3 gap-4 mb-8">
            <div className="bg-blue-50 rounded-lg p-4">
              <div className="flex items-center gap-2 mb-1">
                <TrendingUp className="w-4 h-4 text-blue-600" />
                <span className="text-xs font-medium text-blue-900">Developments</span>
              </div>
              <p className="text-2xl font-bold text-blue-900">{developments.length}</p>
            </div>
            <div className="bg-purple-50 rounded-lg p-4">
              <div className="flex items-center gap-2 mb-1">
                <Clock className="w-4 h-4 text-purple-600" />
                <span className="text-xs font-medium text-purple-900">Last Update</span>
              </div>
              <p className="text-sm font-semibold text-purple-900">
                {story.updated_at ? format(new Date(story.updated_at), 'MMM d') : 'N/A'}
              </p>
            </div>
            <div className="bg-yellow-50 rounded-lg p-4">
              <div className="flex items-center gap-2 mb-1">
                <Video className="w-4 h-4 text-yellow-600" />
                <span className="text-xs font-medium text-yellow-900">Videos</span>
              </div>
              <p className="text-2xl font-bold text-yellow-900">0</p>
            </div>
          </div>

          {/* Timeline */}
          <div className="mb-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
              <Calendar className="w-5 h-5" />
              Story Timeline
            </h2>

            {developments.length > 0 ? (
              <div className="space-y-4">
                {developments.map((dev, index) => (
                  <div key={dev.id || index} className="relative pl-8 pb-8 border-l-2 border-gray-200 last:border-l-0 last:pb-0">
                    <div className="absolute left-0 top-0 -translate-x-1/2 w-4 h-4 bg-primary-600 rounded-full border-4 border-white" />
                    <div className="bg-gray-50 rounded-lg p-4">
                      <div className="flex items-center justify-between mb-2">
                        <span className={`px-2 py-1 text-xs font-medium rounded ${
                          dev.type === 'breaking' 
                            ? 'bg-red-100 text-red-700' 
                            : 'bg-blue-100 text-blue-700'
                        }`}>
                          {dev.type || 'Update'}
                        </span>
                        <span className="text-xs text-gray-500">
                          {dev.timestamp ? format(new Date(dev.timestamp), 'MMM d, h:mm a') : 'N/A'}
                        </span>
                      </div>
                      <p className="text-gray-900">{dev.content || dev.summary}</p>
                      {dev.source && (
                        <a
                          href={dev.source}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-1 text-xs text-primary-600 hover:text-primary-700 mt-2"
                        >
                          View source
                          <ExternalLink className="w-3 h-3" />
                        </a>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-12 bg-gray-50 rounded-lg">
                <Calendar className="w-12 h-12 mx-auto mb-3 text-gray-400" />
                <p className="text-gray-600 font-medium mb-1">No developments yet</p>
                <p className="text-sm text-gray-500">
                  We're monitoring this story and will notify you of any updates
                </p>
              </div>
            )}
          </div>

          {/* Actions */}
          <div className="flex gap-3">
            <button className="flex-1 px-4 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors font-medium">
              Generate Video Summary
            </button>
            <button className="px-4 py-3 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors font-medium">
              Export Timeline
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
