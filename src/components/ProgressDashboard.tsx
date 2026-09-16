import { Chart as ChartJS, CategoryScale, LinearScale, BarElement, Tooltip, Legend } from 'chart.js';
import { Bar } from 'react-chartjs-2';
import type { DashboardData } from '../lib/progress';
ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip, Legend);

export function ProgressDashboard({ data }: { data: DashboardData }) {
  return <div className="space-y-6">
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
      {[[`${data.currentHours.toFixed(1)}h`, `Week ${data.currentWeek}`], [`${data.totalHours.toFixed(1)}h`, 'All recorded study'], [String(data.weeksLogged), 'Weeks with hours'], [`${data.recentAverage.toFixed(1)}h`, '4-week average']].map(([value, label]) =>
        <div key={label} className="rounded-lg bg-zinc-900 border border-zinc-800 p-4"><p className="text-xl font-semibold">{value}</p><p className="text-sm text-zinc-400">{label}</p></div>)}
    </div>
    <section className="space-y-3">
      <h2 className="text-xl font-semibold">This week · {data.currentDates || 'No entry yet'}</h2>
      <div className="grid sm:grid-cols-2 gap-3">{data.tracks.filter(t => !t.name.includes('(historical)')).map(track =>
        <div key={track.key} className="rounded-lg border border-zinc-800 p-4">
          <h3 style={{ color: track.color }}>{track.name}</h3>
          <p>{track.current}h {track.target !== undefined && <span className="text-zinc-400">/ {track.target}h planned</span>}</p>
          {track.target !== undefined && track.target > 0 && <progress aria-label={`${track.name} weekly target`} className="w-full" value={track.current} max={track.target} />}
        </div>)}</div>
    </section>
    <section className="space-y-3"><h2 className="text-xl font-semibold">Study hours over time</h2>
      <p className="text-sm text-zinc-400">Historical categories retain their original hours. Future templates are excluded.</p>
      <div className="h-80"><Bar data={{ labels: data.weeks.map(w => `W${w.week}`), datasets: data.tracks.map(t => ({ label: t.name, data: t.values, backgroundColor: t.color })) }} options={{ responsive: true, maintainAspectRatio: false, scales: { x: { stacked: true }, y: { stacked: true, beginAtZero: true } }, plugins: { legend: { labels: { color: '#a1a1aa' } } } }} /></div>
    </section>
    <section className="space-y-3"><h2 className="text-xl font-semibold">Weekly history</h2>
      <div className="overflow-x-auto"><table className="w-full text-sm text-left"><thead><tr><th className="p-2">Week</th><th className="p-2">Dates</th><th className="p-2">Hours</th></tr></thead>
        <tbody>{[...data.weeks].reverse().map(w => <tr key={w.slug} className="border-t border-zinc-800"><td className="p-2">{w.week}</td><td className="p-2">{w.dates}</td><td className="p-2">{Object.values(w.hours).reduce((sum, h) => sum + h, 0)}</td></tr>)}</tbody></table></div>
    </section>
  </div>;
}
