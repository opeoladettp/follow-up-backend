import { useState, useEffect, useRef } from 'react'
import { Sparkles, Save, Video, Upload, User, Loader2, CheckCircle, XCircle, Plus, Send, Image as ImageIcon, FileText, Mic } from 'lucide-react'
import { api } from '../services/api'
import MediaGallery from './MediaGallery'

// Proxy Google profile pictures through backend to avoid referrer/CORS issues
function getDisplayAvatarUrl(url) {
  if (!url) return null
  if (url.startsWith('data:')) return url
  if (url.includes('googleusercontent.com') || url.includes('lh3.google')) {
    return `/api/v1/rss/proxy-image?url=${encodeURIComponent(url)}`
  }
  return url
}

export default function ReportEditor({ headline, onBack, user, onStoryCreated }) {
  const [script, setScript] = useState('')
  const [author, setAuthor] = useState(user?.name || '')
  const [avatarFile, setAvatarFile] = useState(null)
  const [avatarPreview, setAvatarPreview] = useState(user?.picture || null)
  const [voiceFile, setVoiceFile] = useState(null)
  const [voiceAudioUrl, setVoiceAudioUrl] = useState(null)
  const [cloningVoice, setCloningVoice] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [saving, setSaving] = useState(false)
  const [reportId, setReportId] = useState(null)
  const [refinementInput, setRefinementInput] = useState('')
  const [refining, setRefining] = useState(false)
  const [videoStatus, setVideoStatus] = useState(null)
  const [videoUrl, setVideoUrl] = useState(null)
  const [videoJobId, setVideoJobId] = useState(null)
  const [videoErrorMsg, setVideoErrorMsg] = useState('')
  const [attachments, setAttachments] = useState([])
  const [thinkingProcess, setThinkingProcess] = useState([])
  const [images, setImages] = useState([])
  const [reactions, setReactions] = useState([])
  const [generatingMedia, setGeneratingMedia] = useState(false)
  const fileInputRef = useRef(null)
  const thinkingRef = useRef(null)

  useEffect(() => {
    if (reportId && videoStatus === 'processing') {
      console.log('Starting video status polling for report:', reportId)
      const interval = setInterval(async () => {
        try {
          const status = await api.getReportStatus(reportId)
          console.log('Video status check:', status.video_status, 'URL:', status.video_url)
          setVideoStatus(status.video_status)
          if (status.video_url) {
            setVideoUrl(status.video_url)
          }
          if (status.video_status === 'completed' || status.video_status === 'failed') {
            console.log('Video generation finished with status:', status.video_status)
            clearInterval(interval)
          }
        } catch (error) {
          console.error('Failed to check video status:', error)
        }
      }, 2000) // Poll every 2 seconds instead of 3
      return () => {
        console.log('Clearing video polling interval')
        clearInterval(interval)
      }
    }
  }, [reportId, videoStatus])

  useEffect(() => {
    if (thinkingRef.current) {
      thinkingRef.current.scrollTop = thinkingRef.current.scrollHeight
    }
  }, [thinkingProcess])

  // Auto-save when script is generated
  useEffect(() => {
    if (script && !reportId && !saving) {
      const timer = setTimeout(() => {
        handleSaveReport()
      }, 1000)
      return () => clearTimeout(timer)
    }
  }, [script, reportId, saving])

  const addThinkingStep = (step) => {
    setThinkingProcess(prev => [...prev, { ...step, timestamp: new Date() }])
  }

  const handleGenerateReport = async () => {
    try {
      setGenerating(true)
      setThinkingProcess([])
      
      addThinkingStep({ type: 'info', message: 'Analyzing headline and context...' })
      await new Promise(resolve => setTimeout(resolve, 800))
      
      addThinkingStep({ type: 'info', message: 'Researching related information...' })
      await new Promise(resolve => setTimeout(resolve, 600))
      
      addThinkingStep({ type: 'info', message: 'Structuring news report format...' })
      await new Promise(resolve => setTimeout(resolve, 500))
      
      addThinkingStep({ type: 'info', message: 'Generating script with Gemini AI...' })
      
      const data = await api.generateNewsReport({
        headline_id: headline.id,
        title: headline.title,
        description: headline.description,
        url: headline.url,
        author_name: author || ''
      })
      
      setScript(data.report)
      addThinkingStep({ type: 'success', message: 'Script generated successfully!' })
      
      // Generate media (images and reactions)
      await generateMedia()
    } catch (error) {
      console.error('Failed to generate report:', error)
      addThinkingStep({ type: 'error', message: 'Failed to generate report. Please try again.' })
      alert('Failed to generate report. Please try again.')
    } finally {
      setGenerating(false)
    }
  }

  const generateMedia = async () => {
    try {
      setGeneratingMedia(true)
      addThinkingStep({ type: 'info', message: 'Generating story images with Imagen AI...' })
      
      // Call backend to generate real images with Imagen
      const response = await api.generateStoryMedia({
        title: headline.title,
        description: script
      })
      
      if (response.images && response.images.length > 0) {
        setImages(response.images)
        addThinkingStep({ type: 'success', message: `Generated ${response.images.length} images successfully!` })
      } else {
        // Fallback to placeholder if no images generated
        const mockImages = [
          {
            url: `https://picsum.photos/1024/768?random=${Math.random()}`,
            description: `Illustration for: ${headline.title}`,
            type: 'story',
            source: 'placeholder'
          }
        ]
        setImages(mockImages)
        addThinkingStep({ type: 'info', message: 'Using placeholder images' })
      }
      
      addThinkingStep({ type: 'info', message: 'Media generation complete' })
    } catch (error) {
      console.error('Failed to generate media:', error)
      addThinkingStep({ type: 'error', message: 'Failed to generate images. Using placeholders.' })
      
      // Fallback to placeholder images
      const mockImages = [
        {
          url: `https://picsum.photos/1024/768?random=${Math.random()}`,
          description: `Illustration for: ${headline.title}`,
          type: 'story',
          source: 'placeholder'
        }
      ]
      setImages(mockImages)
    } finally {
      setGeneratingMedia(false)
    }
  }

  const handleRefineScript = async () => {
    if (!refinementInput.trim() && attachments.length === 0) {
      return
    }
    try {
      setRefining(true)
      
      addThinkingStep({ type: 'info', message: 'Processing your refinement request...' })
      await new Promise(resolve => setTimeout(resolve, 500))
      
      if (attachments.length > 0) {
        addThinkingStep({ type: 'info', message: `Analyzing ${attachments.length} attachment(s)...` })
        await new Promise(resolve => setTimeout(resolve, 700))
      }
      
      addThinkingStep({ type: 'info', message: 'Refining script with Gemini AI...' })
      
      const data = await api.refineNewsReport(script, refinementInput, author)
      setScript(data.refined_script)
      
      addThinkingStep({ type: 'success', message: 'Script refined successfully!' })
      setRefinementInput('')
      setAttachments([])
    } catch (error) {
      console.error('Failed to refine script:', error)
      addThinkingStep({ type: 'error', message: 'Failed to refine script. Please try again.' })
      alert('Failed to refine script. Please try again.')
    } finally {
      setRefining(false)
    }
  }

  const handleAttachmentClick = () => {
    fileInputRef.current?.click()
  }

  const handleFileSelect = (e) => {
    const files = Array.from(e.target.files)
    const newAttachments = files.map(file => ({
      id: Math.random().toString(36).substr(2, 9),
      name: file.name,
      type: file.type,
      size: file.size,
      file: file
    }))
    setAttachments(prev => [...prev, ...newAttachments])
  }

  const removeAttachment = (id) => {
    setAttachments(prev => prev.filter(att => att.id !== id))
  }

  const handleAvatarChange = (e) => {
    const file = e.target.files[0]
    if (file) {
      if (file.size > 5 * 1024 * 1024) {
        alert('File size must be less than 5MB')
        return
      }
      setAvatarFile(file)
      const reader = new FileReader()
      reader.onloadend = () => setAvatarPreview(reader.result)
      reader.readAsDataURL(file)
    }
  }

  const handleSaveReport = async () => {
    if (!script.trim() || !author.trim()) {
      alert('Please generate a script first')
      return
    }
    try {
      setSaving(true)
      const data = await api.saveReport({
        headline_id: headline.id,
        title: headline.title,
        script: script,
        author: author
      })
      setReportId(data.report_id)
      // Notify parent to refresh stories list
      if (onStoryCreated) {
        onStoryCreated()
      }
    } catch (error) {
      console.error('Failed to save report:', error)
      alert('Failed to save report. Please try again.')
    } finally {
      setSaving(false)
    }
  }

  const handleDownloadImage = (image) => {
    // Handle data URLs (base64 images)
    if (image.url.startsWith('data:')) {
      const link = document.createElement('a')
      link.href = image.url
      link.download = `${image.type}-${Date.now()}.png`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
    } else {
      // For regular URLs, open in new tab
      window.open(image.url, '_blank')
    }
  }

  const handleEditImage = (image) => {
    // Placeholder for image editing functionality
    console.log('Edit image:', image)
    alert('Image editing feature coming soon')
  }

  const handleGenerateVideo = async () => {
    if (!reportId) {
      alert('Please save the report first')
      return
    }
    if (!avatarFile && !avatarPreview) {
      alert('Please upload your avatar')
      return
    }
    try {
      console.log('Starting video generation for report:', reportId)
      setVideoStatus('processing')
      setVideoErrorMsg('')
      
      const avatarURL = avatarPreview || 'https://placeholder.com/avatar.jpg'

      // If a voice file was uploaded but not yet cloned, clone it now (non-fatal)
      let audioUrl = voiceAudioUrl
      if (voiceFile && !voiceAudioUrl) {
        setCloningVoice(true)
        try {
          const formData = new FormData()
          formData.append('audio', voiceFile)
          formData.append('report_id', reportId)
          formData.append('script', script)
          formData.append('voice_name', author || 'reporter')
          const cloneResp = await api.cloneVoice(formData)
          audioUrl = cloneResp.audio_url
          setVoiceAudioUrl(audioUrl)
        } catch (err) {
          const errMsg = err?.response?.data?.error || err.message || ''
          if (errMsg.includes('paid_plan_required') || errMsg.includes('payment_required')) {
            console.warn('Voice cloning requires ElevenLabs paid plan - using default voice')
          } else {
            console.error('Voice cloning failed, proceeding with default voice:', err)
          }
        } finally {
          setCloningVoice(false)
        }
      }
      
      const data = await api.generateVideo({
        report_id: reportId,
        script: script,
        avatar_url: avatarURL,
        author: author,
        voice_audio_url: audioUrl || ''
      })
      setVideoJobId(data.video_job_id)
    } catch (error) {
      console.error('Failed to generate video:', error)
      setVideoStatus('failed')
      const serverError = error?.response?.data?.error || ''
      if (serverError.includes('not enough credits') || serverError.includes('InsufficientCredits')) {
        setVideoErrorMsg('D-ID credits exhausted. Top up at studio.d-id.com or add a new API key.')
      } else if (serverError.includes('ValidationError') || serverError.includes('valid image URL')) {
        setVideoErrorMsg('Avatar image format rejected by D-ID. Try uploading a JPG/PNG directly.')
      } else if (serverError) {
        setVideoErrorMsg(serverError)
      } else {
        setVideoErrorMsg('Video generation failed. Check backend logs for details.')
      }
    }
  }

  const formatFileSize = (bytes) => {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  }

  const getFileIcon = (type) => {
    if (type.startsWith('image/')) return <ImageIcon className="w-4 h-4" />
    return <FileText className="w-4 h-4" />
  }

  return (
    <div className="flex-1 flex flex-col h-full bg-white dark:bg-gray-900">
      <div className="border-b border-gray-200 dark:border-gray-800 px-4 sm:px-6 py-4">
        <button onClick={onBack} className="text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white mb-2">
          ← Back to Headlines
        </button>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Create News Report</h2>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 line-clamp-1">{headline.title}</p>
      </div>

      <div className="flex-1 overflow-y-auto p-4 sm:p-6">
        <div className="max-w-4xl mx-auto space-y-6">
          {/* Author Name */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              <User className="w-4 h-4 inline mr-1" />
              Correspondent Name
            </label>
            <input
              type="text"
              value={author}
              onChange={(e) => setAuthor(e.target.value)}
              placeholder={user?.name || "Enter your name"}
              className="w-full px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-white text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400"
            />
          </div>

          {/* Generate Button */}
          {!script && (
            <div className="text-center py-8 bg-gray-50 dark:bg-gray-800 rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-700">
              <Sparkles className="w-12 h-12 mx-auto mb-4 text-gray-900 dark:text-white" />
              <button
                onClick={handleGenerateReport}
                disabled={generating}
                className="inline-flex items-center gap-2 px-6 py-3 bg-brand-primary hover:bg-brand-primary-dark text-white rounded-lg disabled:opacity-50 font-medium shadow-md transition-all"
              >
                {generating ? (
                  <>
                    <Loader2 className="w-5 h-5 animate-spin" />
                    Generating with Gemini AI...
                  </>
                ) : (
                  <>
                    <Sparkles className="w-5 h-5" />
                    Generate Report
                  </>
                )}
              </button>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-3">
                AI will create a professional news report script from this headline
              </p>
            </div>
          )}

          {/* Script Editor */}
          {script && (
            <>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">News Report Script</label>
                <textarea
                  value={script}
                  onChange={(e) => setScript(e.target.value)}
                  rows="14"
                  className="w-full px-4 py-3 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-white font-mono text-sm leading-relaxed text-gray-900 dark:text-white"
                />
              </div>

              {/* Media Gallery */}
              {(images.length > 0 || reactions.length > 0 || generatingMedia) && (
                <div className="bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                  <MediaGallery 
                    images={images} 
                    reactions={reactions} 
                    loading={generatingMedia}
                    onDownloadImage={handleDownloadImage}
                    onEditImage={handleEditImage}
                  />
                </div>
              )}

              {/* Gemini Thinking Process */}
              {thinkingProcess.length > 0 && (
                <div className="bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                  <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
                    <Sparkles className="w-4 h-4 text-gray-900 dark:text-white" />
                    Gemini Thinking Process
                  </h3>
                  <div 
                    ref={thinkingRef}
                    className="space-y-2 max-h-48 overflow-y-auto"
                  >
                    {thinkingProcess.map((step, index) => (
                      <div 
                        key={index}
                        className={`flex items-start gap-2 text-sm p-2 rounded ${
                          step.type === 'error' ? 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400' :
                          step.type === 'success' ? 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400' :
                          'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400'
                        }`}
                      >
                        {step.type === 'error' ? (
                          <XCircle className="w-4 h-4 mt-0.5 flex-shrink-0" />
                        ) : step.type === 'success' ? (
                          <CheckCircle className="w-4 h-4 mt-0.5 flex-shrink-0" />
                        ) : (
                          generating || refining ? (
                            <Loader2 className="w-4 h-4 mt-0.5 flex-shrink-0 animate-spin" />
                          ) : (
                            <CheckCircle className="w-4 h-4 mt-0.5 flex-shrink-0" />
                          )
                        )}
                        <span>{step.message}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Refinement Input with Attachments */}
              <div className="space-y-3">
                {/* Attachments Display */}
                {attachments.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {attachments.map(att => (
                      <div 
                        key={att.id}
                        className="flex items-center gap-2 px-3 py-2 bg-gray-100 dark:bg-gray-800 rounded-lg text-sm"
                      >
                        {getFileIcon(att.type)}
                        <span className="text-gray-700 dark:text-gray-300">{att.name}</span>
                        <span className="text-gray-500 dark:text-gray-400 text-xs">({formatFileSize(att.size)})</span>
                        <button
                          onClick={() => removeAttachment(att.id)}
                          className="ml-1 text-gray-400 hover:text-red-600 dark:hover:text-red-400"
                        >
                          <XCircle className="w-4 h-4" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}

                {/* Input Field with Buttons */}
                <div className="flex items-center gap-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-full px-2 py-2 focus-within:ring-2 focus-within:ring-gray-900 dark:focus-within:ring-white focus-within:border-transparent">
                  {/* Attachment Button */}
                  <button
                    onClick={handleAttachmentClick}
                    className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-full transition-colors"
                    title="Attach files"
                  >
                    <Plus className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                  </button>
                  <input
                    ref={fileInputRef}
                    type="file"
                    multiple
                    accept="image/*,.pdf,.doc,.docx,.txt"
                    onChange={handleFileSelect}
                    className="hidden"
                  />

                  {/* Text Input */}
                  <input
                    type="text"
                    value={refinementInput}
                    onChange={(e) => setRefinementInput(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && !refining && handleRefineScript()}
                    placeholder="Ask Gemini to refine the script..."
                    className="flex-1 px-2 py-1 bg-transparent focus:outline-none text-sm text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400"
                    disabled={refining}
                  />

                  {/* Send Button */}
                  <button
                    onClick={handleRefineScript}
                    disabled={refining || (!refinementInput.trim() && attachments.length === 0)}
                    className="p-2 bg-gray-900 dark:bg-white hover:bg-gray-800 dark:hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed rounded-full transition-colors"
                    title="Send"
                  >
                    {refining ? (
                      <Loader2 className="w-5 h-5 text-white dark:text-gray-900 animate-spin" />
                    ) : (
                      <Send className="w-5 h-5 text-white dark:text-gray-900" />
                    )}
                  </button>
                </div>
              </div>

              {/* Avatar Upload */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  <Upload className="w-4 h-4 inline mr-1" />
                  Your Avatar/Photo
                </label>
                <div className="flex items-center gap-4">
                  {avatarPreview && (
                    <img src={getDisplayAvatarUrl(avatarPreview)} alt="Avatar" className="w-24 h-24 rounded-full object-cover border-2 border-gray-200 dark:border-gray-700 shadow-sm" />
                  )}
                  <label className="flex-1 cursor-pointer">
                    <div className="border-2 border-dashed border-gray-300 dark:border-gray-700 rounded-lg p-6 text-center hover:border-gray-900 dark:hover:border-white hover:bg-gray-50 dark:hover:bg-gray-800">
                      <Upload className="w-8 h-8 mx-auto mb-2 text-gray-400 dark:text-gray-500" />
                      <p className="text-sm text-gray-600 dark:text-gray-400 font-medium">
                        {avatarFile ? avatarFile.name : 'Click to upload your photo'}
                      </p>
                      <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">PNG, JPG up to 5MB</p>
                    </div>
                    <input type="file" accept="image/*" onChange={handleAvatarChange} className="hidden" />
                  </label>
                </div>
              </div>

              {/* Voice Clone Upload */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  <Mic className="w-4 h-4 inline mr-1" />
                  Clone Your Voice <span className="text-xs font-normal text-gray-500">(requires ElevenLabs Starter plan — $5/mo)</span>
                </label>
                <label className="cursor-pointer block">
                  <div className={`border-2 border-dashed rounded-lg p-4 text-center transition-colors ${voiceFile ? 'border-green-400 bg-green-50 dark:bg-green-900/20' : 'border-gray-300 dark:border-gray-700 hover:border-gray-900 dark:hover:border-white hover:bg-gray-50 dark:hover:bg-gray-800'}`}>
                    {cloningVoice ? (
                      <div className="flex items-center justify-center gap-2 text-sm text-blue-600">
                        <Loader2 className="w-4 h-4 animate-spin" />
                        Cloning voice...
                      </div>
                    ) : voiceAudioUrl ? (
                      <p className="text-sm text-green-700 dark:text-green-400 font-medium">✓ Voice cloned and ready</p>
                    ) : (
                      <>
                        <Mic className="w-6 h-6 mx-auto mb-1 text-gray-400" />
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          {voiceFile ? voiceFile.name : 'Upload a voice sample (MP3, WAV, M4A)'}
                        </p>
                        <p className="text-xs text-gray-500 mt-1">30s–3min of clear speech works best</p>
                      </>
                    )}
                  </div>
                  <input type="file" accept="audio/*" onChange={(e) => { setVoiceFile(e.target.files[0] || null); setVoiceAudioUrl(null) }} className="hidden" />
                </label>
              </div>

              {/* Action Buttons */}
              <div className="flex gap-3 pt-4">
                <button
                  onClick={handleSaveReport}
                  disabled={saving || !script.trim() || !author.trim()}
                  className="flex-1 flex items-center justify-center gap-2 px-4 py-3 bg-brand-dark hover:bg-brand-dark-light text-white rounded-lg disabled:opacity-50 font-medium shadow-md transition-all"
                >
                  {saving ? (
                    <>
                      <Loader2 className="w-5 h-5 animate-spin" />
                      Saving...
                    </>
                  ) : reportId ? (
                    <>
                      <CheckCircle className="w-5 h-5 text-green-400" />
                      Saved
                    </>
                  ) : (
                    <>
                      <Save className="w-5 h-5" />
                      Save Report
                    </>
                  )}
                </button>
                <button
                  onClick={handleGenerateVideo}
                  disabled={!reportId || (!avatarFile && !avatarPreview) || videoStatus === 'processing'}
                  className="flex-1 flex items-center justify-center gap-2 px-4 py-3 bg-brand-primary hover:bg-brand-primary-dark text-white rounded-lg disabled:opacity-50 font-medium shadow-md transition-all"
                >
                  {videoStatus === 'processing' ? (
                    <>
                      <Loader2 className="w-5 h-5 animate-spin" />
                      Generating Video...
                    </>
                  ) : (
                    <>
                      <Video className="w-5 h-5" />
                      Generate Video
                    </>
                  )}
                </button>
              </div>

              {!reportId && (
                <p className="text-xs text-center text-gray-500 dark:text-gray-400">Save your report before generating video</p>
              )}

              {/* Video Status */}
              {videoStatus && (
                <div className={`p-4 rounded-lg border-2 ${
                  videoStatus === 'processing' ? 'bg-blue-50 border-blue-200' :
                  videoStatus === 'completed' ? 'bg-green-50 border-green-200' :
                  'bg-red-50 border-red-200'
                }`}>
                  <div className="flex items-center gap-3 mb-3">
                    {videoStatus === 'processing' && (
                      <>
                        <Loader2 className="w-6 h-6 text-blue-600 animate-spin" />
                        <div>
                          <p className="font-medium text-blue-900">Generating Your Video</p>
                          <p className="text-sm text-blue-700">This may take a few minutes...</p>
                          {videoJobId && <p className="text-xs text-blue-600 mt-1">Job ID: {videoJobId}</p>}
                        </div>
                      </>
                    )}
                    {videoStatus === 'completed' && (
                      <>
                        <CheckCircle className="w-6 h-6 text-green-600" />
                        <div>
                          <p className="font-medium text-green-900">Video Ready!</p>
                          <p className="text-sm text-green-700">Your news report video has been generated</p>
                        </div>
                      </>
                    )}
                    {videoStatus === 'failed' && (
                      <>
                        <XCircle className="w-6 h-6 text-red-600" />
                        <div>
                          <p className="font-medium text-red-900">Generation Failed</p>
                          <p className="text-sm text-red-700">{videoErrorMsg || 'Please try again or contact support'}</p>
                        </div>
                      </>
                    )}
                  </div>

                  {videoStatus === 'completed' && videoUrl && (
                    <div className="mt-4">
                      <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-4 text-center border-2 border-dashed border-gray-300 dark:border-gray-700">
                        <p className="text-gray-700 dark:text-gray-300 font-medium mb-4">✓ Video Generated Successfully!</p>
                        <video 
                          controls 
                          className="w-full rounded-lg mb-4 bg-black"
                          style={{ maxHeight: '400px' }}
                        >
                          <source src={videoUrl} type="video/mp4" />
                          Your browser does not support the video tag.
                        </video>
                        <div className="flex gap-2 justify-center">
                          <a 
                            href={videoUrl} 
                            download 
                            className="px-4 py-2 bg-brand-primary text-white rounded-lg hover:bg-brand-primary-dark transition-colors text-sm font-medium"
                          >
                            Download Video
                          </a>
                          <a 
                            href={videoUrl} 
                            target="_blank" 
                            rel="noopener noreferrer"
                            className="px-4 py-2 bg-brand-dark text-white rounded-lg hover:bg-brand-dark-light transition-colors text-sm font-medium"
                          >
                            Open in New Tab
                          </a>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}
