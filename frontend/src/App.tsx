import { useEffect, useState, useMemo } from 'react';
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer } from 'recharts';
import { CalendarDays, LogIn } from 'lucide-react';

interface HabitEvent {
  title: string;
  start: string;
  duration: number;
}

export default function App() {
  const [events, setEvents] = useState<HabitEvent[]>([]);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [question, setQuestion] = useState('');
  const [chatHistory, setChatHistory] = useState<{ role: 'user' | 'ai'; text: string }[]>([]);
  const [isTyping, setIsTyping] = useState(false);


  useEffect(() => {
    fetch('http://localhost:8080/api/events', { credentials: 'include' })
      .then((res) => {
        if (res.status === 401) throw new Error('Not logged in');
        return res.json();
      })
      .then((data) => {
        if (data && Array.isArray(data)) {
          setEvents(data);
          setIsAuthenticated(true);
        }
      })
      .catch(() => setIsAuthenticated(false));
  }, []);

  // Compute chart data only when events change. 
  // Recharts can bug out if we give it a brand new array reference on every render.
  const chartData = useMemo(() => {
    const totals: Record<string, number> = {};
    events.forEach((e) => {
      // Handle edge cases where title might be missing
      const title = e.title || 'Untitled';
      totals[title] = (totals[title] || 0) + e.duration;
    });

    return Object.entries(totals)
      .map(([name, duration]) => ({
        name,
        // Convert minutes to hours, keep 1 decimal (e.g., 1.5h) to avoid zero-values
        value: Number((duration / 60).toFixed(1)),
      }))
      .filter((item) => item.value > 0); // Recharts hates 0 or NaN values
  }, [events]);

  const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6'];

  if (!isAuthenticated) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-50">
        <div className="text-center">
          <CalendarDays className="mx-auto h-16 w-16 text-blue-500 mb-4" />
          <h1 className="text-3xl font-bold mb-6 text-gray-800">Habit Coach</h1>
          <a
            href="http://localhost:8080/auth/google/login"
            className="inline-flex items-center justify-center px-6 py-3 bg-white border border-gray-300 rounded-lg shadow-sm hover:bg-gray-50 transition-colors text-gray-700 font-semibold"
          >
            <LogIn className="w-5 h-5 mr-2" />
            Sign in with Google
          </a>
        </div>
      </div>
    );
  }

  const handleAskCoach = async (e: React.FormEvent) => {
    e.preventDefault();
    if(!question.trim()) return;

    // Add user question to UI
    setChatHistory((prev) => [...prev, { role: 'user', text: question }]);
    setQuestion('');
    setIsTyping(true);

    try {
      const res = await fetch('http://localhost:8080/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ question }),
      });
      const data = await res.json();
      setChatHistory((prev) => [...prev, { role: 'ai', text: data.answer }]);
    } catch(err){
      setChatHistory((prev) => [...prev, { role: 'ai', text: 'Error: Unable to get response from Habit Coach.' }]);
    }
    setIsTyping(false);Error: "Unable to get response from Habit Coach."
  };

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-4xl mx-auto">
        <header className="mb-8">
          <h1 className="text-3xl font-bold text-gray-800">Your Habit Dashboard</h1>
          <p className="text-gray-500">Tracking {events.length} total time blocks</p>
        </header>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          {/* The Pie Chart */}
          <div className="bg-white rounded-xl shadow-sm p-6 border border-gray-100">
            <h2 className="text-xl font-semibold mb-4 text-gray-700">Time Breakdown (Hours)</h2>
            {chartData.length > 0 ? (
              <div className="h-80 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={chartData}
                      cx="50%"
                      cy="50%"
                      innerRadius={60}
                      outerRadius={100}
                      paddingAngle={5}
                      dataKey="value"
                      label={({ name, value }) => `${name} (${value}h)`}
                    >
                      {chartData.map((_, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip formatter={(value: number) => [`${value} hours`, 'Time Spent']} />
                  </PieChart>
                </ResponsiveContainer>
              </div>
            ) : (
              <div className="h-80 flex items-center justify-center text-gray-400">
                Processing chart data...
              </div>
            )}
          </div>

          {/* The Habit Coach Chat */}
          <div className="bg-white rounded-xl shadow-sm p-6 border border-gray-100 flex flex-col h-80">
            <h2 className="text-xl font-semibold mb-4 text-gray-700 flex items-center">Habit Coach</h2>
            <div className="flex-1 overflow-y-auto mb-4">
              {chatHistory.map((msg, index) => (
                <div key={index} className={`p-3 rounded-lg mb-2 ${msg.role === 'user' ? 'bg-blue-100 text-blue-800' : 'bg-gray-100 text-gray-800'}`}>
                  {msg.text}
                </div>
              ))}
              {isTyping && (
                <div className="p-3 rounded-lg bg-gray-100 text-gray-800">
                  <span className="animate-pulse">...</span>
                </div>
              )}
            </div>
            <form onSubmit={handleAskCoach} className="flex items-center">
              <input
                type="text"
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder="Ask your habit coach..."
                className="border border-gray-300 rounded-lg py-2 px-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button
                type="submit"
                className="ml-2 bg-blue-500 text-white py-2 px-4 rounded-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                Send
              </button>
            </form>
          </div>
        </div>

      </div>
    </div>
  );
}