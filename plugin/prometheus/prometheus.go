package prometheus

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/DrmagicE/gmqtt/config"
	"github.com/DrmagicE/gmqtt/persistence/subscription"
	"github.com/DrmagicE/gmqtt/server"
)

var _ server.Plugin = (*Prometheus)(nil)

const (
	Name         = "prometheus"
	metricPrefix = "gmqtt_"
)

func init() {
	server.RegisterPlugin(Name, New)
	config.RegisterDefaultPluginConfig(Name, &DefaultConfig)
}

func New(config config.Config) (server.Plugin, error) {
	cfg := config.Plugins[Name].(*Config)
	httpServer := &http.Server{
		Addr: cfg.ListenAddress,
	}
	return &Prometheus{
		httpServer: httpServer,
		path:       cfg.Path,
	}, nil
}

var log *zap.Logger

const dashboardPageTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<title>GMQTT 指标面板</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
:root { color-scheme: dark; font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif; background: #0b1526; color: #e5e7eb; }
body { margin: 0; background: #0b1526; color: #e5e7eb; }
header { padding: 18px 20px; background: rgba(15, 23, 42, 0.9); border-bottom: 1px solid rgba(148,163,184,.15); position: sticky; top: 0; }
header h1 { margin: 0; font-size: 16px; font-weight: 600; }
header p { margin: 6px 0 0; color: #94a3b8; font-size: 12px; }
main { padding: 16px 20px 28px; display: grid; gap: 14px; }
.bar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 4px; }
.pill { font-size: 12px; padding: 4px 10px; border-radius: 999px; background: rgba(59,130,246,.18); color: #bfdbfe; border: 1px solid rgba(59,130,246,.35); }
.status { font-size: 12px; color: #94a3b8; min-height: 18px; }
.status.error { color: #fca5a5; }
.status.success { color: #86efac; }
.status.loading { color: #93c5fd; }
.grid { display: grid; gap: 12px; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); }
.card { background: rgba(15, 23, 42, 0.65); border: 1px solid rgba(148,163,184,.12); border-radius: 10px; padding: 14px; }
.card h3 { margin: 0 0 10px 0; font-size: 14px; color: #e2e8f0; }
.kv { display: grid; grid-template-columns: 1fr auto; gap: 6px 10px; font-size: 13px; }
.kv dt { color: #94a3b8; }
.kv dd { margin: 0; font-variant-numeric: tabular-nums; color: #f8fafc; }
.panel { background: rgba(13, 23, 39, 0.75); border: 1px solid rgba(59,130,246,.15); border-radius: 10px; padding: 14px; }
.panel h2 { margin: 0 0 10px 0; font-size: 14px; color: #e2e8f0; }
table { width: 100%; border-collapse: collapse; font-size: 12px; border-radius: 8px; overflow: hidden; }
th, td { padding: 8px 10px; border-bottom: 1px solid rgba(148,163,184,.12); text-align: left; font-variant-numeric: tabular-nums; }
thead th { background: rgba(30,41,59,.6); color: #cbd5e1; }
tbody tr:hover { background: rgba(30,41,59,.3); }
details { background: rgba(15,23,42,.6); border: 1px solid rgba(148,163,184,.12); border-radius: 8px; }
details summary { cursor: pointer; padding: 10px 12px; color: #cbd5e1; }
pre { margin: 0; background: #0f172a; padding: 12px; border-top: 1px solid rgba(148,163,184,.12); color: #cbd5e1; font-size: 12px; line-height: 1.5; overflow-x: auto; }
code { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono","Courier New", monospace; }
</style>
</head>
<body>
<header>
	<h1>GMQTT 指标面板</h1>
	<p>解析并分组展示 Prometheus 指标 · 源 <code>__METRICS_PATH__</code> · 每 5 秒刷新</p>
</header>
<main>
	<div class="bar">
		<span id="status" class="status">等待刷新...</span>
		<span class="pill">GET <code>__METRICS_PATH__</code></span>
	</div>
	<section class="grid">
		<article class="card">
			<h3>GMQTT 概览</h3>
			<dl class="kv">
				<dt>历史连接总数</dt><dd id="m_clients_connected">--</dd>
				<dt>累计断开总数</dt><dd id="m_clients_disconnected">--</dd>
				<dt>活跃会话</dt><dd id="m_sessions_active">--</dd>
				<dt>非活跃会话</dt><dd id="m_sessions_inactive">--</dd>
				<dt>当前订阅</dt><dd id="m_sub_current">--</dd>
				<dt>累计订阅</dt><dd id="m_sub_total">--</dd>
				<dt>队列消息</dt><dd id="m_msg_queued">--</dd>
				<dt>消息丢弃(总)</dt><dd id="m_msg_dropped">--</dd>
			</dl>
		</article>
		<article class="card">
			<h3>运行时 / 进程</h3>
			<dl class="kv">
				<dt>Go 版本</dt><dd id="m_go_version">--</dd>
				<dt>Goroutines</dt><dd id="m_go_goroutines">--</dd>
				<dt>OS 线程</dt><dd id="m_go_threads">--</dd>
				<dt>CPU 秒</dt><dd id="m_proc_cpu">--</dd>
				<dt>常驻内存</dt><dd id="m_proc_rss">--</dd>
				<dt>打开 FD</dt><dd id="m_proc_fds">--</dd>
			</dl>
		</article>
		<article class="card">
			<h3>PromHTTP</h3>
			<dl class="kv" id="promhttp-overview">
				<dt>200</dt><dd id="m_ph_200">--</dd>
				<dt>500</dt><dd id="m_ph_500">--</dd>
				<dt>503</dt><dd id="m_ph_503">--</dd>
				<dt>并发抓取</dt><dd id="m_ph_inflight">--</dd>
			</dl>
		</article>
	</section>

	<section class="panel">
		<h2>GMQTT 吞吐（按 QoS）</h2>
		<div class="status" id="status-qos"></div>
		<div class="table-wrapper">
			<table>
				<thead>
					<tr>
						<th>QoS</th>
						<th>接收总数</th>
						<th>发送总数</th>
						<th>丢弃总数</th>
					</tr>
				</thead>
				<tbody id="qos-body"></tbody>
			</table>
		</div>
	</section>

	<section class="panel">
		<h2>MQTT 报文统计</h2>
		<div class="table-wrapper">
			<table>
				<thead>
					<tr>
						<th>类型</th>
						<th>接收次数</th>
						<th>接收字节</th>
						<th>发送次数</th>
						<th>发送字节</th>
					</tr>
				</thead>
				<tbody id="pkt-body"></tbody>
			</table>
		</div>
	</section>

	<details>
		<summary>查看原始指标文本</summary>
		<pre id="raw-output">// 正在加载...</pre>
	</details>
</main>
<script>
const metricsPath = "__METRICS_PATH__";
const statusEl = document.getElementById("status");

function formatNumber(value, digits) {
	if (value === null || value === undefined || Number.isNaN(value)) return "--";
	if (digits !== undefined) return Number(value).toLocaleString(undefined, { maximumFractionDigits: digits });
	return Number(value).toLocaleString();
}
function formatBytesToMB(value) {
	if (value === null || value === undefined || Number.isNaN(value)) return "--";
	return (Number(value) / 1024 / 1024).toLocaleString(undefined, { maximumFractionDigits: 2 }) + " MB";
}
function parseMetrics(text) {
	const regex = /^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{([^}]*)\})?\s+([0-9.eE+-]+)$/;
	const metrics = {};
	text.split("\n").forEach(line => {
		const trimmed = line.trim();
		if (!trimmed || trimmed.startsWith("#")) return;
		const match = trimmed.match(regex);
		if (!match) return;
		const name = match[1];
		const labelsRaw = match[3];
		const value = parseFloat(match[4]);
		const entry = { labels: {}, value };
		if (labelsRaw) {
			labelsRaw.split(",").forEach(pair => {
				const idx = pair.indexOf("=");
				if (idx === -1) return;
				const key = pair.slice(0, idx).trim();
				const rawValue = pair.slice(idx + 1).trim();
				entry.labels[key] = rawValue.replace(/^"|"$/g, "");
			});
		}
		if (!metrics[name]) metrics[name] = [];
		metrics[name].push(entry);
	});
	return metrics;
}
function getMetric(metrics, name, labels) {
	const list = metrics[name] || [];
	if (!labels) return list.length ? list[0].value : null;
	for (const item of list) {
		let ok = true;
		for (const k in labels) {
			if (item.labels[k] !== labels[k]) { ok = false; break; }
		}
		if (ok) return item.value;
	}
	return null;
}
function sumMetric(metrics, name, filter) {
	const list = metrics[name] || [];
	return list.filter(it => (filter ? filter(it.labels) : true))
		.reduce((acc, cur) => acc + (Number.isFinite(cur.value) ? cur.value : 0), 0);
}
function renderOverview(m) {
	document.getElementById("m_clients_connected").textContent = formatNumber(getMetric(m, "gmqtt_clients_connected_total"));
	document.getElementById("m_clients_disconnected").textContent = formatNumber(getMetric(m, "gmqtt_clients_disconnected_total"));
	document.getElementById("m_sessions_active").textContent = formatNumber(getMetric(m, "gmqtt_sessions_active_current"));
	document.getElementById("m_sessions_inactive").textContent = formatNumber(getMetric(m, "gmqtt_sessions_inactive_current"));
	document.getElementById("m_sub_current").textContent = formatNumber(getMetric(m, "gmqtt_subscriptions_current"));
	document.getElementById("m_sub_total").textContent = formatNumber(getMetric(m, "gmqtt_subscriptions_total"));
	document.getElementById("m_msg_queued").textContent = formatNumber(getMetric(m, "gmqtt_messages_queued_current"));
	document.getElementById("m_msg_dropped").textContent = formatNumber(sumMetric(m, "gmqtt_messages_dropped_total"));

	const goInfo = (m["go_info"] || [])[0];
	document.getElementById("m_go_version").textContent = goInfo ? (goInfo.labels.version || "--") : "--";
	document.getElementById("m_go_goroutines").textContent = formatNumber(getMetric(m, "go_goroutines"));
	document.getElementById("m_go_threads").textContent = formatNumber(getMetric(m, "go_threads"));
	document.getElementById("m_proc_cpu").textContent = formatNumber(getMetric(m, "process_cpu_seconds_total"), 3);
	document.getElementById("m_proc_rss").textContent = formatBytesToMB(getMetric(m, "process_resident_memory_bytes"));
	document.getElementById("m_proc_fds").textContent = formatNumber(getMetric(m, "process_open_fds"));

	document.getElementById("m_ph_inflight").textContent = formatNumber(getMetric(m, "promhttp_metric_handler_requests_in_flight"));
	document.getElementById("m_ph_200").textContent = formatNumber(getMetric(m, "promhttp_metric_handler_requests_total", { code: "200" }));
	document.getElementById("m_ph_500").textContent = formatNumber(getMetric(m, "promhttp_metric_handler_requests_total", { code: "500" }));
	document.getElementById("m_ph_503").textContent = formatNumber(getMetric(m, "promhttp_metric_handler_requests_total", { code: "503" }));
}
function renderQos(m) {
	const qosBody = document.getElementById("qos-body");
	const rows = ["0", "1", "2"].map(qos => {
		const recv = getMetric(m, "gmqtt_messages_received_total", { qos });
		const sent = getMetric(m, "gmqtt_messages_sent_total", { qos });
		const drop = sumMetric(m, "gmqtt_messages_dropped_total", lab => lab.qos === qos);
		return "<tr><td>QoS " + qos + "</td><td>" + formatNumber(recv) + "</td><td>" + formatNumber(sent) + "</td><td>" + formatNumber(drop) + "</td></tr>";
	});
	qosBody.innerHTML = rows.join("");
	document.getElementById("status-qos").textContent = "更新于 " + new Date().toLocaleTimeString();
}
function renderPackets(m) {
	const recv = m["gmqtt_packets_received_total"] || [];
	const recvBytes = m["gmqtt_packets_received_bytes_total"] || [];
	const sent = m["gmqtt_packets_sent_total"] || [];
	const sentBytes = m["gmqtt_packets_sent_bytes_total"] || [];
	const typeSet = new Set();
	[recv, recvBytes, sent, sentBytes].forEach(list => list.forEach(it => typeSet.add(it.labels.type)));
	const types = Array.from(typeSet).sort();
	const body = document.getElementById("pkt-body");
	const rows = types.map(t => {
		const r = getMetric(m, "gmqtt_packets_received_total", { type: t });
		const rb = getMetric(m, "gmqtt_packets_received_bytes_total", { type: t });
		const s = getMetric(m, "gmqtt_packets_sent_total", { type: t });
		const sb = getMetric(m, "gmqtt_packets_sent_bytes_total", { type: t });
		return "<tr><td>" + t + "</td><td>" + formatNumber(r) + "</td><td>" + formatNumber(rb) + "</td><td>" + formatNumber(s) + "</td><td>" + formatNumber(sb) + "</td></tr>";
	});
	body.innerHTML = rows.join("");
}

async function load() {
	statusEl.className = "status loading";
	statusEl.textContent = "正在获取最新数据...";
	try {
		const res = await fetch(metricsPath, { cache: "no-store" });
		if (!res.ok) throw new Error("HTTP " + res.status);
		const text = await res.text();
		document.getElementById("raw-output").textContent = text;
		const metrics = parseMetrics(text);
		renderOverview(metrics);
		renderQos(metrics);
		renderPackets(metrics);
		statusEl.className = "status success";
		statusEl.textContent = "获取成功 · " + new Date().toLocaleTimeString();
	} catch (err) {
		statusEl.className = "status error";
		statusEl.textContent = "获取失败：" + err.message;
	}
}
load();
setInterval(load, 5000);
</script>
</body>
</html>`

// Prometheus served as a prometheus exporter that exposes gmqtt metrics.
type Prometheus struct {
	statsManager server.StatsReader
	httpServer   *http.Server
	path         string
}

func (p *Prometheus) Load(service server.Server) error {
	log = server.LoggerWithField(zap.String("plugin", Name))
	p.statsManager = service.StatsManager()
	r := prometheus.DefaultRegisterer
	r.MustRegister(p)
	mu := http.NewServeMux()
	mu.Handle(p.path, promhttp.Handler())
	mu.Handle("/", http.HandlerFunc(p.dashboardHandler))
	p.httpServer.Handler = mu
	go func() {
		err := p.httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			panic(err.Error())
		}
	}()
	return nil
}

func (p *Prometheus) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page := strings.ReplaceAll(dashboardPageTemplate, "__METRICS_PATH__", p.path)
	_, _ = w.Write([]byte(page))
}

func (p *Prometheus) Unload() error {
	return p.httpServer.Shutdown(context.Background())
}

func (p *Prometheus) Name() string {
	return Name
}

func (p *Prometheus) Describe(desc chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(p, desc)
}

func (p *Prometheus) Collect(m chan<- prometheus.Metric) {
	log.Debug("metrics collected")
	st := p.statsManager.GetGlobalStats()
	collectPacketsStats(&st.PacketStats, m)
	collectClientStats(&st.ConnectionStats, m)
	collectSubscriptionStats(&st.SubscriptionStats, m)
	collectMessageStats(&st.MessageStats, m)
}

func collectPacketsStats(ps *server.PacketStats, m chan<- prometheus.Metric) {
	bytesReceivedMetricName := metricPrefix + "packets_received_bytes_total"
	ReceivedCounterMetricName := metricPrefix + "packets_received_total"
	bytesSentMetricName := metricPrefix + "packets_sent_bytes_total"
	sentCounterMetricName := metricPrefix + "packets_sent_total"

	collectPacketsStatsBytes(bytesReceivedMetricName, &ps.BytesReceived, m)
	collectPacketsStatsBytes(bytesSentMetricName, &ps.BytesSent, m)

	collectPacketsStatsCounter(ReceivedCounterMetricName, &ps.ReceivedTotal, m)
	collectPacketsStatsCounter(sentCounterMetricName, &ps.SentTotal, m)
}
func collectPacketsStatsBytes(metricName string, pb *server.PacketBytes, m chan<- prometheus.Metric) {
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Connect)),
		"CONNECT",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Connack)),
		"CONNACK",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Disconnect)),
		"DISCONNECT",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Pingreq)),
		"PINGREQ",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Pingresp)),
		"PINGRESP",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Puback)),
		"PUBACK",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Pubcomp)),
		"PUBCOMP",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Publish)),
		"PUBLISH",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Pubrec)),
		"PUBREC",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Pubrel)),
		"PUBREL",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Suback)),
		"SUBACK",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Subscribe)),
		"SUBSCRIBE",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Unsuback)),
		"UNSUBACK",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pb.Unsubscribe)),
		"UNSUBSCRIBE",
	)
}
func collectPacketsStatsCounter(metricName string, pc *server.PacketCount, m chan<- prometheus.Metric) {
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Connect)),
		"CONNECT",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Connack)),
		"CONNACK",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Disconnect)),
		"DISCONNECT",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Pingreq)),
		"PINGREQ",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Pingresp)),
		"PINGRESP",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Puback)),
		"PUBACK",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Pubcomp)),
		"PUBCOMP",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Publish)),
		"PUBLISH",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Pubrec)),
		"PUBREC",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Pubrel)),
		"PUBREL",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Suback)),
		"SUBACK",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Subscribe)),
		"SUBSCRIBE",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Unsuback)),
		"UNSUBACK",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&pc.Unsubscribe)),
		"UNSUBSCRIBE",
	)
}

func collectClientStats(c *server.ConnectionStats, m chan<- prometheus.Metric) {
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricPrefix+"clients_connected_total", "", nil, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&c.ConnectedTotal)),
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricPrefix+"sessions_created_total", "", nil, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&c.SessionCreatedTotal)),
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricPrefix+"sessions_terminated_total", "", []string{"reason"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&c.SessionTerminated.Expired)), "expired",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricPrefix+"sessions_terminated_total", "", []string{"reason"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&c.SessionTerminated.TakenOver)), "taken_over",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricPrefix+"sessions_terminated_total", "", []string{"reason"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&c.SessionTerminated.Normal)), "normal",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricPrefix+"sessions_active_current", "", nil, nil),
		prometheus.GaugeValue,
		float64(atomic.LoadUint64(&c.ActiveCurrent)),
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricPrefix+"sessions_inactive_current", "", nil, nil),
		prometheus.GaugeValue,
		float64(atomic.LoadUint64(&c.InactiveCurrent)),
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricPrefix+"clients_disconnected_total", "", nil, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&c.DisconnectedTotal)),
	)
}
func collectMessageStats(ms *server.MessageStats, m chan<- prometheus.Metric) {
	collectMessageStatsDropped(ms, m)
	collectMessageStatsQueued(ms, m)
	collectMessageStatsReceived(ms, m)
	collectMessageStatsSent(ms, m)
}

func collectQoSDropped(metricName string, qos string, stats *server.MessageQosStats, m chan<- prometheus.Metric) {
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"qos", "type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&stats.DroppedTotal.Internal)), qos, "internal",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"qos", "type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&stats.DroppedTotal.Expired)), qos, "expired",
	)

	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"qos", "type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&stats.DroppedTotal.QueueFull)), qos, "queue_full",
	)

	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"qos", "type"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&stats.DroppedTotal.ExceedsMaxPacketSize)), qos, "exceeds_max_size",
	)
}

func collectMessageStatsDropped(ms *server.MessageStats, m chan<- prometheus.Metric) {
	metricName := metricPrefix + "messages_dropped_total"
	collectQoSDropped(metricName, "0", &ms.Qos0, m)
	collectQoSDropped(metricName, "1", &ms.Qos1, m)
	collectQoSDropped(metricName, "2", &ms.Qos2, m)
}

func collectMessageStatsQueued(ms *server.MessageStats, m chan<- prometheus.Metric) {
	metricName := metricPrefix + "messages_queued_current"
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", nil, nil),
		prometheus.GaugeValue,
		float64(atomic.LoadUint64(&ms.QueuedCurrent)),
	)
}
func collectMessageStatsReceived(ms *server.MessageStats, m chan<- prometheus.Metric) {
	metricName := metricPrefix + "messages_received_total"
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"qos"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&ms.Qos0.ReceivedTotal)), "0",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"qos"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&ms.Qos1.ReceivedTotal)), "1",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"qos"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&ms.Qos2.ReceivedTotal)), "2",
	)
}
func collectMessageStatsSent(ms *server.MessageStats, m chan<- prometheus.Metric) {
	metricName := metricPrefix + "messages_sent_total"
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"qos"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&ms.Qos0.SentTotal)), "0",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"qos"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&ms.Qos1.SentTotal)), "1",
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricName, "", []string{"qos"}, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&ms.Qos2.SentTotal)), "2",
	)
}

func collectSubscriptionStats(s *subscription.Stats, m chan<- prometheus.Metric) {
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricPrefix+"subscriptions_total", "", nil, nil),
		prometheus.CounterValue,
		float64(atomic.LoadUint64(&s.SubscriptionsTotal)),
	)
	m <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(metricPrefix+"subscriptions_current", "", nil, nil),
		prometheus.GaugeValue,
		float64(atomic.LoadUint64(&s.SubscriptionsCurrent)),
	)
}
