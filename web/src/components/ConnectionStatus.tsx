import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { Wifi, WifiOff, Activity } from 'lucide-react'
import './ConnectionStatus.css'

export function ConnectionStatus() {
  const [wsConnected, setWsConnected] = useState(false)
  const [apiHealthy, setApiHealthy] = useState(false)

  useEffect(() => {
    // Check API health
    const checkHealth = async () => {
      try {
        const response = await fetch('http://localhost:8080/health')
        const data = await response.json()
        setApiHealthy(data.success)
        if (data.data?.ws_clients !== undefined) {
          setWsConnected(true)
        }
      } catch {
        setApiHealthy(false)
      }
    }

    checkHealth()
    const interval = setInterval(checkHealth, 5000)
    return () => clearInterval(interval)
  }, [])

  // Listen for WebSocket connection events from the hook
  useEffect(() => {
    const handleWsConnect = () => setWsConnected(true)
    const handleWsDisconnect = () => setWsConnected(false)

    window.addEventListener('ws-connected', handleWsConnect)
    window.addEventListener('ws-disconnected', handleWsDisconnect)

    return () => {
      window.removeEventListener('ws-connected', handleWsConnect)
      window.removeEventListener('ws-disconnected', handleWsDisconnect)
    }
  }, [])

  return (
    <div className="connection-status">
      <motion.div
        className={`status-indicator ${apiHealthy ? 'connected' : 'disconnected'}`}
        initial={{ opacity: 0, scale: 0.8 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.3 }}
      >
        <Activity size={14} />
        <span>API</span>
        <div className={`status-dot ${apiHealthy ? 'active' : ''}`} />
      </motion.div>

      <motion.div
        className={`status-indicator ${wsConnected ? 'connected' : 'disconnected'}`}
        initial={{ opacity: 0, scale: 0.8 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.3, delay: 0.1 }}
      >
        {wsConnected ? <Wifi size={14} /> : <WifiOff size={14} />}
        <span>WebSocket</span>
        <div className={`status-dot ${wsConnected ? 'active' : ''}`} />
      </motion.div>
    </div>
  )
}

