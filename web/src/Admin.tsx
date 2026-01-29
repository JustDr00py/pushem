import { useState, useEffect } from 'react';

const API_BASE = window.location.origin;

interface TopicInfo {
  name: string;
  is_protected: boolean;
  subscription_count: number;
  message_count: number;
  created_at?: string;
}

function Admin() {
  const [password, setPassword] = useState('');
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [topics, setTopics] = useState<TopicInfo[]>([]);
  const [status, setStatus] = useState('');
  const [loading, setLoading] = useState(false);
  const [token, setToken] = useState('');

  useEffect(() => {
    // Check if we have a stored token
    const storedToken = sessionStorage.getItem('adminToken');
    if (storedToken) {
      setToken(storedToken);
      setIsAuthenticated(true);
      loadTopics(storedToken);
    }
  }, []);

  const login = async (pwd: string) => {
    try {
      const response = await fetch(`${API_BASE}/api/admin/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ password: pwd }),
      });

      if (response.ok) {
        const data = await response.json();
        const token = data.token;

        setToken(token);
        sessionStorage.setItem('adminToken', token);
        setIsAuthenticated(true);
        setPassword('');
        loadTopics(token);
      } else {
        sessionStorage.removeItem('adminToken');
        const errorText = await response.text();
        setStatus(errorText || 'Invalid password');
        setTimeout(() => setStatus(''), 3000);
      }
    } catch (error) {
      setStatus(`Error: ${error instanceof Error ? error.message : String(error)}`);
      setTimeout(() => setStatus(''), 3000);
    }
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    await login(password);
  };

  const loadTopics = async (authToken: string) => {
    setLoading(true);
    try {
      const response = await fetch(`${API_BASE}/api/admin/topics`, {
        headers: {
          'Authorization': `Bearer ${authToken}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        setTopics(data || []);
      } else if (response.status === 401) {
        // Token expired or invalid
        setStatus('Session expired. Please login again.');
        handleLogout();
      } else {
        setStatus('Failed to load topics');
        setTimeout(() => setStatus(''), 3000);
      }
    } catch (error) {
      setStatus(`Error: ${error instanceof Error ? error.message : String(error)}`);
      setTimeout(() => setStatus(''), 3000);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteTopic = async (topicName: string) => {
    if (!confirm(`Are you sure you want to delete topic "${topicName}"? This will remove all subscriptions and messages.`)) {
      return;
    }

    try {
      const response = await fetch(`${API_BASE}/api/admin/topics/${encodeURIComponent(topicName)}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });

      if (response.ok) {
        setStatus(`Topic "${topicName}" deleted successfully`);
        loadTopics(token);
      } else if (response.status === 401) {
        setStatus('Session expired. Please login again.');
        handleLogout();
      } else {
        setStatus('Failed to delete topic');
      }
      setTimeout(() => setStatus(''), 3000);
    } catch (error) {
      setStatus(`Error: ${error instanceof Error ? error.message : String(error)}`);
      setTimeout(() => setStatus(''), 3000);
    }
  };

  const handleProtectTopic = async (topicName: string) => {
    const secret = prompt(`Enter a secret key to protect topic "${topicName}" (min 8 characters):`);
    if (!secret) return;

    if (secret.length < 8) {
      setStatus('Secret must be at least 8 characters');
      setTimeout(() => setStatus(''), 3000);
      return;
    }

    try {
      const response = await fetch(`${API_BASE}/topics/${encodeURIComponent(topicName)}/protect`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ secret }),
      });

      if (response.ok) {
        setStatus(`Topic "${topicName}" is now protected`);
        loadTopics(token);
      } else if (response.status === 401) {
        setStatus('Session expired. Please login again.');
        handleLogout();
      } else {
        const errorText = await response.text();
        setStatus(errorText || 'Failed to protect topic');
      }
      setTimeout(() => setStatus(''), 3000);
    } catch (error) {
      setStatus(`Error: ${error instanceof Error ? error.message : String(error)}`);
      setTimeout(() => setStatus(''), 3000);
    }
  };

  const handleUnprotectTopic = async (topicName: string) => {
    if (!confirm(`Remove protection from topic "${topicName}"?`)) {
      return;
    }

    try {
      const response = await fetch(`${API_BASE}/api/admin/topics/${encodeURIComponent(topicName)}/protection`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });

      if (response.ok) {
        setStatus(`Protection removed from "${topicName}"`);
        loadTopics(token);
      } else if (response.status === 401) {
        setStatus('Session expired. Please login again.');
        handleLogout();
      } else {
        setStatus('Failed to remove protection');
      }
      setTimeout(() => setStatus(''), 3000);
    } catch (error) {
      setStatus(`Error: ${error instanceof Error ? error.message : String(error)}`);
      setTimeout(() => setStatus(''), 3000);
    }
  };

  const handleLogout = () => {
    sessionStorage.removeItem('adminToken');
    setIsAuthenticated(false);
    setPassword('');
    setToken('');
    setTopics([]);
  };

  const goToMain = () => {
    window.location.href = '/';
  };

  if (!isAuthenticated) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-gray-900 to-gray-800 flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow-xl p-8 max-w-md w-full">
          <div className="text-center mb-8">
            <h1 className="text-3xl font-bold text-gray-900 mb-2">Admin Panel</h1>
            <p className="text-gray-600">Enter password to continue</p>
          </div>

          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-1">
                Admin Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter admin password"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-black focus:border-transparent outline-none text-gray-900"
                autoFocus
              />
            </div>

            <button
              type="submit"
              disabled={!password.trim()}
              className={`w-full py-3 px-4 rounded-lg font-medium transition-colors ${
                !password.trim()
                  ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
                  : 'bg-black text-white hover:bg-gray-800'
              }`}
            >
              Login
            </button>

            {status && (
              <div className="p-3 rounded-lg text-sm bg-red-100 text-red-700">
                {status}
              </div>
            )}

            <button
              type="button"
              onClick={goToMain}
              className="w-full py-2 px-4 rounded-lg font-medium text-gray-700 border border-gray-300 hover:bg-gray-50"
            >
              Back to Main
            </button>
          </form>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-900 to-gray-800 p-4">
      <div className="max-w-6xl mx-auto">
        <div className="bg-white rounded-lg shadow-xl p-8">
          <div className="flex justify-between items-center mb-8">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 mb-2">Admin Panel</h1>
              <p className="text-gray-600">Manage topics and subscriptions</p>
            </div>
            <div className="flex gap-3">
              <button
                onClick={goToMain}
                className="py-2 px-4 rounded-lg font-medium text-gray-700 border border-gray-300 hover:bg-gray-50"
              >
                Main Page
              </button>
              <button
                onClick={() => loadTopics(token)}
                disabled={loading}
                className="py-2 px-4 rounded-lg font-medium bg-blue-600 text-white hover:bg-blue-700 disabled:bg-blue-300"
              >
                {loading ? 'Loading...' : 'Refresh'}
              </button>
              <button
                onClick={handleLogout}
                className="py-2 px-4 rounded-lg font-medium bg-red-600 text-white hover:bg-red-700"
              >
                Logout
              </button>
            </div>
          </div>

          {status && (
            <div
              className={`mb-4 p-3 rounded-lg text-sm ${
                status.includes('Error') || status.includes('Failed')
                  ? 'bg-red-100 text-red-700'
                  : 'bg-green-100 text-green-700'
              }`}
            >
              {status}
            </div>
          )}

          {loading ? (
            <div className="text-center py-12 text-gray-500">Loading topics...</div>
          ) : topics.length === 0 ? (
            <div className="text-center py-12 text-gray-500">
              <p className="text-lg font-medium mb-2">No topics found</p>
              <p className="text-sm">Topics will appear here once users subscribe to them</p>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-xl font-semibold text-gray-900">
                  Topics ({topics.length})
                </h2>
              </div>

              <div className="grid gap-4">
                {topics.map((topic) => (
                  <div
                    key={topic.name}
                    className="border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow"
                  >
                    <div className="flex justify-between items-start mb-3">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <h3 className="text-lg font-semibold text-gray-900 font-mono">
                            {topic.name}
                          </h3>
                          {topic.is_protected && (
                            <span className="text-xs bg-yellow-100 text-yellow-800 px-2 py-1 rounded font-medium">
                              Protected
                            </span>
                          )}
                        </div>
                        {topic.created_at && (
                          <p className="text-xs text-gray-500">
                            Created: {new Date(topic.created_at).toLocaleString()}
                          </p>
                        )}
                      </div>
                      <div className="flex gap-2">
                        {topic.is_protected ? (
                          <button
                            onClick={() => handleUnprotectTopic(topic.name)}
                            className="py-1.5 px-3 text-sm font-medium bg-yellow-600 text-white rounded hover:bg-yellow-700"
                          >
                            Unprotect
                          </button>
                        ) : (
                          <button
                            onClick={() => handleProtectTopic(topic.name)}
                            className="py-1.5 px-3 text-sm font-medium bg-green-600 text-white rounded hover:bg-green-700"
                          >
                            Protect
                          </button>
                        )}
                        <button
                          onClick={() => handleDeleteTopic(topic.name)}
                          className="py-1.5 px-3 text-sm font-medium bg-red-600 text-white rounded hover:bg-red-700"
                        >
                          Delete
                        </button>
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div className="bg-blue-50 p-3 rounded">
                        <p className="text-blue-600 font-medium mb-1">Subscriptions</p>
                        <p className="text-2xl font-bold text-blue-900">
                          {topic.subscription_count}
                        </p>
                      </div>
                      <div className="bg-green-50 p-3 rounded">
                        <p className="text-green-600 font-medium mb-1">Messages</p>
                        <p className="text-2xl font-bold text-green-900">
                          {topic.message_count}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default Admin;
