import { useState, useEffect, useRef } from 'react'

/**
 * WaitingListCounter — React version of the localStorage-based counter.
 * Starts at 46, persists via localStorage, allows one click per page load.
 */
export default function WaitingListCounter() {
  const [count, setCount] = useState(() => {
    const stored = localStorage.getItem('waitingListCount')
    return stored ? parseInt(stored, 10) : 46
  })
  const [hasClicked, setHasClicked] = useState(false)

  const handleClick = () => {
    if (hasClicked) return
    const newCount = count + 1
    setCount(newCount)
    localStorage.setItem('waitingListCount', newCount)
    setHasClicked(true)
  }

  return (
    <button
      type="button"
      className="premium-button"
      onClick={handleClick}
      style={hasClicked ? { opacity: 0.7, cursor: 'not-allowed' } : {}}
    >
      Waiting list: {count}{hasClicked ? ' (Joined!)' : ''}
    </button>
  )
}
