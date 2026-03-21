import { useState } from 'react'
import { Image as ImageIcon, Share2, Heart, Loader2, AlertCircle, Download, Edit2 } from 'lucide-react'

export default function MediaGallery({ images, reactions, loading, onDownloadImage, onEditImage }) {
  const [selectedImage, setSelectedImage] = useState(null)
  const [brokenImages, setBrokenImages] = useState(new Set())

  const handleImageError = (imageUrl) => {
    setBrokenImages(prev => new Set([...prev, imageUrl]))
  }

  const isBroken = (imageUrl) => brokenImages.has(imageUrl)

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="w-6 h-6 animate-spin text-gray-600 dark:text-gray-400" />
        <span className="ml-2 text-gray-600 dark:text-gray-400">Generating media...</span>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Images Section */}
      {images && images.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
            <ImageIcon className="w-4 h-4" />
            Story Images
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {images.map((image, idx) => (
              <div
                key={idx}
                className="relative group cursor-pointer rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-800 aspect-video"
                onClick={() => !isBroken(image.url) && setSelectedImage(image)}
              >
                {isBroken(image.url) ? (
                  <div className="w-full h-full flex flex-col items-center justify-center bg-gray-200 dark:bg-gray-700">
                    <AlertCircle className="w-8 h-8 text-gray-500 dark:text-gray-400 mb-2" />
                    <p className="text-sm text-gray-600 dark:text-gray-400 text-center px-2">
                      Image failed to load
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-500 mt-1 text-center px-2">
                      {image.type === 'story' && 'Story Image'}
                      {image.type === 'reaction' && 'Reactions'}
                      {image.type === 'background' && 'Background'}
                    </p>
                  </div>
                ) : (
                  <>
                    <img
                      src={image.url}
                      alt={image.description}
                      onError={() => handleImageError(image.url)}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                    />
                    <div className="absolute inset-0 bg-black/0 group-hover:bg-black/30 transition-colors flex items-end p-3">
                      <div className="text-white text-sm font-medium opacity-0 group-hover:opacity-100 transition-opacity flex-1">
                        {image.type === 'story' && '📰 Story Image'}
                        {image.type === 'reaction' && '💬 Reactions'}
                        {image.type === 'background' && '🎨 Background'}
                      </div>
                      {(onDownloadImage || onEditImage) && (
                        <div className="flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                          {onDownloadImage && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation()
                                onDownloadImage(image)
                              }}
                              className="p-2 bg-white/20 hover:bg-white/40 rounded-lg transition-colors"
                              title="Download image"
                            >
                              <Download className="w-4 h-4 text-white" />
                            </button>
                          )}
                          {onEditImage && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation()
                                onEditImage(image)
                              }}
                              className="p-2 bg-white/20 hover:bg-white/40 rounded-lg transition-colors"
                              title="Edit image"
                            >
                              <Edit2 className="w-4 h-4 text-white" />
                            </button>
                          )}
                        </div>
                      )}
                    </div>
                  </>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Reactions Section */}
      {reactions && reactions.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
            <Share2 className="w-4 h-4" />
            Social Media Reactions
          </h3>
          <div className="space-y-3">
            {reactions.map((reaction, idx) => (
              <div
                key={idx}
                className="bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4 hover:border-gray-300 dark:hover:border-gray-600 transition-colors"
              >
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className="inline-block px-2 py-1 bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 text-xs font-medium rounded">
                      {reaction.platform.toUpperCase()}
                    </span>
                    <span className="text-sm font-medium text-gray-900 dark:text-white">
                      @{reaction.author}
                    </span>
                  </div>
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    {reaction.timestamp}
                  </span>
                </div>

                <p className="text-sm text-gray-700 dark:text-gray-300 mb-3 line-clamp-3">
                  {reaction.content}
                </p>

                <div className="flex items-center gap-4 text-xs text-gray-600 dark:text-gray-400">
                  <div className="flex items-center gap-1">
                    <Heart className="w-4 h-4" />
                    <span>{reaction.likes.toLocaleString()}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <Share2 className="w-4 h-4" />
                    <span>{reaction.shares.toLocaleString()}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Image Modal */}
      {selectedImage && !isBroken(selectedImage.url) && (
        <div
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
          onClick={() => setSelectedImage(null)}
        >
          <div
            className="bg-white dark:bg-gray-900 rounded-lg max-w-2xl w-full max-h-[80vh] overflow-auto"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="sticky top-0 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800 p-4 flex items-center justify-between">
              <h3 className="font-medium text-gray-900 dark:text-white">
                {selectedImage.type === 'story' && '📰 Story Image'}
                {selectedImage.type === 'reaction' && '💬 Reactions'}
                {selectedImage.type === 'background' && '🎨 Background'}
              </h3>
              <button
                onClick={() => setSelectedImage(null)}
                className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
              >
                ✕
              </button>
            </div>
            <div className="p-4">
              <img
                src={selectedImage.url}
                alt={selectedImage.description}
                onError={() => handleImageError(selectedImage.url)}
                className="w-full rounded-lg mb-4"
              />
              <div className="space-y-2">
                <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Description
                </p>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  {selectedImage.description}
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-500 mt-4">
                  Source: {selectedImage.source}
                </p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
