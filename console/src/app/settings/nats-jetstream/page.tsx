"use client";
import { useState, useEffect, useCallback } from "react";
import { Radio, Server, Users } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Stream { name: string; subjects: string[]; msgs: number; bytes: number; consumer_count: number; last_msg: string; }
interface Consumer { name: string; stream: string; delivered: number; ack_floor: number; pending: number; status: string; }
interface JetStreamData { streams: Stream[]; consumers: Consumer[]; connections: number; total_msgs: number; total_bytes: number; throughput_per_sec: number; status: string; }

export default function NatsJetstreamPage() {
  const t = useTranslations();

  const [data, setData] = useState<JetStreamData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/admin/nats-health", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  if (!data) return <p className="text-sm text-gray-500 text-center py-8">Loading...</p>;

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Radio className="w-6 h-6 text-blue-500" /> {t("natsJetstream.title")}</h1><p className="text-sm text-gray-500 mt-1">Monitor streams, consumers, and messaging throughput.</p></div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Server className="w-8 h-8 text-blue-500" /><div><span className="text-sm text-gray-500">Streams</span><p className="text-xl font-bold">{data.streams.length}</p></div></div>
        <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Users className="w-8 h-8 text-purple-500" /><div><span className="text-sm text-gray-500">Consumers</span><p className="text-xl font-bold">{data.consumers.length}</p></div></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Messages</span><p className="text-xl font-bold mt-1">{data.total_msgs.toLocaleString()}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Throughput</span><p className="text-xl font-bold mt-1 text-blue-600">{data.throughput_per_sec}/s</p></div>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Stream</th><th className="px-4 py-3 text-left font-medium">Subjects</th><th className="px-4 py-3 text-left font-medium">Messages</th><th className="px-4 py-3 text-left font-medium">Size</th><th className="px-4 py-3 text-left font-medium">Consumers</th><th className="px-4 py-3 text-left font-medium">Last Msg</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{data.streams.map((s) => (<tr key={s.name} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs font-medium">{s.name}</td><td className="px-4 py-3"><div className="flex flex-wrap gap-1">{s.subjects.slice(0, 3).map((sub) => (<span key={sub} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{sub}</span>))}{s.subjects.length > 3 && <span className="text-xs text-gray-400">+{s.subjects.length - 3}</span>}</div></td><td className="px-4 py-3 font-bold">{s.msgs.toLocaleString()}</td><td className="px-4 py-3 text-xs text-gray-500">{(s.bytes / 1024 / 1024).toFixed(1)} MB</td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-purple-100 dark:bg-purple-900/30 dark:text-purple-400">{s.consumer_count}</span></td><td className="px-4 py-3 text-xs text-gray-400">{s.last_msg}</td></tr>))}</tbody></table></div>

      {data.consumers.length > 0 && (<div className="overflow-x-auto rounded-lg border dark:border-gray-800"><h3 className="text-sm font-semibold px-4 py-3 border-b dark:border-gray-800">Consumers</h3><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Name</th><th className="px-4 py-3 text-left font-medium">Stream</th><th className="px-4 py-3 text-left font-medium">Delivered</th><th className="px-4 py-3 text-left font-medium">Pending</th><th className="px-4 py-3 text-left font-medium">Status</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{data.consumers.map((c, i) => (<tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs">{c.name}</td><td className="px-4 py-3 text-xs text-gray-500">{c.stream}</td><td className="px-4 py-3 font-bold">{c.delivered.toLocaleString()}</td><td className="px-4 py-3"><span className={"text-xs font-bold " + (c.pending > 100 ? "text-red-600" : "text-gray-500")}>{c.pending}</span></td><td className="px-4 py-3"><span className="text-xs text-green-600">{c.status}</span></td></tr>))}</tbody></table></div>)}
    </div>
  );
}
