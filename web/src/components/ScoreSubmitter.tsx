import { useState } from 'react'
import { motion } from 'framer-motion'
import { Send, User, Hash, Zap, Loader2, CheckCircle, AlertCircle } from 'lucide-react'
import './ScoreSubmitter.css'

interface ScoreSubmitterProps {
  leaderboardId: string
}

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export function ScoreSubmitter({ leaderboardId }: ScoreSubmitterProps) {
  const [userId, setUserId] = useState('')
  const [username, setUsername] = useState('')
  const [score, setScore] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [status, setStatus] = useState<'idle' | 'success' | 'error'>('idle')
  const [message, setMessage] = useState('')

  const generateRandomUser = () => {
    const id = `user-${Math.random().toString(36).substring(2, 10)}`
    const names = ['Phantom', 'Shadow', 'Nova', 'Blaze', 'Storm', 'Thunder', 'Viper', 'Ghost', 'Raven', 'Wolf']
    const adjectives = ['Dark', 'Swift', 'Silent', 'Fierce', 'Mystic', 'Cyber', 'Neon', 'Pixel', 'Turbo', 'Ultra']
    const name = `${adjectives[Math.floor(Math.random() * adjectives.length)]}${names[Math.floor(Math.random() * names.length)]}`
    setUserId(id)
    setUsername(name)
    setScore(String(Math.floor(Math.random() * 10000) + 1000))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!userId || !score) {
      setStatus('error')
      setMessage('User ID and Score are required')
      return
    }

    setIsSubmitting(true)
    setStatus('idle')

    try {
      const response = await fetch(`${API_URL}/api/v1/scores`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          leaderboard_id: leaderboardId,
          user_id: userId,
          username: username || undefined,
          score: parseInt(score, 10),
        }),
      })

      const data = await response.json()

      if (data.success) {
        setStatus('success')
        setMessage(`Score submitted! Rank: ${data.data?.rank || 'N/A'}`)
        // Reset form after success
        setTimeout(() => {
          setStatus('idle')
          setMessage('')
        }, 3000)
      } else {
        setStatus('error')
        setMessage(data.error || 'Failed to submit score')
      }
    } catch (err) {
      setStatus('error')
      setMessage('Network error. Is the API running?')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="score-submitter">
      <div className="submitter-header">
        <Zap className="header-icon" />
        <h3>Submit Score</h3>
      </div>

      <form onSubmit={handleSubmit} className="submit-form">
        <div className="form-group">
          <label>
            <User size={16} />
            User ID
          </label>
          <input
            type="text"
            value={userId}
            onChange={(e) => setUserId(e.target.value)}
            placeholder="Enter user ID"
            disabled={isSubmitting}
          />
        </div>

        <div className="form-group">
          <label>
            <User size={16} />
            Username (optional)
          </label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Display name"
            disabled={isSubmitting}
          />
        </div>

        <div className="form-group">
          <label>
            <Hash size={16} />
            Score
          </label>
          <input
            type="number"
            value={score}
            onChange={(e) => setScore(e.target.value)}
            placeholder="Enter score"
            min="0"
            disabled={isSubmitting}
          />
        </div>

        <button
          type="button"
          className="random-btn"
          onClick={generateRandomUser}
          disabled={isSubmitting}
        >
          🎲 Generate Random
        </button>

        <motion.button
          type="submit"
          className="submit-btn"
          disabled={isSubmitting}
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
        >
          {isSubmitting ? (
            <>
              <Loader2 className="spinning" size={18} />
              Submitting...
            </>
          ) : (
            <>
              <Send size={18} />
              Submit Score
            </>
          )}
        </motion.button>

        {status !== 'idle' && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            className={`status-message ${status}`}
          >
            {status === 'success' ? (
              <CheckCircle size={18} />
            ) : (
              <AlertCircle size={18} />
            )}
            {message}
          </motion.div>
        )}
      </form>

      <div className="submitter-footer">
        <p>Scores are sent via WebSocket and update in real-time</p>
      </div>
    </div>
  )
}

