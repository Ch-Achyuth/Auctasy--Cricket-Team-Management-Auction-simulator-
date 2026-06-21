import { useState, useEffect } from 'react'

/**
 * Toast — lightweight notification component.
 * Auto-dismisses after 3 seconds.
 * Usage: <Toast message="Saved!" type="success" onClose={() => {}} />
 */
export default function Toast({ message, type = 'success', onClose }) {
  const [visible, setVisible] = useState(true)

  useEffect(() => {
    const timer = setTimeout(() => {
      setVisible(false)
      setTimeout(onClose, 300) // wait for fade-out animation
    }, 3000)

    return () => clearTimeout(timer)
  }, [onClose])

  return (
    <div className={`toast toast-${type} ${visible ? 'toast-enter' : 'toast-exit'}`}>
      <span className="toast-icon">
        {type === 'success' ? '✓' : type === 'error' ? '✕' : 'ℹ'}
      </span>
      <span>{message}</span>
    </div>
  )
}

/**
 * useToast — hook to manage toast notifications.
 * Returns [toasts, showToast] where showToast('message', 'success'|'error')
 */
export function useToast() {
  const [toasts, setToasts] = useState([])

  const showToast = (message, type = 'success') => {
    const id = Date.now()
    setToasts((prev) => [...prev, { id, message, type }])
  }

  const removeToast = (id) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }

  const ToastContainer = () => (
    <div className="toast-container">
      {toasts.map((t) => (
        <Toast
          key={t.id}
          message={t.message}
          type={t.type}
          onClose={() => removeToast(t.id)}
        />
      ))}
    </div>
  )

  return { showToast, ToastContainer }
}
