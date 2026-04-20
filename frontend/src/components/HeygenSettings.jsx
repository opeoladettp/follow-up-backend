import { useState } from 'react'
import { Video, Mic, Save, CheckCircle, Loader2, ExternalLink } from 'lucide-react'
import { api } from '../services/api'

export default function HeygenSettings({ user, onSaved }) {
  const [avatarID, setAvatarID] = useState(user?.heygen_avatar_id || '')
  const [voiceID, setVoiceID] = useState(user?.heygen_voice_id || '')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState(null)

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      await api.updateHeygenSettings(avatarID.trim(), voiceID.trim())
      setSaved(true)
      if (onSaved) onSaved({ heygen_avatar_id: avatarID.trim(), heygen_voice_id: voiceID.trim() })
      setTimeout(() => setSaved(false), 3000)
    } catch (err) {
      setError(err?.response?.data?.error || 'Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <div className="flex items-center gap-2 mb-6">
        <Video className="w-5 h-5 text-gray-700 dark:text-gray-300" />
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">HeyGen Avatar &amp; Voice</h2>
      </div>

      <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">
        Set your personal HeyGen avatar and cloned voice IDs. These will be used every time you generate a video,
        overriding the system defaults.{' '}
        <a
          href="https://app.heygen.com/avatars"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-blue-600 hover:underline"
        >
          Open HeyGen Studio <ExternalLink className="w-3 h-3" />
        </a>
      </p>

      {error && (
        <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded-lg text-sm">
          {error}
        </div>
      )}

      <div className="space-y-5">
        {/* Avatar ID */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            <Video className="w-4 h-4 inline mr-1" />
            Avatar ID
          </label>
          <input
            type="text"
            value={avatarID}
            onChange={(e) => setAvatarID(e.target.value)}
            placeholder="e.g. Abigail_expressive_2024112501"
            className="w-full px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-white text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 font-mono text-sm"
          />
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Find this in HeyGen Studio → Avatars → your avatar → copy the ID from the URL or details panel.
          </p>
        </div>

        {/* Voice ID */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            <Mic className="w-4 h-4 inline mr-1" />
            Voice ID
          </label>
          <input
            type="text"
            value={voiceID}
            onChange={(e) => setVoiceID(e.target.value)}
            placeholder="e.g. 2d5b0e6cf36f460aa7fc47e3eee4ba54"
            className="w-full px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-white text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 font-mono text-sm"
          />
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Find this in HeyGen Studio → Voices → your cloned voice → copy the Voice ID.
          </p>
        </div>

        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-2 px-5 py-2.5 bg-gray-900 dark:bg-white hover:bg-gray-800 dark:hover:bg-gray-100 text-white dark:text-gray-900 rounded-lg disabled:opacity-50 font-medium transition-all"
        >
          {saving ? (
            <>
              <Loader2 className="w-4 h-4 animate-spin" />
              Saving...
            </>
          ) : saved ? (
            <>
              <CheckCircle className="w-4 h-4 text-green-400" />
              Saved!
            </>
          ) : (
            <>
              <Save className="w-4 h-4" />
              Save Settings
            </>
          )}
        </button>
      </div>
    </div>
  )
}
