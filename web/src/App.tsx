import { useState, useEffect } from 'react';

const API_BASE = window.location.origin;

interface IOSModalProps {
  isOpen: boolean;
  onClose: () => void;
}

function IOSModal({ isOpen, onClose }: IOSModalProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg p-6 max-w-md">
        <h2 className="text-xl font-bold mb-4 text-gray-900">iOS Setup Required</h2>
        <p className="text-gray-700 mb-4">
          To receive notifications on iOS:
        </p>
        <ol className="list-decimal list-inside space-y-2 text-gray-700 mb-6">
          <li>Tap the Share button (square with arrow)</li>
          <li>Select "Add to Home Screen"</li>
          <li>Open Pushem from your home screen</li>
          <li>Subscribe to your topic</li>
        </ol>
        <button
          onClick={onClose}
          className="w-full bg-black text-white py-2 px-4 rounded hover:bg-gray-800"
        >
          Got it
        </button>
      </div>
    </div>
  );
}

interface Message {
  ID: number;
  Topic: string;
  Title: string;
  Message: string;
  CreatedAt: string;
}

interface HistoryModalProps {
  isOpen: boolean;
  onClose: () => void;
  topic: string;
}

function HistoryModal({ isOpen, onClose, topic }: HistoryModalProps) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isOpen && topic) {
      fetchHistory();
    }
  }, [isOpen, topic]);

  const fetchHistory = async () => {
    setLoading(true);
    try {
      const response = await fetch(`${API_BASE}/history/${encodeURIComponent(topic)}`);
      if (response.ok) {
        const data = await response.json();
        setMessages(data || []);
      }
    } catch (error) {
      console.error('Failed to fetch history:', error);
    } finally {
      setLoading(false);
    }
  };

  const clearHistory = async () => {
    if (!confirm('Are you sure you want to clear all history for this topic?')) return;

    try {
      const response = await fetch(`${API_BASE}/history/${encodeURIComponent(topic)}`, {
        method: 'DELETE',
      });
      if (response.ok) {
        setMessages([]);
      }
    } catch (error) {
      console.error('Failed to clear history:', error);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg p-6 max-w-lg w-full max-h-[80vh] flex flex-col">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-bold text-gray-900">History: {topic}</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700">
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="flex-1 overflow-y-auto mb-4 space-y-3">
          {loading ? (
            <div className="text-center py-4 text-gray-500">Loading...</div>
          ) : messages.length === 0 ? (
            <div className="text-center py-4 text-gray-500">No messages found</div>
          ) : (
            messages.map((msg) => (
              <div key={msg.ID} className="bg-gray-50 p-3 rounded border border-gray-200">
                <div className="flex justify-between items-start mb-1">
                  <h3 className="font-medium text-gray-900">{msg.Title || 'Notification'}</h3>
                  <span className="text-xs text-gray-500 whitespace-nowrap ml-2">
                    {new Date(msg.CreatedAt).toLocaleString()}
                  </span>
                </div>
                <p className="text-sm text-gray-600 break-words">{msg.Message}</p>
              </div>
            ))
          )}
        </div>

        <div className="flex gap-3 mt-auto pt-4 border-t border-gray-200">
          <button
            onClick={fetchHistory}
            className="flex-1 py-2 px-4 bg-gray-100 text-gray-700 rounded hover:bg-gray-200 font-medium"
          >
            Refresh
          </button>
          <button
            onClick={clearHistory}
            disabled={messages.length === 0}
            className={`flex-1 py-2 px-4 text-white rounded font-medium ${messages.length === 0
              ? 'bg-red-300 cursor-not-allowed'
              : 'bg-red-600 hover:bg-red-700'
              }`}
          >
            Clear History
          </button>
        </div>
      </div>
    </div>
  );
}

function App() {
  const [topic, setTopic] = useState('');
  const [status, setStatus] = useState('');
  const [isSubscribed, setIsSubscribed] = useState(false);
  const [showIOSModal, setShowIOSModal] = useState(false);
  const [activeHistoryTopic, setActiveHistoryTopic] = useState<string | null>(null);
  const [subscribedTopics, setSubscribedTopics] = useState<string[]>([]);

  useEffect(() => {
    registerServiceWorker();
    loadSubscribedTopics();
  }, []);

  const registerServiceWorker = async () => {
    if ('serviceWorker' in navigator) {
      try {
        const registration = await navigator.serviceWorker.register('/sw.js');
        console.log('Service Worker registered:', registration);
      } catch (error) {
        console.error('Service Worker registration failed:', error);
        setStatus('Failed to register service worker');
      }
    }
  };

  const loadSubscribedTopics = () => {
    const topics = JSON.parse(localStorage.getItem('subscribedTopics') || '[]');
    setSubscribedTopics(topics);
  };

  const saveSubscribedTopic = (topic: string) => {
    const topics = JSON.parse(localStorage.getItem('subscribedTopics') || '[]');
    if (!topics.includes(topic)) {
      topics.push(topic);
      localStorage.setItem('subscribedTopics', JSON.stringify(topics));
      setSubscribedTopics(topics);
    }
  };

  const removeSubscribedTopic = (topic: string) => {
    const topics = JSON.parse(localStorage.getItem('subscribedTopics') || '[]');
    const filtered = topics.filter((t: string) => t !== topic);
    localStorage.setItem('subscribedTopics', JSON.stringify(filtered));
    setSubscribedTopics(filtered);
  };

  const handleTestNotification = async (topic: string) => {
    try {
      const response = await fetch(`${API_BASE}/publish/${encodeURIComponent(topic)}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          title: 'Test Notification',
          message: `This is a test notification for topic "${topic}"`,
        }),
      });

      if (response.ok) {
        const result = await response.json();
        if (result.sent > 0) {
          setStatus(`Test notification sent to "${topic}"!`);
        } else {
          setStatus(`No active subscription found for "${topic}"`);
        }
      } else {
        setStatus('Failed to send test notification');
      }

      setTimeout(() => setStatus(''), 3000);
    } catch (error) {
      setStatus(`Error: ${error instanceof Error ? error.message : String(error)}`);
    }
  };

  const handleUnsubscribe = async (topic: string) => {
    try {
      const registration = await navigator.serviceWorker.ready;
      const subscription = await registration.pushManager.getSubscription();

      if (subscription) {
        await subscription.unsubscribe();
        removeSubscribedTopic(topic);
        setStatus(`Unsubscribed from "${topic}"`);
        setTimeout(() => setStatus(''), 3000);
      }
    } catch (error) {
      setStatus(`Error unsubscribing: ${error instanceof Error ? error.message : String(error)}`);
    }
  };

  const isIOS = () => {
    return /iPad|iPhone|iPod/.test(navigator.userAgent) && !(window as any).MSStream;
  };

  const isStandalone = () => {
    return (window.navigator as any).standalone === true;
  };

  const handleSubscribe = async () => {
    if (!topic.trim()) {
      setStatus('Please enter a topic name');
      return;
    }

    if (isIOS() && !isStandalone()) {
      setShowIOSModal(true);
      return;
    }

    if (!('Notification' in window)) {
      setStatus('This browser does not support notifications');
      return;
    }

    if (!('serviceWorker' in navigator)) {
      setStatus('This browser does not support service workers');
      return;
    }

    try {
      setStatus('Requesting permission...');

      const permission = await Notification.requestPermission();
      if (permission !== 'granted') {
        setStatus('Notification permission denied');
        return;
      }

      setStatus('Getting VAPID public key...');
      const vapidResponse = await fetch(`${API_BASE}/vapid-public-key`);
      const { publicKey } = await vapidResponse.json();

      setStatus('Checking for existing subscriptions...');
      const registration = await navigator.serviceWorker.ready;

      // Unsubscribe from any existing subscription first
      const existingSubscription = await registration.pushManager.getSubscription();
      if (existingSubscription) {
        setStatus('Removing old subscription...');
        await existingSubscription.unsubscribe();
      }

      setStatus('Subscribing to push notifications...');
      const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(publicKey),
      });

      setStatus('Registering subscription with server...');
      const response = await fetch(`${API_BASE}/subscribe/${encodeURIComponent(topic)}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(subscription.toJSON()),
      });

      if (!response.ok) {
        throw new Error(`Server returned ${response.status}`);
      }

      setStatus(`Successfully subscribed to "${topic}"!`);
      setIsSubscribed(true);
      saveSubscribedTopic(topic);

      setTimeout(() => {
        setStatus('');
      }, 3000);
    } catch (error) {
      console.error('Subscription failed:', error);
      setStatus(`Error: ${error instanceof Error ? error.message : String(error)}`);
    }
  };

  const urlBase64ToUint8Array = (base64String: string) => {
    const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');

    const rawData = window.atob(base64);
    const outputArray = new Uint8Array(rawData.length);

    for (let i = 0; i < rawData.length; ++i) {
      outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-900 to-gray-800 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-xl p-8 max-w-md w-full">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">Pushem</h1>
          <p className="text-gray-600 mb-4">Self-hosted notifications</p>
          {subscribedTopics.length > 0 && (
            <div className="inline-flex items-center gap-2 text-sm text-green-700 bg-green-50 px-3 py-1 rounded-full">
              <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
              {subscribedTopics.length} {subscribedTopics.length === 1 ? 'topic' : 'topics'} active
            </div>
          )}
        </div>

        <div className="space-y-4">
          {subscribedTopics.length === 0 && (
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
              <p className="text-sm text-blue-800">
                <strong>What are topics?</strong> Topics are channels that group notifications.
                Subscribe to a topic (like "alerts" or "server-status") to receive notifications sent to it.
              </p>
            </div>
          )}

          <div>
            <label htmlFor="topic" className="block text-sm font-medium text-gray-700 mb-2">
              Topic Name
            </label>
            <input
              id="topic"
              type="text"
              value={topic}
              onChange={(e) => setTopic(e.target.value)}
              placeholder="e.g., my-alerts"
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-black focus:border-transparent outline-none text-gray-900"
              disabled={isSubscribed}
            />
          </div>

          <button
            onClick={handleSubscribe}
            disabled={isSubscribed || !topic.trim()}
            className={`w-full py-3 px-4 rounded-lg font-medium transition-colors ${isSubscribed || !topic.trim()
              ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
              : 'bg-black text-white hover:bg-gray-800'
              }`}
          >
            {isSubscribed ? 'Subscribed' : 'Subscribe'}
          </button>

          {status && (
            <div
              className={`p-3 rounded-lg text-sm ${status.includes('Error') || status.includes('denied') || status.includes('failed')
                ? 'bg-red-100 text-red-700'
                : status.includes('Success')
                  ? 'bg-green-100 text-green-700'
                  : 'bg-blue-100 text-blue-700'
                }`}
            >
              {status}
            </div>
          )}

          {isSubscribed && (
            <button
              onClick={() => {
                setTopic('');
                setIsSubscribed(false);
                setStatus('');
              }}
              className="w-full py-2 px-4 rounded-lg font-medium text-gray-700 border border-gray-300 hover:bg-gray-50"
            >
              Subscribe to Another Topic
            </button>
          )}
        </div>

        {subscribedTopics.length > 0 && (
          <div className="mt-8 pt-6 border-t border-gray-200">
            <h2 className="text-sm font-medium text-gray-700 mb-3">
              Subscribed Topics ({subscribedTopics.length})
            </h2>
            <div className="space-y-3">
              {subscribedTopics.map((t) => (
                <div key={t} className="bg-gray-50 px-4 py-3 rounded-lg border border-gray-200">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium text-gray-900 font-mono">{t}</span>
                    <span className="text-xs text-green-600 bg-green-50 px-2 py-1 rounded">Active</span>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => handleTestNotification(t)}
                      className="flex-1 py-1.5 px-3 text-xs font-medium bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
                    >
                      Send Test
                    </button>
                    <button
                      onClick={() => setActiveHistoryTopic(t)}
                      className="flex-1 py-1.5 px-3 text-xs font-medium bg-gray-600 text-white rounded hover:bg-gray-700 transition-colors"
                    >
                      History
                    </button>
                    <button
                      onClick={() => {
                        if (confirm(`Unsubscribe from "${t}"?`)) {
                          handleUnsubscribe(t);
                        }
                      }}
                      className="flex-1 py-1.5 px-3 text-xs font-medium bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
                    >
                      Unsubscribe
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="mt-8 pt-6 border-t border-gray-200">
          <h2 className="text-sm font-medium text-gray-700 mb-2">API Usage</h2>
          <p className="text-xs text-gray-500 mb-2">Send notifications from the command line:</p>
          <div className="relative">
            <pre className="bg-gray-900 text-gray-100 p-3 rounded text-xs overflow-x-auto">
              {`curl -X POST ${API_BASE}/publish/${subscribedTopics[0] || 'YOUR_TOPIC'} \\
  -H "Content-Type: application/json" \\
  -d '{"title":"Hello","message":"Test!"}'`}
            </pre>
            <button
              onClick={() => {
                const cmd = `curl -X POST ${API_BASE}/publish/${subscribedTopics[0] || 'YOUR_TOPIC'} -H "Content-Type: application/json" -d '{"title":"Hello","message":"Test!"}'`;
                navigator.clipboard.writeText(cmd);
                setStatus('Copied to clipboard!');
                setTimeout(() => setStatus(''), 2000);
              }}
              className="absolute top-2 right-2 px-2 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600"
            >
              Copy
            </button>
          </div>

          <div className="mt-4">
            <p className="text-xs text-gray-500 mb-2">Or send plain text:</p>
            <div className="relative">
              <pre className="bg-gray-900 text-gray-100 p-3 rounded text-xs overflow-x-auto">
                {`curl -X POST ${API_BASE}/publish/${subscribedTopics[0] || 'YOUR_TOPIC'} \\
  -d "Your message here"`}
              </pre>
              <button
                onClick={() => {
                  const cmd = `curl -X POST ${API_BASE}/publish/${subscribedTopics[0] || 'YOUR_TOPIC'} -d "Your message here"`;
                  navigator.clipboard.writeText(cmd);
                  setStatus('Copied to clipboard!');
                  setTimeout(() => setStatus(''), 2000);
                }}
                className="absolute top-2 right-2 px-2 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600"
              >
                Copy
              </button>
            </div>
          </div>
        </div>
      </div>

      <IOSModal isOpen={showIOSModal} onClose={() => setShowIOSModal(false)} />

      {activeHistoryTopic && (
        <HistoryModal
          isOpen={true}
          onClose={() => setActiveHistoryTopic(null)}
          topic={activeHistoryTopic}
        />
      )}
    </div>
  );
}

export default App;
