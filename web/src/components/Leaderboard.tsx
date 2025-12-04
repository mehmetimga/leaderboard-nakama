import { useState, useCallback, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Trophy, Medal, Crown, TrendingUp, Users, RefreshCw } from 'lucide-react'
import { useWebSocket, type LeaderboardResult, type LeaderboardRecord } from '../hooks/useWebSocket'
import './Leaderboard.css'

interface LeaderboardProps {
  leaderboardId: string
}

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'

export function Leaderboard({ leaderboardId }: LeaderboardProps) {
  const [records, setRecords] = useState<LeaderboardRecord[]>([])
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null)
  const [highlightedUser, setHighlightedUser] = useState<string | null>(null)

  const handleLeaderboardUpdate = useCallback((result: LeaderboardResult) => {
    setRecords(result.records || [])
    setLastUpdate(new Date())
  }, [])

  const handleScoreUpdate = useCallback((record: LeaderboardRecord) => {
    setHighlightedUser(record.owner_id)
    setTimeout(() => setHighlightedUser(null), 2000)
  }, [])

  const { isConnected } = useWebSocket({
    url: WS_URL,
    leaderboardId,
    onLeaderboardUpdate: handleLeaderboardUpdate,
    onScoreUpdate: handleScoreUpdate,
  })

  // Fallback: fetch via HTTP if WebSocket fails
  useEffect(() => {
    if (!isConnected && records.length === 0) {
      fetch(`http://localhost:8080/api/v1/leaderboards/${leaderboardId}?type=live&limit=100`)
        .then(res => res.json())
        .then(data => {
          if (data.success && data.data) {
            setRecords(data.data.records || [])
            setLastUpdate(new Date())
          }
        })
        .catch(console.error)
    }
  }, [isConnected, leaderboardId, records.length])

  const getRankIcon = (rank: number) => {
    switch (rank) {
      case 1:
        return <Crown className="rank-icon gold" />
      case 2:
        return <Medal className="rank-icon silver" />
      case 3:
        return <Medal className="rank-icon bronze" />
      default:
        return <span className="rank-number">{rank}</span>
    }
  }

  const formatScore = (score: number) => {
    return score.toLocaleString()
  }

  return (
    <div className="leaderboard">
      <div className="leaderboard-header">
        <div className="header-left">
          <Trophy className="header-icon" />
          <div>
            <h2>Global Rankings</h2>
            <p className="leaderboard-id">{leaderboardId}</p>
          </div>
        </div>
        <div className="header-right">
          <div className="stat">
            <Users size={16} />
            <span>{records.length} players</span>
          </div>
          {lastUpdate && (
            <div className="stat">
              <RefreshCw size={14} className={isConnected ? 'connected' : ''} />
              <span>{lastUpdate.toLocaleTimeString()}</span>
            </div>
          )}
        </div>
      </div>

      <div className="leaderboard-content">
        {records.length === 0 ? (
          <div className="empty-state">
            <TrendingUp size={48} />
            <p>No scores yet</p>
            <span>Be the first to submit a score!</span>
          </div>
        ) : (
          <div className="records-list">
            <AnimatePresence mode="popLayout">
              {records.map((record, index) => (
                <motion.div
                  key={record.owner_id}
                  layout
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ 
                    opacity: 1, 
                    x: 0,
                    scale: highlightedUser === record.owner_id ? 1.02 : 1,
                  }}
                  exit={{ opacity: 0, x: 20 }}
                  transition={{ 
                    duration: 0.3,
                    delay: index * 0.02,
                    layout: { duration: 0.3 }
                  }}
                  className={`record-item rank-${record.rank} ${
                    highlightedUser === record.owner_id ? 'highlighted' : ''
                  }`}
                >
                  <div className="rank-badge">
                    {getRankIcon(record.rank)}
                  </div>

                  <div className="player-info">
                    <div className="avatar">
                      {(record.username || record.owner_id).charAt(0).toUpperCase()}
                    </div>
                    <div className="player-details">
                      <span className="player-name">
                        {record.username || `Player ${record.owner_id.slice(0, 8)}`}
                      </span>
                      <span className="player-id">{record.owner_id.slice(0, 12)}...</span>
                    </div>
                  </div>

                  <div className="score-section">
                    <motion.span 
                      className="score"
                      key={record.score}
                      initial={{ scale: 1.2, color: '#00f5d4' }}
                      animate={{ scale: 1, color: '#ffffff' }}
                      transition={{ duration: 0.3 }}
                    >
                      {formatScore(record.score)}
                    </motion.span>
                    {record.subscore > 0 && (
                      <span className="subscore">+{record.subscore}</span>
                    )}
                  </div>

                  {record.rank <= 3 && (
                    <div className="glow-effect" />
                  )}
                </motion.div>
              ))}
            </AnimatePresence>
          </div>
        )}
      </div>
    </div>
  )
}

