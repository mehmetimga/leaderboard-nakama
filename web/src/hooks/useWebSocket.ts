import { useEffect, useRef, useState, useCallback } from 'react'

export interface LeaderboardRecord {
  leaderboard_id: string
  owner_id: string
  username: string
  score: number
  subscore: number
  rank: number
  metadata?: Record<string, string>
  create_time: string
  update_time: string
}

export interface LeaderboardResult {
  leaderboard_id: string
  type: string
  records: LeaderboardRecord[]
  owner_records?: LeaderboardRecord[]
  next_cursor?: string
  prev_cursor?: string
  total_count?: number
  snapshot_time?: string
}

export interface WSMessage {
  type: 'subscribe' | 'unsubscribe' | 'leaderboard' | 'score_update' | 'error' | 'ping' | 'pong'
  leaderboard_id?: string
  data?: LeaderboardResult | LeaderboardRecord
  error?: string
  timestamp: number
}

interface UseWebSocketOptions {
  url: string
  leaderboardId: string
  onLeaderboardUpdate?: (result: LeaderboardResult) => void
  onScoreUpdate?: (record: LeaderboardRecord) => void
  reconnectInterval?: number
}

export function useWebSocket({
  url,
  leaderboardId,
  onLeaderboardUpdate,
  onScoreUpdate,
  reconnectInterval = 3000,
}: UseWebSocketOptions) {
  const ws = useRef<WebSocket | null>(null)
  const reconnectTimeout = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const pingInterval = useRef<ReturnType<typeof setInterval> | undefined>(undefined)
  
  const [isConnected, setIsConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const connect = useCallback(() => {
    if (ws.current?.readyState === WebSocket.OPEN) return

    try {
      ws.current = new WebSocket(url)

      ws.current.onopen = () => {
        console.log('[WS] Connected')
        setIsConnected(true)
        setError(null)

        // Subscribe to leaderboard
        ws.current?.send(JSON.stringify({
          type: 'subscribe',
          leaderboard_id: leaderboardId,
          timestamp: Date.now(),
        }))

        // Start ping interval
        pingInterval.current = setInterval(() => {
          if (ws.current?.readyState === WebSocket.OPEN) {
            ws.current.send(JSON.stringify({ type: 'ping', timestamp: Date.now() }))
          }
        }, 25000)
      }

      ws.current.onmessage = (event) => {
        try {
          const messages = event.data.split('\n')
          for (const msgStr of messages) {
            if (!msgStr.trim()) continue
            const msg: WSMessage = JSON.parse(msgStr)

            switch (msg.type) {
              case 'leaderboard':
                if (msg.data && onLeaderboardUpdate) {
                  onLeaderboardUpdate(msg.data as LeaderboardResult)
                }
                break
              case 'score_update':
                if (msg.data && onScoreUpdate) {
                  onScoreUpdate(msg.data as LeaderboardRecord)
                }
                break
              case 'error':
                console.error('[WS] Error:', msg.error)
                setError(msg.error || 'Unknown error')
                break
              case 'pong':
                // Heartbeat received
                break
            }
          }
        } catch (e) {
          console.error('[WS] Parse error:', e)
        }
      }

      ws.current.onclose = () => {
        console.log('[WS] Disconnected')
        setIsConnected(false)
        
        if (pingInterval.current) {
          clearInterval(pingInterval.current)
        }

        // Reconnect after delay
        reconnectTimeout.current = setTimeout(connect, reconnectInterval)
      }

      ws.current.onerror = (e) => {
        console.error('[WS] Error:', e)
        setError('Connection error')
      }
    } catch (e) {
      console.error('[WS] Connect error:', e)
      setError('Failed to connect')
      reconnectTimeout.current = setTimeout(connect, reconnectInterval)
    }
  }, [url, leaderboardId, onLeaderboardUpdate, onScoreUpdate, reconnectInterval])

  const disconnect = useCallback(() => {
    if (reconnectTimeout.current) {
      clearTimeout(reconnectTimeout.current)
    }
    if (pingInterval.current) {
      clearInterval(pingInterval.current)
    }
    if (ws.current) {
      ws.current.close()
      ws.current = null
    }
  }, [])

  const subscribe = useCallback((newLeaderboardId: string) => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify({
        type: 'subscribe',
        leaderboard_id: newLeaderboardId,
        timestamp: Date.now(),
      }))
    }
  }, [])

  const unsubscribe = useCallback((oldLeaderboardId: string) => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify({
        type: 'unsubscribe',
        leaderboard_id: oldLeaderboardId,
        timestamp: Date.now(),
      }))
    }
  }, [])

  useEffect(() => {
    connect()
    return disconnect
  }, [connect, disconnect])

  return {
    isConnected,
    error,
    subscribe,
    unsubscribe,
    disconnect,
  }
}

