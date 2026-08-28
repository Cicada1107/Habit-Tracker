import { useEffect, useState, useMemo, useRef } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Cell } from 'recharts';
import { CalendarDays, LogIn, ChevronUp, ChevronDown, Send, Trophy, Flame, BrainCircuit, Moon, Sun, Plus, Activity, Target } from 'lucide-react';

interface HabitEvent {
  title: string;
  start: string;
  duration: number; // in minutes
}

interface Habit {
  id: string;
  name: string;
  description: string;
}

// --- Heatmap Component ---
function HabitHeatmap({ habit, events }: { habit: Habit, events: HabitEvent[] }) {
  const { heatmapGrid, maxDailyMin } = useMemo(() => {
    const dailyTotals: Record<string, number> = {};
    events.forEach((e) => {
      // The backend mapper now sets e.title to the exact habit name if mapped
      if (e.title === habit.name) {
        const day = e.start.split('T')[0];
        dailyTotals[day] = (dailyTotals[day] || 0) + e.duration;
      }
    });

    const grid = [];
    const today = new Date();
    const startDate = new Date(today);
    startDate.setDate(today.getDate() - 89);
    
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
  }, [events, habit.name]);

  const getHeatmapColor = (value: number) => {
    if (value === 0) return 'bg-gray-100 dark:bg-slate-700';
    const intensity = value / maxDailyMin;
    if (intensity < 0.25) return 'bg-emerald-200 dark:bg-emerald-900';
    if (intensity < 0.5) return 'bg-emerald-300 dark:bg-emerald-700';
    if (intensity < 0.75) return 'bg-emerald-400 dark:bg-emerald-500';
    return 'bg-emerald-500 dark:bg-emerald-400';
  };

  return (
    <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-sm border border-gray-100 dark:border-slate-700 p-6 relative group transition-colors">
      <h3 className="text-md font-bold text-gray-800 dark:text-slate-100 mb-1">{habit.name}</h3>
      <p className="text-xs text-gray-500 dark:text-slate-400 mb-4 line-clamp-1" title={habit.description}>{habit.description}</p>
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
  const [habits, setHabits] = useState<Habit[]>([]);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  
  // Dark Mode State
  const [isDark, setIsDark] = useState(() => localStorage.getItem('theme') === 'dark');
  
  // Chat State
  const [question, setQuestion] = useState('');
  const [chatHistory, setChatHistory] = useState<{ role: 'user' | 'ai'; text: string; thought?: string }[]>([]);
  const [isTyping, setIsTyping] = useState(false);
  const [isChatOpen, setIsChatOpen] = useState(false);
  
  // Add Habit Modal State
  const [showAddModal, setShowAddModal] = useState(false);
  const [newHabitName, setNewHabitName] = useState('');
  const [newHabitDesc, setNewHabitDesc] = useState('');

  const chatScrollRef = useRef<HTMLDivElement>(null);

  // Apply dark mode
  useEffect(() => {
    if (isDark) document.documentElement.classList.add('dark');
    else document.documentElement.classList.remove('dark');
    localStorage.setItem('theme', isDark ? 'dark' : 'light');
  }, [isDark]);

  // Auto-scroll chat to bottom
  useEffect(() => {
    if (chatScrollRef.current) {
      chatScrollRef.current.scrollTop = chatScrollRef.current.scrollHeight;
    }
  }, [chatHistory, isTyping, isChatOpen]);

  // Fetch Data
  const fetchData = async () => {
    try {
      const [eventsRes, habitsRes] = await Promise.all([
        fetch('http://localhost:8080/api/events', { credentials: 'include' }),
        fetch('http://localhost:8080/api/habits', { credentials: 'include' })
      ]);
      
      if (eventsRes.status === 401) throw new Error('Not logged in');
      
      const eventsData = await eventsRes.json();
      const habitsData = await habitsRes.json();
      
      if (eventsData && Array.isArray(eventsData)) {
        const now = new Date();
        const pastEvents = eventsData.filter(e => new Date(e.start) <= now);
        setEvents(pastEvents);
      }
      
      if (habitsData && Array.isArray(habitsData)) {
        setHabits(habitsData);
      }
      
      setIsAuthenticated(true);
    } catch (err) {
      setIsAuthenticated(false);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleAddHabit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newHabitName.trim()) return;

    try {
      const res = await fetch('http://localhost:8080/api/habits', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ name: newHabitName.trim(), description: newHabitDesc.trim() }),
      });
      if (res.ok) {
        setNewHabitName('');
        setNewHabitDesc('');
        setShowAddModal(false);
        fetchData(); // Refresh to get the new habit
      }
    } catch (err) {
      console.error("Failed to add habit", err);
    }
  };

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

  // --- Data Processors (Only for explicitly mapped habits!) ---
  const mappedEvents = useMemo(() => {
    const habitNames = new Set(habits.map(h => h.name));
    return events.filter(e => habitNames.has(e.title));
  }, [events, habits]);

  const topHabits = useMemo(() => {
    const totals: Record<string, number> = {};
    mappedEvents.forEach((e) => {
      totals[e.title] = (totals[e.title] || 0) + e.duration;
    });
    return Object.entries(totals)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 5)
      .map(([name, duration]) => ({ name, durationHours: Number((duration / 60).toFixed(1)) }));
  }, [mappedEvents]);

  const consistencyData = useMemo(() => {
    const habitDays: Record<string, Set<string>> = {};
    mappedEvents.forEach((e) => {
      const day = e.start.split('T')[0];
      if (!habitDays[e.title]) habitDays[e.title] = new Set();
      habitDays[e.title].add(day);
    });
    return Object.entries(habitDays)
      .map(([name, daysSet]) => ({ name, days: daysSet.size }))
      .sort((a, b) => b.days - a.days)
      .slice(0, 5);
  }, [mappedEvents]);

  const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'];

  if (isLoading) {
    return <div className={`flex h-screen items-center justify-center ${isDark ? 'bg-slate-900' : 'bg-gray-50'}`} />
  }

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

  // ONBOARDING FLOW
  if (habits.length === 0) {
    return (
      <div className={`min-h-screen flex items-center justify-center p-6 ${isDark ? 'bg-slate-900' : 'bg-gray-50'}`}>
        <div className={`max-w-md w-full ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-gray-200'} rounded-2xl shadow-xl border p-8`}>
          <div className="w-12 h-12 bg-blue-100 text-blue-600 rounded-xl flex items-center justify-center mb-6">
            <Target size={24} />
          </div>
          <h1 className={`text-2xl font-bold mb-2 ${isDark ? 'text-white' : 'text-gray-900'}`}>What do you want to track?</h1>
          <p className={`text-sm mb-6 ${isDark ? 'text-slate-400' : 'text-gray-500'}`}>
            Define your core habits. Our background AI will automatically scan your Google Calendar and intelligently map related events (like "Hitting the weights") to your defined habit (like "Gym").
          </p>
          
          <form onSubmit={handleAddHabit} className="space-y-4">
            <div>
              <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-slate-300' : 'text-gray-700'}`}>Habit Name</label>
              <input 
                required
                placeholder="e.g. Reading"
                value={newHabitName}
                onChange={e => setNewHabitName(e.target.value)}
                className={`w-full p-2.5 rounded-lg border ${isDark ? 'bg-slate-900 border-slate-700 text-white' : 'bg-gray-50 border-gray-300'} focus:ring-2 focus:ring-blue-500 outline-none`}
              />
            </div>
            <div>
              <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-slate-300' : 'text-gray-700'}`}>Description (Helps AI categorize)</label>
              <input 
                placeholder="e.g. Reading books, kindle, or articles"
                value={newHabitDesc}
                onChange={e => setNewHabitDesc(e.target.value)}
                className={`w-full p-2.5 rounded-lg border ${isDark ? 'bg-slate-900 border-slate-700 text-white' : 'bg-gray-50 border-gray-300'} focus:ring-2 focus:ring-blue-500 outline-none`}
              />
            </div>
            <button type="submit" className="w-full bg-blue-600 text-white font-semibold py-3 rounded-lg hover:bg-blue-700 transition">
              Save & Start Tracking
            </button>
          </form>
        </div>
      </div>
    );
  }

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
            <button 
              onClick={() => setShowAddModal(true)}
              className="flex items-center gap-1.5 text-sm font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300 px-4 py-1.5 rounded-full hover:bg-blue-200 dark:hover:bg-blue-900 transition-colors"
            >
              <Plus size={16} /> New Habit
            </button>
            <div className="text-sm font-medium text-gray-500 dark:text-slate-400 bg-gray-100 dark:bg-slate-700 px-4 py-1.5 rounded-full hidden sm:block">
              {mappedEvents.length} Mapped Events
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

      <main className="max-w-6xl mx-auto px-4 sm:px-6 mt-6 sm:mt-10 grid grid-cols-1 lg:grid-cols-3 gap-6 sm:gap-8">
        
        {/* Left Column (2/3 width) - Tracked Habits & Charts */}
        <div className="lg:col-span-2 space-y-6 sm:space-y-8">
          
          <section>
            <h2 className="text-xl font-bold text-gray-800 dark:text-slate-100 mb-6 flex items-center gap-2">
              <Target className="w-5 h-5 text-blue-500" /> Tracked Habits
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {habits.map(habit => (
                <HabitHeatmap 
                  key={habit.id} 
                  habit={habit} 
                  events={mappedEvents} 
                />
              ))}
            </div>
          </section>

          {/* Habit vs Consistency Bar Chart */}
          <section className="bg-white dark:bg-slate-800 rounded-2xl shadow-sm border border-gray-100 dark:border-slate-700 p-6 transition-colors">
            <div className="flex items-center gap-2 mb-6">
              <Flame className="w-5 h-5 text-orange-500" />
              <h2 className="text-lg font-bold text-gray-800 dark:text-slate-100">Consistency Ranking</h2>
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
              <div className="h-72 flex items-center justify-center text-gray-400 text-sm">No mapped data yet...</div>
            )}
          </section>
        </div>

        {/* Right Column (1/3 width) - Leaderboards & Stats */}
        <div className="space-y-6 sm:space-y-8">
          
          {/* Top 5 Habits */}
          <section className="bg-white dark:bg-slate-800 rounded-2xl shadow-sm border border-gray-100 dark:border-slate-700 p-6 transition-colors">
            <div className="flex items-center gap-2 mb-6">
              <Trophy className="w-5 h-5 text-yellow-500" />
              <h2 className="text-lg font-bold text-gray-800 dark:text-slate-100">Top 5 Habits</h2>
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
                <div className="text-sm text-gray-400 text-center py-4">No mapped data yet</div>
              )}
            </div>
          </section>
          
          {/* Quick Summary Card */}
          <section className="bg-gradient-to-br from-blue-600 to-indigo-700 rounded-2xl shadow-sm p-6 text-white relative overflow-hidden">
             <div className="relative z-10">
               <h3 className="text-blue-100 font-medium mb-1">Total Mapped Time</h3>
               <div className="text-4xl font-extrabold tracking-tight">
                 {Math.round(mappedEvents.reduce((acc, e) => acc + e.duration, 0) / 60)}<span className="text-2xl font-semibold text-blue-200 ml-1">hrs</span>
               </div>
               <p className="text-sm text-blue-200 mt-4 leading-relaxed">
                 Only tracking time spent on your explicit goals. Pure signal, no noise.
               </p>
             </div>
             <BrainCircuit className="absolute -bottom-4 -right-4 w-32 h-32 text-white opacity-10" />
          </section>
        </div>
      </main>

      {/* Add Habit Modal (Secondary) */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
          <div className={`max-w-md w-full ${isDark ? 'bg-slate-800' : 'bg-white'} rounded-2xl shadow-xl p-6`}>
            <h2 className={`text-xl font-bold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>Add New Habit</h2>
            <form onSubmit={handleAddHabit} className="space-y-4">
              <div>
                <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-slate-300' : 'text-gray-700'}`}>Habit Name</label>
                <input required autoFocus value={newHabitName} onChange={e => setNewHabitName(e.target.value)} className={`w-full p-2.5 rounded-lg border ${isDark ? 'bg-slate-900 border-slate-700 text-white' : 'bg-gray-50 border-gray-300'} outline-none`} />
              </div>
              <div>
                <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-slate-300' : 'text-gray-700'}`}>Description</label>
                <input value={newHabitDesc} onChange={e => setNewHabitDesc(e.target.value)} className={`w-full p-2.5 rounded-lg border ${isDark ? 'bg-slate-900 border-slate-700 text-white' : 'bg-gray-50 border-gray-300'} outline-none`} />
              </div>
              <div className="flex gap-2 justify-end pt-2">
                <button type="button" onClick={() => setShowAddModal(false)} className={`px-4 py-2 rounded-lg font-medium ${isDark ? 'text-slate-300 hover:bg-slate-700' : 'text-gray-600 hover:bg-gray-100'}`}>Cancel</button>
                <button type="submit" className="px-4 py-2 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700">Save Habit</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* LinkedIn Style Chat Widget */}
      <div className={`fixed bottom-0 right-0 sm:right-10 w-full sm:w-[360px] bg-white dark:bg-slate-800 sm:rounded-t-xl shadow-[0_-4px_20px_-5px_rgba(0,0,0,0.1)] border-t sm:border-x border-gray-200 dark:border-slate-700 flex flex-col z-50 transition-all duration-300 ease-in-out ${isChatOpen ? 'h-[80vh] sm:h-[500px]' : 'h-14 hover:bg-gray-50 dark:hover:bg-slate-700'}`}>
        
        {/* Chat Header */}
        <div onClick={() => setIsChatOpen(!isChatOpen)} className="bg-slate-800 text-white px-4 h-14 flex justify-between items-center cursor-pointer sm:rounded-t-xl select-none">
          <div className="flex items-center gap-3">
            <div className="relative">
              <div className="bg-blue-500 p-1.5 rounded-full"><BrainCircuit size={18} /></div>
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
            <form onSubmit={handleAskCoach} className="p-3 bg-white dark:bg-slate-800 border-t border-gray-100 dark:border-slate-700 flex items-center gap-2">
              <input type="text" value={question} onChange={(e) => setQuestion(e.target.value)} placeholder="Ask about your habits..." className="flex-1 bg-gray-100 dark:bg-slate-700 text-gray-900 dark:text-white text-sm border-transparent focus:border-blue-500 focus:bg-white dark:focus:bg-slate-600 focus:ring-0 rounded-full py-2.5 px-4 outline-none transition-all" autoComplete="off" />
              <button type="submit" disabled={isTyping || !question.trim()} className="bg-blue-600 disabled:bg-gray-300 dark:disabled:bg-slate-600 text-white p-2.5 rounded-full hover:bg-blue-700 transition-colors flex-shrink-0">
                <Send size={16} className={question.trim() ? "translate-x-0.5" : ""} />
              </button>
            </form>
          </>
        )}
      </div>
    </div>
  );
}
