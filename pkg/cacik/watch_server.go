package cacik

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// ── Event types ──────────────────────────────────────────────────────────────

// WatchEventType identifies the kind of SSE event.
type WatchEventType string

const (
	EventRunStarted        WatchEventType = "run_started"
	EventScenarioStarted   WatchEventType = "scenario_started"
	EventStepCompleted     WatchEventType = "step_completed"
	EventScenarioCompleted WatchEventType = "scenario_completed"
	EventRunCompleted      WatchEventType = "run_completed"
)

// RunStartedEvent is the payload for EventRunStarted.
type RunStartedEvent struct {
	Total     int                     `json:"total"`
	Scenarios []ScenarioMetadataEvent `json:"scenarios"`
}

// ScenarioMetadataEvent describes a single scenario in the run_started payload.
type ScenarioMetadataEvent struct {
	Index   int      `json:"index"`
	Name    string   `json:"name"`
	Feature string   `json:"feature"`
	Rule    string   `json:"rule"`
	Tags    []string `json:"tags"`
}

// ScenarioStartedEvent is the payload for EventScenarioStarted.
type ScenarioStartedEvent struct {
	Index int `json:"index"`
}

// StepCompletedEvent is the payload for EventStepCompleted.
type StepCompletedEvent struct {
	ScenarioIndex int    `json:"scenario_index"`
	StepCategory  string `json:"step_category"` // "feature_bg", "rule_bg", "scenario"
	StepIndex     int    `json:"step_index"`
	Keyword       string `json:"keyword"`
	Text          string `json:"text"`
	Status        string `json:"status"` // "passed", "failed", "skipped"
	Error         string `json:"error"`
	DurationMs    int64  `json:"duration_ms"`
	MatchLocs     []int  `json:"match_locs"`
}

// ScenarioCompletedEvent is the payload for EventScenarioCompleted.
type ScenarioCompletedEvent struct {
	Index      int    `json:"index"`
	Passed     bool   `json:"passed"`
	Error      string `json:"error"`
	DurationMs int64  `json:"duration_ms"`
}

// RunCompletedEvent is the payload for EventRunCompleted.
type RunCompletedEvent struct {
	DurationMs int64               `json:"duration_ms"`
	Summary    RunCompletedSummary `json:"summary"`
}

// RunCompletedSummary mirrors ReporterSummary for JSON serialization.
type RunCompletedSummary struct {
	ScenariosTotal  int `json:"scenarios_total"`
	ScenariosPassed int `json:"scenarios_passed"`
	ScenariosFailed int `json:"scenarios_failed"`
	StepsTotal      int `json:"steps_total"`
	StepsPassed     int `json:"steps_passed"`
	StepsFailed     int `json:"steps_failed"`
	StepsSkipped    int `json:"steps_skipped"`
}

// ── EventBroker ──────────────────────────────────────────────────────────────

// EventBroker fans out SSE events to all connected browser clients.
// It maintains a history buffer so late-connecting or reconnecting clients
// can replay all past events.
type EventBroker struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
	history [][]byte
	closed  bool
}

// NewEventBroker creates a ready-to-use broker.
func NewEventBroker() *EventBroker {
	return &EventBroker{
		clients: make(map[chan []byte]struct{}),
	}
}

// Publish marshals the event and sends it to every connected client.
// The raw JSON bytes are also appended to the history buffer.
func (b *EventBroker) Publish(eventType WatchEventType, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	// SSE format: "event: <type>\ndata: <json>\n\n"
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, payload)
	raw := []byte(msg)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.history = append(b.history, raw)

	for ch := range b.clients {
		select {
		case ch <- raw:
		default:
			// slow client — drop the message rather than blocking
		}
	}
}

// Subscribe registers a new SSE client and replays the full event history.
// The returned channel receives pre-formatted SSE messages.
func (b *EventBroker) Subscribe() chan []byte {
	ch := make(chan []byte, 256)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Replay history so the client catches up.
	for _, msg := range b.history {
		ch <- msg
	}

	b.clients[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a client and closes its channel.
// It is safe to call after Close — if the channel was already removed
// (and closed) by Close, this is a no-op.
func (b *EventBroker) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.clients[ch]; ok {
		delete(b.clients, ch)
		close(ch)
	}
}

// Close marks the broker as closed and closes all client channels.
func (b *EventBroker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	for ch := range b.clients {
		close(ch)
		delete(b.clients, ch)
	}
}

// ── WatchServer ──────────────────────────────────────────────────────────────

// WatchServer serves the live HTML page and SSE event stream.
type WatchServer struct {
	broker   *EventBroker
	server   *http.Server
	listener net.Listener
	addr     string // "http://localhost:<port>"
}

// StartWatchServer starts the HTTP server on an auto-selected free port.
// It returns immediately; the server runs in a background goroutine.
func StartWatchServer(broker *EventBroker) (*WatchServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("could not start watch server: %w", err)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("http://localhost:%d", port)

	mux := http.NewServeMux()
	ws := &WatchServer{
		broker:   broker,
		listener: ln,
		addr:     addr,
		server:   &http.Server{Handler: mux},
	}

	mux.HandleFunc("/", ws.handleIndex)
	mux.HandleFunc("/events", ws.handleSSE)

	go func() {
		_ = ws.server.Serve(ln)
	}()

	return ws, nil
}

// Addr returns the full URL the server is listening on.
func (ws *WatchServer) Addr() string {
	return ws.addr
}

// Shutdown gracefully stops the server with a timeout.
func (ws *WatchServer) Shutdown(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_ = ws.server.Shutdown(ctx)
}

// handleIndex serves the live HTML page.
func (ws *WatchServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(liveHTMLPage))
}

// handleSSE streams events to the browser via Server-Sent Events.
func (ws *WatchServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher.Flush() // send headers immediately so clients don't hang

	ch := ws.broker.Subscribe()
	defer ws.broker.Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			_, _ = w.Write(msg)
			flusher.Flush()
		}
	}
}

// OpenBrowser opens the given URL in the user's default browser.
// It is fire-and-forget: errors are silently ignored (e.g. in CI).
func OpenBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// ── Live HTML page ───────────────────────────────────────────────────────────

// liveHTMLPage is the self-contained HTML page served by the watch server.
// It connects to /events via SSE and updates the DOM as events arrive.
// The visual design matches the static HTML report exactly.
const liveHTMLPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Live Test Watch</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen,
                 Ubuntu, Cantarell, "Fira Sans", "Droid Sans", "Helvetica Neue", sans-serif;
    background: #f8f9fa; color: #212529; line-height: 1.6; padding: 2rem;
  }
  h1 { font-size: 1.5rem; margin-bottom: 0.25rem; color: #212529; font-weight: 700; }
  .status-line {
    font-size: 0.8rem; color: #868e96; margin-bottom: 1.5rem;
    display: flex; align-items: center; gap: 0.5rem;
  }
  .status-dot {
    width: 8px; height: 8px; border-radius: 50%; display: inline-block;
  }
  .status-dot.running { background: #228be6; animation: pulse 1s infinite; }
  .status-dot.complete { background: #2b8a3e; }
  .status-dot.has-failures { background: #c92a2a; }
  @keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.4; } }

  /* ---- Summary dashboard ---- */
  .summary {
    display: flex; gap: 1rem; flex-wrap: wrap;
    margin-bottom: 2rem; padding: 1rem 1.25rem; background: #fff;
    border-radius: 10px; border: 1px solid #e9ecef;
    box-shadow: 0 1px 3px rgba(0,0,0,0.04);
  }
  .summary.all-passed { border: 2px solid #2b8a3e; background: #f6fef7; }
  .summary.has-failures { border: 2px solid #c92a2a; background: #fff5f5; }
  .summary-item { text-align: center; min-width: 90px; }
  .summary-item .number { font-size: 1.8rem; font-weight: 700; }
  .summary-item .label {
    font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em; color: #868e96;
  }
  .number.green  { color: #2b8a3e; }
  .number.red    { color: #c92a2a; }
  .number.yellow { color: #e67700; }
  .number.blue   { color: #1864ab; }

  /* ---- Filter bar ---- */
  .filter-bar {
    display: none; gap: 0.4rem; flex-wrap: wrap; align-items: center;
    margin-bottom: 0.75rem; padding: 0.75rem 1rem;
    background: #fff; border-radius: 8px; border: 1px solid #e9ecef;
    box-shadow: 0 1px 3px rgba(0,0,0,0.04);
  }
  .filter-bar.visible { display: flex; }
  .filter-label { font-size: 0.8rem; font-weight: 600; color: #495057; margin-right: 0.25rem; }
  .filter-btn {
    background: #fff; border: 1px solid #dee2e6; border-radius: 6px;
    padding: 0.25rem 0.6rem; font-size: 0.72rem; color: #495057;
    cursor: pointer; font-weight: 500; transition: background 0.15s;
  }
  .filter-btn:hover { background: #f1f3f5; }
  .filter-tag {
    border: 1px solid #dee2e6; border-radius: 12px;
    padding: 0.2rem 0.6rem; font-size: 0.72rem; cursor: pointer;
    font-weight: 500; transition: all 0.15s;
  }
  .filter-tag.active { background: #228be6; color: #fff; border-color: #228be6; }
  .filter-tag.inactive { background: #fff; color: #adb5bd; border-color: #dee2e6; }
  .filter-tag:hover { opacity: 0.85; }

  /* ---- Toggle bar ---- */
  .toggle-bar { display: flex; gap: 0.5rem; margin-bottom: 1rem; }
  .toggle-btn {
    background: #fff; border: 1px solid #dee2e6; border-radius: 6px;
    padding: 0.3rem 0.75rem; font-size: 0.75rem; color: #495057;
    cursor: pointer; font-weight: 500; transition: background 0.15s, border-color 0.15s;
  }
  .toggle-btn:hover { background: #f1f3f5; border-color: #adb5bd; }

  /* ---- Section ---- */
  .section { margin-bottom: 2rem; }
  .section-header {
    font-size: 1.1rem; font-weight: 700; margin-bottom: 0.75rem;
    padding-bottom: 0.4rem; border-bottom: 2px solid #dee2e6;
    display: flex; align-items: center; gap: 0.5rem;
  }
  .section-header .dot { width: 10px; height: 10px; border-radius: 50%; display: inline-block; }
  .section-header .section-meta { font-size: 0.8rem; font-weight: 500; color: #868e96; }
  .section.failed .section-header { color: #c92a2a; }
  .section.failed .dot { background: #c92a2a; }
  .section.passed .section-header { color: #2b8a3e; }
  .section.passed .dot { background: #2b8a3e; }
  .section.pending .section-header { color: #868e96; }
  .section.pending .dot { background: #adb5bd; }
  .section.running .section-header { color: #1864ab; }
  .section.running .dot { background: #228be6; }
  .section-toggle {
    background: none; border: none; font-size: 0.75rem; color: #868e96;
    cursor: pointer; padding: 0; margin-left: 0.5rem; font-weight: 500;
  }
  .section-toggle:hover { color: #495057; }

  /* ---- Tag group ---- */
  .tag-group { margin-bottom: 1.25rem; margin-left: 0.25rem; }
  .tag-group-label {
    font-size: 0.8rem; font-weight: 600; color: #495057;
    margin-bottom: 0.4rem; padding-left: 0.25rem;
    display: flex; align-items: center; gap: 0.4rem;
  }
  .tag-group-label .tag-icon { color: #868e96; }
  .tag-group-meta { font-size: 0.75rem; font-weight: 400; color: #868e96; }
  .tag-group-toggle {
    background: none; border: none; font-size: 0.7rem; color: #868e96;
    cursor: pointer; padding: 0; margin-left: auto; font-weight: 500;
  }
  .tag-group-toggle:hover { color: #495057; }

  /* ---- Scenario card ---- */
  .scenario {
    margin-bottom: 0.5rem; background: #fff; border-radius: 8px;
    border-left: 4px solid #ced4da; overflow: hidden;
    border: 1px solid #e9ecef; box-shadow: 0 1px 2px rgba(0,0,0,0.03);
  }
  .scenario.passed { border-left: 4px solid #69db7c; }
  .scenario.failed { border-left: 4px solid #ff6b6b; }
  .scenario.running { border-left: 4px solid #228be6; }
  .scenario.pending { border-left: 4px solid #ced4da; }

  .scenario-header {
    display: flex; justify-content: space-between; align-items: center;
    padding: 0.6rem 1rem; cursor: pointer; user-select: none;
    transition: background 0.15s;
  }
  .scenario-header:hover { background: #f1f3f5; }
  .scenario-name { font-weight: 600; font-size: 0.9rem; color: #212529; }
  .scenario-meta {
    display: flex; gap: 0.75rem; align-items: center;
    font-size: 0.78rem; color: #868e96;
  }
  .tag { background: #e9ecef; border-radius: 4px; padding: 0.1rem 0.45rem; font-size: 0.68rem; color: #495057; font-weight: 500; }

  .steps {
    padding: 0.5rem 1rem 0.75rem 1rem; display: none;
    background: #1e1f22; border-radius: 0 0 6px 6px;
  }
  .scenario.open .steps { display: block; }

  .step {
    display: flex; align-items: baseline; gap: 0.5rem;
    padding: 0.2rem 0;
    font-family: "JetBrains Mono", "Fira Code", "Cascadia Code", "SF Mono", monospace;
    font-size: 0.82rem;
  }
  .step-symbol { width: 1.2rem; text-align: center; flex-shrink: 0; font-weight: 700; }
  .step-symbol.passed  { color: #32cd32; }
  .step-symbol.failed  { color: #ff4444; }
  .step-symbol.skipped { color: #e6b800; }
  .step-keyword { color: #CF8E6D; font-weight: 600; white-space: pre; }
  .step-text { color: #BCBEC4; }
  .step-text.skipped { color: #6F737A; }
  .step-keyword.skipped { color: #6F737A; }
  .step-param { font-weight: 600; }
  .step-duration { margin-left: auto; color: #6F737A; font-size: 0.72rem; white-space: nowrap; }
  .step-error {
    color: #ff4444; background: #2c1a1a; border-radius: 4px;
    padding: 0.3rem 0.5rem; margin: 0.15rem 0 0.15rem 1.7rem;
    font-size: 0.78rem; white-space: pre-wrap; border: 1px solid #4a2020;
  }

  .step-group-label {
    font-family: "JetBrains Mono", "Fira Code", "Cascadia Code", "SF Mono", monospace;
    font-size: 0.82rem; color: #BCBEC4; padding: 0.35rem 0 0.1rem 0;
  }
  .step-group-label.step-group-rule { margin-top: 0.6rem; }
  .step-group-label.step-group-scenario { margin-top: 0.6rem; }
  .step-group-kw { color: #CF8E6D; font-weight: 600; }

  .chevron { transition: transform 0.2s; font-size: 0.7rem; color: #adb5bd; }
  .scenario.open .chevron { transform: rotate(90deg); }

  .empty-msg { color: #868e96; font-style: italic; padding: 1rem 0; text-align: center; }

  @media (max-width: 600px) {
    body { padding: 0.75rem; }
    .summary { flex-direction: column; gap: 0.5rem; }
  }
</style>
</head>
<body>
<h1>Live Test Watch</h1>
<div class="status-line">
  <span id="status-dot" class="status-dot running"></span>
  <span id="status-text">Waiting for test data...</span>
</div>

<div id="summary" class="summary">
  <div class="summary-item"><div id="sum-scenarios" class="number blue">0</div><div class="label">Scenarios</div></div>
  <div class="summary-item"><div id="sum-passed" class="number green">0</div><div class="label">Passed</div></div>
  <div class="summary-item"><div id="sum-failed" class="number red">0</div><div class="label">Failed</div></div>
  <div class="summary-item"><div id="sum-steps" class="number blue">0</div><div class="label">Steps</div></div>
  <div class="summary-item"><div id="sum-steps-passed" class="number green">0</div><div class="label">Steps Passed</div></div>
  <div class="summary-item"><div id="sum-steps-failed" class="number red">0</div><div class="label">Steps Failed</div></div>
  <div class="summary-item"><div id="sum-steps-skipped" class="number yellow">0</div><div class="label">Steps Skipped</div></div>
  <div class="summary-item"><div id="sum-duration" class="number blue">-</div><div class="label">Duration</div></div>
</div>

<div id="filter-bar" class="filter-bar">
  <span class="filter-label">Filter by Tag</span>
  <button class="filter-btn" onclick="selectAllTags()">Select All</button>
  <button class="filter-btn" onclick="selectNoneTags()">Select None</button>
</div>

<div class="toggle-bar">
  <button class="toggle-btn" onclick="expandAll()">Expand All</button>
  <button class="toggle-btn" onclick="collapseAll()">Collapse All</button>
</div>

<div id="scenarios-container"></div>

<script>
(function() {
  // ---- State ----
  var scenarios = [];       // from run_started
  var summary = { scenariosPassed:0, scenariosFailed:0, stepsPassed:0, stepsFailed:0, stepsSkipped:0 };
  var completed = false;
  var paramColors = ['#5C92FF','#00CED1','#E5C07B','#C0A0FF','#98C379','#56B6C2','#E06C75','#D19A66','#7ECEA0','#A993D6'];

  // ---- SSE ----
  var es = new EventSource('/events');

  es.addEventListener('run_started', function(e) {
    var data = JSON.parse(e.data);
    scenarios = data.scenarios;
    document.getElementById('sum-scenarios').textContent = data.total;
    document.getElementById('status-text').textContent = 'Running ' + data.total + ' scenarios...';
    buildInitialDOM();
    buildFilterBar();
  });

  es.addEventListener('scenario_started', function(e) {
    var data = JSON.parse(e.data);
    var card = document.getElementById('scenario-' + data.index);
    if (card) {
      card.className = 'scenario running';
      card.classList.add('open');
    }
  });

  es.addEventListener('step_completed', function(e) {
    var data = JSON.parse(e.data);
    appendStep(data);
    // Update summary counters
    if (data.status === 'passed') { summary.stepsPassed++; }
    else if (data.status === 'failed') { summary.stepsFailed++; }
    else if (data.status === 'skipped') { summary.stepsSkipped++; }
    updateSummaryDOM();
  });

  es.addEventListener('scenario_completed', function(e) {
    var data = JSON.parse(e.data);
    var card = document.getElementById('scenario-' + data.index);
    if (card) {
      card.className = 'scenario ' + (data.passed ? 'passed' : 'failed');
      if (!data.passed) card.classList.add('open');
    }
    // Update duration
    var dur = card ? card.querySelector('.scenario-duration') : null;
    if (dur) dur.textContent = formatDuration(data.duration_ms);
    // Update summary
    if (data.passed) { summary.scenariosPassed++; } else { summary.scenariosFailed++; }
    updateSummaryDOM();
  });

  es.addEventListener('run_completed', function(e) {
    var data = JSON.parse(e.data);
    completed = true;
    es.close();

    // Final summary from server
    summary.scenariosPassed = data.summary.scenarios_passed;
    summary.scenariosFailed = data.summary.scenarios_failed;
    summary.stepsPassed = data.summary.steps_passed;
    summary.stepsFailed = data.summary.steps_failed;
    summary.stepsSkipped = data.summary.steps_skipped;
    updateSummaryDOM();

    document.getElementById('sum-duration').textContent = formatDuration(data.duration_ms);
    document.getElementById('sum-scenarios').textContent = data.summary.scenarios_total;
    document.getElementById('sum-steps').textContent = data.summary.steps_total;

    var dot = document.getElementById('status-dot');
    dot.classList.remove('running');
    if (data.summary.scenarios_failed > 0) {
      dot.classList.add('has-failures');
      document.getElementById('status-text').textContent = 'Complete - ' + data.summary.scenarios_failed + ' failure(s)';
      document.getElementById('summary').classList.add('has-failures');
    } else {
      dot.classList.add('complete');
      document.getElementById('status-text').textContent = 'Complete - all passed';
      document.getElementById('summary').classList.add('all-passed');
    }
  });

  es.onerror = function() {
    if (!completed) {
      document.getElementById('status-text').textContent = 'Connection lost. Refresh to reconnect.';
      var dot = document.getElementById('status-dot');
      dot.classList.remove('running');
      dot.style.background = '#868e96';
    }
  };

  // ---- DOM builders ----

  function buildInitialDOM() {
    var container = document.getElementById('scenarios-container');
    container.innerHTML = '';

    // Group by tag set for display
    var groups = groupByTags(scenarios);
    var groupKeys = Object.keys(groups).sort();
    // Move "Untagged" to end
    var utIdx = groupKeys.indexOf('Untagged');
    if (utIdx >= 0) { groupKeys.splice(utIdx, 1); groupKeys.push('Untagged'); }

    // Single section for all pending scenarios initially
    var section = document.createElement('div');
    section.className = 'section pending';
    section.id = 'main-section';
    section.innerHTML = '<div class="section-header"><span class="dot"></span> All Scenarios <span class="section-meta">' +
      scenarios.length + ' scenarios</span>' +
      '<button class="section-toggle" onclick="expandSection(this)">Expand All</button>' +
      '<button class="section-toggle" onclick="collapseSection(this)">Collapse All</button></div>';

    for (var gi = 0; gi < groupKeys.length; gi++) {
      var gk = groupKeys[gi];
      var gScenarios = groups[gk];
      var tg = document.createElement('div');
      tg.className = 'tag-group';
      tg.innerHTML = '<div class="tag-group-label"><span class="tag-icon">#</span> ' + escapeHtml(gk) +
        ' <span class="tag-group-meta">(' + gScenarios.length + ' scenarios)</span>' +
        '<button class="tag-group-toggle" onclick="expandGroup(this)">Expand</button>' +
        '<button class="tag-group-toggle" onclick="collapseGroup(this)">Collapse</button></div>';

      for (var si = 0; si < gScenarios.length; si++) {
        var s = gScenarios[si];
        tg.appendChild(buildScenarioCard(s));
      }
      section.appendChild(tg);
    }

    container.appendChild(section);
  }

  function buildScenarioCard(s) {
    var card = document.createElement('div');
    card.className = 'scenario pending';
    card.id = 'scenario-' + s.index;
    card.setAttribute('data-tags', (s.tags || []).join(','));

    var tagsHtml = '';
    if (s.tags) {
      for (var i = 0; i < s.tags.length; i++) {
        tagsHtml += '<span class="tag">' + escapeHtml(s.tags[i]) + '</span> ';
      }
    }

    card.innerHTML =
      '<div class="scenario-header" onclick="this.parentElement.classList.toggle(\'open\')">' +
        '<div>' +
          '<span class="scenario-name">' + escapeHtml(s.name) + '</span> ' + tagsHtml +
        '</div>' +
        '<div class="scenario-meta">' +
          '<span class="scenario-duration">-</span>' +
          '<span class="chevron">&#9654;</span>' +
        '</div>' +
      '</div>' +
      '<div class="steps" id="steps-' + s.index + '"></div>';

    return card;
  }

  function appendStep(data) {
    var stepsDiv = document.getElementById('steps-' + data.scenario_index);
    if (!stepsDiv) return;

    // Inject Feature: label as the very first element in the steps block
    var featureLabelId = 'label-' + data.scenario_index + '-feature';
    if (!document.getElementById(featureLabelId)) {
      var sInfo = scenarios[data.scenario_index];
      var fl = document.createElement('div');
      fl.id = featureLabelId;
      fl.className = 'step-group-label';
      fl.innerHTML = '<span class="step-group-kw">Feature:</span> ' + escapeHtml(sInfo ? sInfo.feature : '');
      stepsDiv.appendChild(fl);
    }

    // Add group label if this is the first step in a category
    var labelId = 'label-' + data.scenario_index + '-' + data.step_category;
    if (!document.getElementById(labelId) && data.step_index === 0) {
      // For rule_bg, inject Rule: label before the Background: label
      if (data.step_category === 'rule_bg') {
        var ruleLabelId = 'label-' + data.scenario_index + '-rule';
        if (!document.getElementById(ruleLabelId)) {
          var sInfo = scenarios[data.scenario_index];
          var rl = document.createElement('div');
          rl.id = ruleLabelId;
          rl.className = 'step-group-label step-group-rule';
          rl.innerHTML = '<span class="step-group-kw">Rule:</span> ' + escapeHtml(sInfo ? sInfo.rule : '');
          stepsDiv.appendChild(rl);
        }
      }

      var label = document.createElement('div');
      label.id = labelId;
      label.className = 'step-group-label';
      if (data.step_category === 'feature_bg') {
        label.innerHTML = '<span class="step-group-kw">Background:</span>';
      } else if (data.step_category === 'rule_bg') {
        label.innerHTML = '<span class="step-group-kw">Background:</span>';
      } else {
        // For scenario steps, inject Rule: label if the scenario has a rule
        // and it hasn't been injected yet (e.g. no rule_bg steps)
        var sInfo = scenarios[data.scenario_index];
        if (sInfo && sInfo.rule) {
          var ruleLabelId = 'label-' + data.scenario_index + '-rule';
          if (!document.getElementById(ruleLabelId)) {
            var rl = document.createElement('div');
            rl.id = ruleLabelId;
            rl.className = 'step-group-label step-group-rule';
            rl.innerHTML = '<span class="step-group-kw">Rule:</span> ' + escapeHtml(sInfo.rule);
            stepsDiv.appendChild(rl);
          }
        }
        label.className += ' step-group-scenario';
        label.innerHTML = '<span class="step-group-kw">Scenario:</span> ' + escapeHtml(sInfo ? sInfo.name : '');
      }
      stepsDiv.appendChild(label);
    }

    var sym = data.status === 'passed' ? '\u2713' : (data.status === 'failed' ? '\u2717' : '\u2013');
    var statusCls = data.status;

    var stepDiv = document.createElement('div');
    stepDiv.className = 'step';
    stepDiv.innerHTML =
      '<span class="step-symbol ' + statusCls + '">' + sym + '</span>' +
      '<span class="step-keyword ' + statusCls + '">' + escapeHtml(data.keyword) + '</span>' +
      colorizeText(data.text, data.match_locs, data.status) +
      '<span class="step-duration">' + formatDuration(data.duration_ms) + '</span>';

    stepsDiv.appendChild(stepDiv);

    if (data.error) {
      var errDiv = document.createElement('div');
      errDiv.className = 'step-error';
      errDiv.textContent = data.error;
      stepsDiv.appendChild(errDiv);
    }
  }

  function colorizeText(text, matchLocs, status) {
    if (!matchLocs || matchLocs.length === 0) {
      var cls = status === 'skipped' ? 'step-text skipped' : 'step-text';
      return '<span class="' + cls + '">' + escapeHtml(text) + '</span>';
    }
    var result = '';
    var cursor = 0;
    var paramIdx = 0;
    var statusCls = status === 'skipped' ? 'skipped' : '';
    for (var i = 0; i + 1 < matchLocs.length; i += 2) {
      var start = matchLocs[i], end = matchLocs[i+1];
      if (start < 0 || end < 0) continue;
      if (cursor < start) {
        result += '<span class="step-text ' + statusCls + '">' + escapeHtml(text.substring(cursor, start)) + '</span>';
      }
      if (status === 'skipped') {
        result += '<span class="step-text skipped">' + escapeHtml(text.substring(start, end)) + '</span>';
      } else {
        var ci = paramIdx % paramColors.length;
        result += '<span class="step-param" style="color:' + paramColors[ci] + '">' + escapeHtml(text.substring(start, end)) + '</span>';
      }
      paramIdx++;
      cursor = end;
    }
    if (cursor < text.length) {
      result += '<span class="step-text ' + statusCls + '">' + escapeHtml(text.substring(cursor)) + '</span>';
    }
    return result;
  }

  // ---- Filter bar ----

  function buildFilterBar() {
    var tagSet = {};
    var hasUntagged = false;
    for (var i = 0; i < scenarios.length; i++) {
      var tags = scenarios[i].tags || [];
      if (tags.length === 0) hasUntagged = true;
      for (var j = 0; j < tags.length; j++) tagSet[tags[j]] = true;
    }
    var allTags = Object.keys(tagSet).sort();
    if (allTags.length === 0 && !hasUntagged) return;

    var bar = document.getElementById('filter-bar');
    // Remove existing tag buttons (keep label and Select All/None)
    var existing = bar.querySelectorAll('.filter-tag');
    for (var k = 0; k < existing.length; k++) existing[k].remove();

    for (var t = 0; t < allTags.length; t++) {
      var btn = document.createElement('button');
      btn.className = 'filter-tag active';
      btn.setAttribute('data-tag', allTags[t]);
      btn.textContent = allTags[t];
      btn.onclick = function() { toggleTagBtn(this); };
      bar.appendChild(btn);
    }
    if (hasUntagged) {
      var noTag = document.createElement('button');
      noTag.className = 'filter-tag active';
      noTag.setAttribute('data-tag', '');
      noTag.textContent = 'No Tag';
      noTag.onclick = function() { toggleTagBtn(this); };
      bar.appendChild(noTag);
    }
    bar.classList.add('visible');
  }

  function toggleTagBtn(btn) {
    btn.classList.toggle('active');
    btn.classList.toggle('inactive');
    applyTagFilter();
  }

  // ---- Summary update ----

  function updateSummaryDOM() {
    document.getElementById('sum-passed').textContent = summary.scenariosPassed;
    document.getElementById('sum-failed').textContent = summary.scenariosFailed;
    document.getElementById('sum-steps-passed').textContent = summary.stepsPassed;
    document.getElementById('sum-steps-failed').textContent = summary.stepsFailed;
    document.getElementById('sum-steps-skipped').textContent = summary.stepsSkipped;
    document.getElementById('sum-steps').textContent = summary.stepsPassed + summary.stepsFailed + summary.stepsSkipped;
  }

  // ---- Helpers ----

  function groupByTags(scenarioList) {
    var groups = {};
    for (var i = 0; i < scenarioList.length; i++) {
      var s = scenarioList[i];
      var key = (s.tags && s.tags.length > 0) ? s.tags.slice().sort().join(', ') : 'Untagged';
      if (!groups[key]) groups[key] = [];
      groups[key].push(s);
    }
    return groups;
  }

  function formatDuration(ms) {
    if (ms <= 0) return '-';
    if (ms < 1) return Math.round(ms * 1000) + '\u00b5s';
    if (ms < 1000) return Math.round(ms) + 'ms';
    return (ms / 1000).toFixed(2) + 's';
  }

  function escapeHtml(s) {
    if (!s) return '';
    return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  }

  // Expose to global scope for inline onclick handlers
  window.selectAllTags = function() {
    document.querySelectorAll('.filter-tag').forEach(function(btn) { btn.classList.add('active'); btn.classList.remove('inactive'); });
    applyTagFilter();
  };
  window.selectNoneTags = function() {
    document.querySelectorAll('.filter-tag').forEach(function(btn) { btn.classList.remove('active'); btn.classList.add('inactive'); });
    applyTagFilter();
  };

  window.applyTagFilter = applyTagFilter;
  function applyTagFilter() {
    var activeTags = [];
    var noTagActive = false;
    document.querySelectorAll('.filter-tag.active').forEach(function(btn) {
      var tag = btn.getAttribute('data-tag');
      if (tag === '') noTagActive = true; else activeTags.push(tag);
    });
    document.querySelectorAll('.scenario').forEach(function(el) {
      var tags = el.getAttribute('data-tags');
      var show = false;
      if (!tags) { show = noTagActive; }
      else {
        var st = tags.split(',');
        for (var i = 0; i < st.length; i++) { if (activeTags.indexOf(st[i]) >= 0) { show = true; break; } }
      }
      el.style.display = show ? '' : 'none';
    });
    document.querySelectorAll('.tag-group').forEach(function(g) {
      var any = false;
      g.querySelectorAll('.scenario').forEach(function(s) { if (s.style.display !== 'none') any = true; });
      g.style.display = any ? '' : 'none';
    });
    document.querySelectorAll('.section').forEach(function(s) {
      var any = false;
      s.querySelectorAll('.scenario').forEach(function(sc) { if (sc.style.display !== 'none') any = true; });
      s.style.display = any ? '' : 'none';
    });
  }
})();

// ---- Global expand/collapse ----
function expandAll() { document.querySelectorAll('.scenario').forEach(function(el) { el.classList.add('open'); }); }
function collapseAll() { document.querySelectorAll('.scenario').forEach(function(el) { el.classList.remove('open'); }); }
function expandSection(btn) { btn.closest('.section').querySelectorAll('.scenario').forEach(function(el) { el.classList.add('open'); }); }
function collapseSection(btn) { btn.closest('.section').querySelectorAll('.scenario').forEach(function(el) { el.classList.remove('open'); }); }
function expandGroup(btn) { btn.closest('.tag-group').querySelectorAll('.scenario').forEach(function(el) { el.classList.add('open'); }); }
function collapseGroup(btn) { btn.closest('.tag-group').querySelectorAll('.scenario').forEach(function(el) { el.classList.remove('open'); }); }
</script>
</body>
</html>
`
