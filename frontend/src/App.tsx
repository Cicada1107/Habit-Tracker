import { useEffect, useState, useMemo, useRef } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Cell } from 'recharts';
import { CalendarDays, LogIn, ChevronUp, ChevronDown, Send, Trophy, Flame, BrainCircuit, Moon, Sun, Plus, Trash2, Activity } from 'lucide-react';

interface HabitEvent {
  title: string;
  start: string;
  duration: number; // in minutes
}

// --- Heatmap Component ---
function HabitHeatmap({ title, events, onRemove }: { title: string, events: HabitEvent[], onRemove?: () => void }) {
  const { heatmapGrid, maxDailyMin } = useMemo(() => {
    const dailyTotals: Record<string, number> = {};
    events.forEach((e) => {
      if (e.title === title) {
        const day = e.start.split('T')[0];
        dailyTotals[day] = (dailyTotals[day] || 0) + e.duration;
      }
    });

    const grid = [];
    const today = new Date();
    const startDate = new Date(today);
    startDate.setDate(today.getDate() - 89);
    
    // Pad beginning to align with Sunday (0)
    const startDayOfWeek = startDate.getDay();
    for (let i = 0; i < startDayOfWeek; i++) {
      grid.push(null); 
    }

    let maxVal = 1;
    for (let i = 89; i >= 0; i--) {
      const d = new Date(today);
      d.setDate(today.getDate() - i);
      const dateStr = d.toISOString().split('T')[0];
      const val = dailyTotals[dateStr] || 0;
      if (val > maxVal) maxVal = val;
      grid.push({ date: dateStr, value: val });
    }
    return { heatmapGrid: grid, maxDailyMin: maxVal };
  }, [events, title]);

  const getHeatmapColor = (value: number) => {
    if (value === 0) return 'bg-gray-100 dark:bg-slate-700';
    const intensity = value / maxDailyMin;
    if (intensity < 0.25) return 'bg-emerald-200 dark:bg-emerald-900';
    if (intensity < 0.5) return 'bg-emerald-300 dark:bg-emerald-700';
    if (intensity < 0.75) return 'bg-emerald-400 dark:bg-emerald-500';
    return 'bg-emerald-500 dark:bg-emerald-400';
  };

  return (
    <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-sm border border-gray-100 dark:border-slate-700 p-6 relative group">
      {onRemove && (
        <button 
          onClick={onRemove} 
          className="absolute top-4 right-4 text-gray-400 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity"
          title="Untrack Habit"
        >
          <Trash2 size={16} />
        </button>
      )}
      <h3 className="text-md font-bold text-gray-800 dark:text-slate-100 mb-4">{title}</h3>
      <div className="overflow-x-auto pb-2">
        <div className="grid grid-rows-7 grid-flow-col gap-1.5 min-w-max">
          {heatmapGrid.map((day, i) => {
            if (!day) return <div key={i} className="w-3.5 h-3.5 bg-transparent rounded-sm" />;
            return (
              <div 
                key={day.date} 
                title={`${day.date}: ${Math.round(day.value)} mins`} 
                className={`w-3.5 h-3.5 rounded-sm transition-colors cursor-pointer hover:ring-2 hover:ring-offset-1 hover:ring-emerald-400 ${getHeatmapColor(day.value)}`} 
              />
            );
          })}
        </div>
      </div>
    </div>
  );
}

export default function App() {
  const [events, setEvents] = useState<HabitEvent[]>([]);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  
  // Dark Mode State
  const [isDark, setIsDark] = useState(() => localStorage.getItem('theme') === 'dark');
  
  // Tracked Habits State
  const [trackedHabits, setTrackedHabits] = useState<string[]>(() => {
    const saved = localStorage.getItem('trackedHabits');
    return saved ? JSON.parse(saved) : [];
  });
  const [newHabitInput, setNewHabitInput] = useState('');

  // Chat State
  const [question, setQuestion] = useState('');
  const [chatHistory, setChatHistory] = useState<{ role: 'user' | 'ai'; text: string; thought?: string }[]>([]);
  const [isTyping, setIsTyping] = useState(false);
  const [isChatOpen, setIsChatOpen] = useState(false);
  
  const chatScrollRef = useRef<HTMLDivElement>(null);

  // Apply dark mode
  useEffect(() => {
    if (isDark) document.documentElement.classList.add('dark');
    else document.documentElement.classList.remove('dark');
    localStorage.setItem('theme', isDark ? 'dark' : 'light');
  }, [isDark]);

  useEffect(() => {
    localStorage.setItem('trackedHabits', JSON.stringify(trackedHabits));
  }, [trackedHabits]);

  // Auto-scroll chat to bottom
  useEffect(() => {
    if (chatScrollRef.current) {
      chatScrollRef.current.scrollTop = chatScrollRef.current.scrollHeight;
    }
  }, [chatHistory, isTyping, isChatOpen]);

  useEffect(() => {
    fetch('http://localhost:8080/api/events', { credentials: 'include' })
      .then((res) => {
        if (res.status === 401) throw new Error('Not logged in');
        return res.json();
      })
      .then((data) => {
        if (data && Array.isArray(data)) {
          // Feature 2: Filter out future events
          const now = new Date();
          const pastEvents = data.filter(e => new Date(e.start) <= now);
          setEvents(pastEvents);
          setIsAuthenticated(true);
        }
      })
      .catch(() => setIsAuthenticated(false));
  }, []);

  // --- Data Processors ---

  // All unique habit names for the datalist autocomplete
  const uniqueHabitNames = useMemo(() => {
    return Array.from(new Set(events.map(e => e.title || 'Untitled'))).filter(Boolean);
  }, [events]);

  const topHabits = useMemo(() => {
    const totals: Record<string, number> = {};
    events.forEach((e) => {
      const title = e.title || 'Untitled';
      totals[title] = (totals[title] || 0) + e.duration;
    });
    return Object.entries(totals)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 5)
      .map(([name, duration]) => ({ name, durationHours: Number((duration / 60).toFixed(1)) }));
  }, [events]);

  const consistencyData = useMemo(() => {
    const habitDays: Record<string, Set<string>> = {};
    events.forEach((e) => {
      const title = e.title || 'Untitled';
      const day = e.start.split('T')[0];
      if (!habitDays[title]) habitDays[title] = new Set();
      habitDays[title].add(day);
    });
    return Object.entries(habitDays)
      .map(([name, daysSet]) => ({ name, days: daysSet.size }))
      .sort((a, b) => b.days - a.days)
      .slice(0, 5);
  }, [events]);

  const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'];

  if (!isAuthenticated) {
    return (
      <div className={`flex h-screen items-center justify-center ${isDark ? 'bg-slate-900' : 'bg-gray-50'}`}>
        <div className={`text-center ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-gray-100'} p-10 rounded-2xl shadow-xl border`}>
          <CalendarDays className="mx-auto h-20 w-20 text-blue-600 mb-6" />
          <h1 className={`text-4xl font-extrabold mb-3 tracking-tight ${isDark ? 'text-white' : 'text-gray-900'}`}>Habit Coach</h1>
          <p className={`${isDark ? 'text-slate-400' : 'text-gray-500'} mb-8 max-w-sm`}>Connect your Google Calendar and transform your routines with AI-driven insights.</p>
          <a
            href="http://localhost:8080/auth/google/login"
            className="inline-flex items-center justify-center px-8 py-3.5 bg-blue-600 text-white rounded-xl shadow-md hover:bg-blue-700 transition-all font-semibold w-full"
          >
            <LogIn className="w-5 h-5 mr-3" />
            Sign in with Google
          </a>
        </div>
      </div>
    );
  }

  const handleAskCoach = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!question.trim()) return;

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
      setChatHistory((prev) => [...prev, { role: 'ai', text: data.answer, thought: data.thought }]);
    } catch (err) {
      setChatHistory((prev) => [...prev, { role: 'ai', text: 'Error: Unable to connect to your Habit Coach.' }]);
    }
    setIsTyping(false);
  };

  const trackNewHabit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = newHabitInput.trim();
    if (trimmed && !trackedHabits.includes(trimmed)) {
      setTrackedHabits([...trackedHabits, trimmed]);
    }
    setNewHabitInput('');
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 font-sans pb-20 transition-colors duration-200">
      
      {/* Top Navbar */}
      <nav className="bg-white dark:bg-slate-800 shadow-sm border-b border-gray-200 dark:border-slate-700 sticky top-0 z-30 transition-colors">
        <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="bg-blue-600 p-2 rounded-lg">
              <BrainCircuit className="text-white w-5 h-5" />
            </div>
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">Habit Coach</h1>
          </div>
          <div className="flex items-center gap-4">
            <div className="text-sm font-medium text-gray-500 dark:text-slate-400 bg-gray-100 dark:bg-slate-700 px-4 py-1.5 rounded-full">
              {events.length} Past Events
            </div>
            <button 
              onClick={() => setIsDark(!isDark)}
              className="p-2 rounded-full hover:bg-gray-100 dark:hover:bg-slate-700 text-gray-600 dark:text-slate-300 transition-colors"
            >
              {isDark ? <Sun size={20} /> : <Moon size={20} />}
            </button>
          </div>
        </div>
      </nav>

      <main className="max-w-6xl mx-auto px-6 mt-10 grid grid-cols-1 lg:grid-cols-3 gap-8">
        
        {/* Left Column (2/3 width) - Tracked Habits & Charts */}
        <div className="lg:col-span-2 space-y-8">
          
          {/* Specific Tracked Habits Section */}
          <section>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold text-gray-800 dark:text-slate-100">Tracked Habits</h2>
              <form onSubmit={trackNewHabit} className="flex items-center gap-2">
                <input
                  list="habit-suggestions"
                  value={newHabitInput}
                  onChange={e => setNewHabitInput(e.target.value)}
                  placeholder="e.g. Reading, Gym..."
                  className="text-sm border border-gray-300 dark:border-slate-600 bg-white dark:bg-slate-800 dark:text-white rounded-lg py-1.5 px-3 focus:ring-2 focus:ring-blue-500 outline-none"
                />
                <datalist id="habit-suggestions">
                  {uniqueHabitNames.map(name => <option key={name} value={name} />)}
                </datalist>
                <button type="submit" className="bg-blue-600 text-white p-1.5 rounded-lg hover:bg-blue-700 transition">
                  <Plus size={18} />
                </button>
              </form>
            </div>
            
            {trackedHabits.length === 0 ? (
              <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-sm border border-dashed border-gray-300 dark:border-slate-600 p-10 text-center">
                <p className="text-gray-500 dark:text-slate-400">You aren't tracking any specific habits yet.</p>
                <p className="text-sm text-gray-400 dark:text-slate-500 mt-2">Add a habit above to generate its heatmap!</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {trackedHabits.map(habit => (
                  <HabitHeatmap 
                    key={habit} 
                    title={habit} 
                    events={events} 
                    onRemove={() => setTrackedHabits(trackedHabits.filter(h => h !== habit))}
                  />
                ))}
              </div>
            )}
          </section>

          {/* Habit vs Consistency Bar Chart */}
          <section className="bg-white dark:bg-slate-800 rounded-2xl shadow-sm border border-gray-100 dark:border-slate-700 p-6 transition-colors">
            <div className="flex items-center gap-2 mb-6">
              <Flame className="w-5 h-5 text-orange-500" />
              <h2 className="text-lg font-bold text-gray-800 dark:text-slate-100">Habit Consistency</h2>
              <span className="text-sm font-normal text-gray-400 dark:text-slate-500 ml-auto">Unique Days Active</span>
            </div>
            {consistencyData.length > 0 ? (
              <div className="h-72 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={consistencyData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={isDark ? '#334155' : '#f1f5f9'} />
                    <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 12, fill: isDark ? '#94a3b8' : '#64748b' }} />
                    <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 12, fill: isDark ? '#94a3b8' : '#64748b' }} />
                    <Tooltip 
                      cursor={{ fill: isDark ? '#334155' : '#f8fafc' }} 
                      contentStyle={{ borderRadius: '8px', border: 'none', backgroundColor: isDark ? '#1e293b' : '#fff', color: isDark ? '#f8fafc' : '#000', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }} 
                    />
                    <Bar dataKey="days" radius={[4, 4, 0, 0]}>
                      {consistencyData.map((_, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>
            ) : (
              <div className="h-72 flex items-center justify-center text-gray-400 text-sm">Waiting for calendar data...</div>
            )}
          </section>
        </div>

        {/* Right Column (1/3 width) - Leaderboards & Stats */}
        <div className="space-y-8">
          
          {/* Top 5 Habits */}
          <section className="bg-white dark:bg-slate-800 rounded-2xl shadow-sm border border-gray-100 dark:border-slate-700 p-6 transition-colors">
            <div className="flex items-center gap-2 mb-6">
              <Trophy className="w-5 h-5 text-yellow-500" />
              <h2 className="text-lg font-bold text-gray-800 dark:text-slate-100">Top 5 Habits (All Time)</h2>
            </div>
            <div className="space-y-4">
              {topHabits.length > 0 ? (
                topHabits.map((habit) => (
                  <div key={habit.name} className="relative">
                    <div className="flex justify-between items-end mb-1">
                      <span className="font-semibold text-gray-700 dark:text-slate-300 text-sm truncate pr-4">{habit.name}</span>
                      <span className="text-sm font-bold text-blue-600 dark:text-blue-400">{habit.durationHours}h</span>
                    </div>
                    <div className="w-full bg-gray-100 dark:bg-slate-700 rounded-full h-2">
                      <div 
                        className="bg-blue-500 h-2 rounded-full transition-all duration-1000" 
                        style={{ width: `${Math.min(100, (habit.durationHours / (topHabits[0]?.durationHours || 1)) * 100)}%` }}
                      />
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-sm text-gray-400 text-center py-4">No habits logged yet</div>
              )}
            </div>
          </section>

          {/* Quick Summary Card */}
          <section className="bg-gradient-to-br from-blue-600 to-indigo-700 rounded-2xl shadow-sm p-6 text-white relative overflow-hidden">
             <div className="relative z-10">
               <h3 className="text-blue-100 font-medium mb-1">Total Time Invested</h3>
               <div className="text-4xl font-extrabold tracking-tight">
                 {Math.round(events.reduce((acc, e) => acc + e.duration, 0) / 60)}<span className="text-2xl font-semibold text-blue-200 ml-1">hrs</span>
               </div>
               <p className="text-sm text-blue-200 mt-4 leading-relaxed">
                 You're building incredible momentum. Keep syncing your Google Calendar!
               </p>
             </div>
             <BrainCircuit className="absolute -bottom-4 -right-4 w-32 h-32 text-white opacity-10" />
          </section>
        </div>
      </main>

      {/* LinkedIn Style Chat Widget (Bottom Right) */}
      <div className={`fixed bottom-0 right-4 sm:right-10 w-full sm:w-[360px] bg-white dark:bg-slate-800 rounded-t-xl shadow-2xl border border-gray-200 dark:border-slate-700 flex flex-col z-50 transition-all duration-300 ease-in-out ${isChatOpen ? 'h-[500px]' : 'h-14 hover:bg-gray-50 dark:hover:bg-slate-700'}`}>
        
        {/* Chat Header (Click to toggle) */}
        <div 
          onClick={() => setIsChatOpen(!isChatOpen)}
          className="bg-slate-800 text-white px-4 h-14 flex justify-between items-center cursor-pointer rounded-t-xl select-none"
        >
          <div className="flex items-center gap-3">
            <div className="relative">
              <div className="bg-blue-500 p-1.5 rounded-full">
                <BrainCircuit size={18} />
              </div>
              <span className="absolute bottom-0 right-0 w-2.5 h-2.5 bg-green-400 border-2 border-slate-800 rounded-full"></span>
            </div>
            <span className="font-semibold tracking-wide">Habit Coach</span>
          </div>
          <div className="text-gray-300 hover:text-white transition-colors">
            {isChatOpen ? <ChevronDown size={22} /> : <ChevronUp size={22} />}
          </div>
        </div>

        {/* Chat Body */}
        {isChatOpen && (
          <>
            <div ref={chatScrollRef} className="flex-1 overflow-y-auto p-4 bg-gray-50 dark:bg-slate-900 flex flex-col gap-4">
              
              {/* Initial Welcome Message */}
              {chatHistory.length === 0 && (
                 <div className="flex gap-3 max-w-[85%]">
                   <div className="bg-white dark:bg-slate-800 border border-gray-200 dark:border-slate-700 text-gray-800 dark:text-slate-200 p-3 rounded-2xl rounded-tl-sm text-sm shadow-sm">
                     Hi! I'm your AI Habit Coach. Ask me to analyze your consistency or calculate probabilities for your goals!
                   </div>
                 </div>
              )}

              {chatHistory.map((msg, index) => (
                <div key={index} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'} w-full`}>
                  <div className={`max-w-[85%] ${msg.role === 'user' ? 'bg-blue-600 text-white rounded-2xl rounded-tr-sm' : 'bg-white dark:bg-slate-800 border border-gray-200 dark:border-slate-700 text-gray-800 dark:text-slate-200 rounded-2xl rounded-tl-sm'} p-3 text-sm shadow-sm flex flex-col`}>
                    
                    {/* The AI's Thought Dropdown */}
                    {msg.thought && (
                      <details className="mb-2 text-xs text-gray-600 dark:text-gray-400 bg-gray-100/50 dark:bg-slate-700/50 p-2 rounded-lg border border-gray-200 dark:border-slate-600 cursor-pointer">
                        <summary className="font-semibold select-none flex items-center gap-1">
                          <Activity size={12} className="text-blue-500 dark:text-blue-400" />
                          View AI Actions & Thoughts
                        </summary>
                        <div className="mt-2 whitespace-pre-wrap font-mono text-[10px] leading-relaxed opacity-80">{msg.thought}</div>
                      </details>
                    )}

                    <div className="whitespace-pre-wrap leading-relaxed">{msg.text}</div>
                  </div>
                </div>
              ))}

              {isTyping && (
                <div className="flex justify-start max-w-[85%]">
                  <div className="bg-white dark:bg-slate-800 border border-gray-200 dark:border-slate-700 p-4 rounded-2xl rounded-tl-sm shadow-sm flex items-center gap-1.5">
                    <div className="w-1.5 h-1.5 bg-gray-400 dark:bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                    <div className="w-1.5 h-1.5 bg-gray-400 dark:bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                    <div className="w-1.5 h-1.5 bg-gray-400 dark:bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                  </div>
                </div>
              )}
            </div>

            {/* Chat Input */}
            <form onSubmit={handleAskCoach} className="p-3 bg-white dark:bg-slate-800 border-t border-gray-100 dark:border-slate-700 flex items-center gap-2">
              <input
                type="text"
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder="Ask about your habits..."
                className="flex-1 bg-gray-100 dark:bg-slate-700 text-gray-900 dark:text-white text-sm border-transparent focus:border-blue-500 focus:bg-white dark:focus:bg-slate-600 focus:ring-0 rounded-full py-2.5 px-4 outline-none transition-all"
                autoComplete="off"
              />
              <button
                type="submit"
                disabled={isTyping || !question.trim()}
                className="bg-blue-600 disabled:bg-gray-300 dark:disabled:bg-slate-600 text-white p-2.5 rounded-full hover:bg-blue-700 transition-colors flex-shrink-0"
              >
                <Send size={16} className={question.trim() ? "translate-x-0.5" : ""} />
              </button>
            </form>
          </>
        )}
      </div>

    </div>
  );
}