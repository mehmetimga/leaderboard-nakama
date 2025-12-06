import { useState, useCallback, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Trophy, Medal, Crown, TrendingUp, Users, RefreshCw, ChevronUp, ChevronDown, Sparkles } from 'lucide-react'
import { useWebSocket, type LeaderboardResult, type LeaderboardRecord } from '../hooks/useWebSocket'
import './Leaderboard.css'

interface LeaderboardProps {
  leaderboardId: string
}

interface RankChange {
  change: number
  isNew: boolean
  timestamp: number
}

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'
const RANK_CHANGE_DISPLAY_MS = 3000

export function Leaderboard({ leaderboardId }: LeaderboardProps) {
  const [records, setRecords] = useState<LeaderboardRecord[]>([])
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null)
  const [rankChanges, setRankChanges] = useState<Map<string, RankChange>>(new Map())
  const previousRanksRef = useRef<Map<string, number>>(new Map())
  const isFirstLoadRef = useRef(true)

  const handleLeaderboardUpdate = useCallback((result: LeaderboardResult) => {
    const newRecords = result.records || []
    
    if (!isFirstLoadRef.current && previousRanksRef.current.size > 0) {
      const now = Date.now()
      const changes = new Map<string, RankChange>()
      
      newRecords.forEach((record) => {
        const prevRank = previousRanksRef.current.get(record.owner_id)
        if (prevRank === undefined) {
          changes.set(record.owner_id, { change: 0, isNew: true, timestamp: now })
        } else if (prevRank !== record.rank) {
          changes.set(record.owner_id, { change: prevRank - record.rank, isNew: false, timestamp: now })
        }
      })
      
      if (changes.size > 0) {
        setRankChanges(prev => {
          const merged = new Map(prev)
          changes.forEach((v, k) => merged.set(k, v))
          return merged
        })
      }
    } else {
      isFirstLoadRef.current = false
    }
    
    const newPrevRanks = new Map<string, number>()
    newRecords.forEach(r => newPrevRanks.set(r.owner_id, r.rank))
    previousRanksRef.current = newPrevRanks
    
    setRecords(newRecords)
    setLastUpdate(new Date())
  }, [])

  const { isConnected } = useWebSocket({
    url: WS_URL,
    leaderboardId,
    onLeaderboardUpdate: handleLeaderboardUpdate,
    onScoreUpdate: () => {},
  })

  // Clean up expired rank changes
  useEffect(() => {
    const interval = setInterval(() => {
      const now = Date.now()
      setRankChanges(prev => {
        if (prev.size === 0) return prev
        const updated = new Map(prev)
        let changed = false
        updated.forEach((v, k) => {
          if (now - v.timestamp > RANK_CHANGE_DISPLAY_MS) {
            updated.delete(k)
            changed = true
          }
        })
        return changed ? updated : prev
      })
    }, 500)
    return () => clearInterval(interval)
  }, [])

  // Fallback HTTP fetch
  useEffect(() => {
    if (!isConnected && records.length === 0) {
      fetch(`http://localhost:8080/api/v1/leaderboards/${leaderboardId}?type=live&limit=100`)
        .then(res => res.json())
        .then(data => {
          if (data.success && data.data) {
            const recs = data.data.records || []
            setRecords(recs)
            setLastUpdate(new Date())
            isFirstLoadRef.current = false
            const ranks = new Map<string, number>()
            recs.forEach((r: LeaderboardRecord) => ranks.set(r.owner_id, r.rank))
            previousRanksRef.current = ranks
          }
        })
        .catch(console.error)
    }
  }, [isConnected, leaderboardId, records.length])

  const getRankIcon = (rank: number) => {
    switch (rank) {
      case 1: return <Crown className="rank-icon gold" />
      case 2: return <Medal className="rank-icon silver" />
      case 3: return <Medal className="rank-icon bronze" />
      default: return <span className="rank-number">{rank}</span>
    }
  }

  const formatScore = (score: number) => score.toLocaleString()

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
              {records.map((record) => {
                const change = rankChanges.get(record.owner_id)
                const hasChange = !!change
                const movedUp = change?.change && change.change > 0
                const movedDown = change?.change && change.change < 0
                const isNew = change?.isNew
                
                return (
                  <motion.div
                    key={record.owner_id}
                    layout
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -20 }}
                    transition={{ duration: 0.3 }}
                    className={`record-item rank-${record.rank} ${hasChange ? 'has-rank-change' : ''} ${movedUp ? 'moved-up' : ''} ${movedDown ? 'moved-down' : ''} ${isNew ? 'is-new' : ''}`}
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
                      <div className="score-with-change">
                        <span className="score">{formatScore(record.score)}</span>
                        
                        {/* Rank change indicator */}
                        <AnimatePresence>
                          {isNew && (
                            <motion.div 
                              className="rank-change new-entry"
                              initial={{ scale: 0 }}
                              animate={{ scale: 1 }}
                              exit={{ scale: 0 }}
                            >
                              <Sparkles size={14} />
                              <span>NEW</span>
                            </motion.div>
                          )}
                          {movedUp && (
                            <motion.div 
                              className="rank-change rank-up"
                              initial={{ y: 20, opacity: 0 }}
                              animate={{ y: 0, opacity: 1 }}
                              exit={{ y: -20, opacity: 0 }}
                            >
                              <ChevronUp size={18} strokeWidth={3} />
                              <span>+{change!.change}</span>
                            </motion.div>
                          )}
                          {movedDown && (
                            <motion.div 
                              className="rank-change rank-down"
                              initial={{ y: -20, opacity: 0 }}
                              animate={{ y: 0, opacity: 1 }}
                              exit={{ y: 20, opacity: 0 }}
                            >
                              <ChevronDown size={18} strokeWidth={3} />
                              <span>{change!.change}</span>
                            </motion.div>
                          )}
                        </AnimatePresence>
                      </div>
                    </div>

                    {record.rank <= 3 && <div className="glow-effect" />}
                    
                    {hasChange && (
                      <motion.div 
                        className={`rank-change-bg ${movedUp ? 'bg-up' : ''} ${movedDown ? 'bg-down' : ''} ${isNew ? 'bg-new' : ''}`}
                        initial={{ opacity: 0.6, scaleX: 0 }}
                        animate={{ opacity: 0.2, scaleX: 1 }}
                        transition={{ duration: 0.4 }}
                      />
                    )}
                  </motion.div>
                )
              })}
            </AnimatePresence>
          </div>
        )}
      </div>
    </div>
  )
}
