import { GoogleOAuthProvider, GoogleLogin } from '@react-oauth/google'
import { jwtDecode } from 'jwt-decode'
import { Newspaper } from 'lucide-react'

export default function GoogleAuth({ onLoginSuccess }) {
  const handleSuccess = async (credentialResponse) => {
    try {
      const decoded = jwtDecode(credentialResponse.credential)
      
      // Send to backend for authentication
      const response = await fetch('http://localhost:8080/api/v1/auth/google', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          google_id: decoded.sub,
          email: decoded.email,
          name: decoded.name,
          picture: decoded.picture,
        }),
      })

      if (!response.ok) {
        throw new Error('Authentication failed')
      }

      const data = await response.json()
      
      // Store user data in localStorage
      localStorage.setItem('user', JSON.stringify(data.user))
      localStorage.setItem('userId', data.user.id)
      
      onLoginSuccess(data.user)
    } catch (error) {
      console.error('Login error:', error)
      alert('Login failed. Please try again.')
    }
  }

  const handleError = () => {
    console.error('Login Failed')
    alert('Login failed. Please try again.')
  }

  return (
    <GoogleOAuthProvider clientId={import.meta.env.VITE_GOOGLE_CLIENT_ID || 'YOUR_GOOGLE_CLIENT_ID'}>
      <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 flex items-center justify-center p-4">
        <div className="max-w-md w-full">
          <div className="bg-white rounded-2xl shadow-xl p-8 space-y-6">
            {/* Logo and Title */}
            <div className="text-center space-y-3">
              <div className="flex justify-center">
                <div className="p-4 bg-blue-100 rounded-full">
                  <Newspaper className="w-12 h-12 text-blue-600" />
                </div>
              </div>
              <h1 className="text-3xl font-bold text-gray-900">FollowUpMedium</h1>
              <p className="text-gray-600">AI-Powered News Tracking & Reporting</p>
            </div>

            {/* Features */}
            <div className="space-y-3 py-4">
              <div className="flex items-start gap-3">
                <div className="w-2 h-2 bg-blue-600 rounded-full mt-2"></div>
                <p className="text-sm text-gray-700">Track breaking news stories in real-time</p>
              </div>
              <div className="flex items-start gap-3">
                <div className="w-2 h-2 bg-blue-600 rounded-full mt-2"></div>
                <p className="text-sm text-gray-700">Generate AI-powered video scripts</p>
              </div>
              <div className="flex items-start gap-3">
                <div className="w-2 h-2 bg-blue-600 rounded-full mt-2"></div>
                <p className="text-sm text-gray-700">Collaborate with your newsroom team</p>
              </div>
            </div>

            {/* Google Login Button */}
            <div className="flex justify-center pt-4">
              <GoogleLogin
                onSuccess={handleSuccess}
                onError={handleError}
                useOneTap
                theme="outline"
                size="large"
                text="continue_with"
                shape="rectangular"
              />
            </div>

            {/* User Types Info */}
            <div className="pt-4 border-t border-gray-200">
              <p className="text-xs text-gray-500 text-center mb-3">User Roles:</p>
              <div className="grid grid-cols-3 gap-2 text-center">
                <div className="p-2 bg-gray-50 rounded-lg">
                  <p className="text-xs font-medium text-gray-900">Correspondent</p>
                  <p className="text-xs text-gray-500 mt-1">Create stories</p>
                </div>
                <div className="p-2 bg-gray-50 rounded-lg">
                  <p className="text-xs font-medium text-gray-900">Editor</p>
                  <p className="text-xs text-gray-500 mt-1">Manage feeds</p>
                </div>
                <div className="p-2 bg-gray-50 rounded-lg">
                  <p className="text-xs font-medium text-gray-900">Admin</p>
                  <p className="text-xs text-gray-500 mt-1">Full access</p>
                </div>
              </div>
            </div>
          </div>

          {/* Footer */}
          <p className="text-center text-xs text-gray-500 mt-6">
            By signing in, you agree to our Terms of Service and Privacy Policy
          </p>
        </div>
      </div>
    </GoogleOAuthProvider>
  )
}
