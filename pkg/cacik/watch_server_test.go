package cacik

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// =============================================================================
// EventBroker Tests
// =============================================================================

func TestEventBroker_PublishAndSubscribe(t *testing.T) {
	t.Run("single subscriber receives published event", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ch := broker.Subscribe()
		broker.Publish(EventRunStarted, RunStartedEvent{Total: 3})

		select {
		case msg := <-ch:
			s := string(msg)
			require.Contains(t, s, "event: run_started")
			require.Contains(t, s, `"total":3`)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
	})

	t.Run("multiple subscribers each receive the event", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ch1 := broker.Subscribe()
		ch2 := broker.Subscribe()
		ch3 := broker.Subscribe()

		broker.Publish(EventScenarioStarted, ScenarioStartedEvent{Index: 7})

		for i, ch := range []chan []byte{ch1, ch2, ch3} {
			select {
			case msg := <-ch:
				s := string(msg)
				require.Contains(t, s, "event: scenario_started", "subscriber %d", i)
				require.Contains(t, s, `"index":7`, "subscriber %d", i)
			case <-time.After(time.Second):
				t.Fatalf("subscriber %d timed out", i)
			}
		}
	})

	t.Run("SSE message format has correct structure", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ch := broker.Subscribe()
		broker.Publish(EventStepCompleted, StepCompletedEvent{
			ScenarioIndex: 0,
			StepCategory:  "scenario",
			StepIndex:     1,
			Keyword:       "When ",
			Text:          "user clicks button",
			Status:        "passed",
			DurationMs:    42,
		})

		msg := <-ch
		s := string(msg)
		// Must start with "event: " line
		require.True(t, strings.HasPrefix(s, "event: step_completed\n"))
		// Must have "data: " line
		require.Contains(t, s, "\ndata: ")
		// Must end with double newline (SSE terminator)
		require.True(t, strings.HasSuffix(s, "\n\n"))
	})
}

func TestEventBroker_HistoryReplay(t *testing.T) {
	t.Run("new subscriber receives all past events", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		// Publish 3 events before anyone subscribes
		broker.Publish(EventRunStarted, RunStartedEvent{Total: 2})
		broker.Publish(EventScenarioStarted, ScenarioStartedEvent{Index: 0})
		broker.Publish(EventStepCompleted, StepCompletedEvent{
			ScenarioIndex: 0, StepCategory: "scenario", StepIndex: 0,
			Keyword: "Given ", Text: "something", Status: "passed",
		})

		// Subscribe late
		ch := broker.Subscribe()

		// Should receive all 3 events from history
		var events []string
		for i := 0; i < 3; i++ {
			select {
			case msg := <-ch:
				events = append(events, string(msg))
			case <-time.After(time.Second):
				t.Fatalf("timed out waiting for history event %d", i)
			}
		}

		require.Contains(t, events[0], "event: run_started")
		require.Contains(t, events[1], "event: scenario_started")
		require.Contains(t, events[2], "event: step_completed")
	})

	t.Run("late subscriber receives history plus live events", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		// Publish before subscribing
		broker.Publish(EventRunStarted, RunStartedEvent{Total: 1})

		ch := broker.Subscribe()

		// Drain history
		<-ch

		// Now publish a live event
		broker.Publish(EventScenarioStarted, ScenarioStartedEvent{Index: 0})

		select {
		case msg := <-ch:
			require.Contains(t, string(msg), "event: scenario_started")
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for live event")
		}
	})
}

func TestEventBroker_Unsubscribe(t *testing.T) {
	t.Run("unsubscribed client channel is closed", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ch := broker.Subscribe()
		broker.Unsubscribe(ch)

		// Channel should be closed
		_, ok := <-ch
		require.False(t, ok, "channel should be closed after unsubscribe")
	})

	t.Run("unsubscribed client does not receive new events", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ch1 := broker.Subscribe()
		ch2 := broker.Subscribe()

		broker.Unsubscribe(ch1)

		// Publish after ch1 is unsubscribed
		broker.Publish(EventRunStarted, RunStartedEvent{Total: 1})

		// ch2 should still get the event
		select {
		case msg := <-ch2:
			require.Contains(t, string(msg), "event: run_started")
		case <-time.After(time.Second):
			t.Fatal("ch2 timed out")
		}

		// ch1 channel is closed, reading returns zero value
		msg, ok := <-ch1
		require.False(t, ok)
		require.Nil(t, msg)
	})
}

func TestEventBroker_Close(t *testing.T) {
	t.Run("close closes all client channels", func(t *testing.T) {
		broker := NewEventBroker()

		ch1 := broker.Subscribe()
		ch2 := broker.Subscribe()

		broker.Close()

		_, ok1 := <-ch1
		_, ok2 := <-ch2
		require.False(t, ok1)
		require.False(t, ok2)
	})

	t.Run("publish after close is a no-op", func(t *testing.T) {
		broker := NewEventBroker()
		broker.Close()

		// Should not panic
		broker.Publish(EventRunStarted, RunStartedEvent{Total: 1})
	})

	t.Run("close is idempotent for publish", func(t *testing.T) {
		broker := NewEventBroker()
		broker.Close()
		// Calling Publish on closed broker should not panic
		require.NotPanics(t, func() {
			broker.Publish(EventRunCompleted, RunCompletedEvent{DurationMs: 100})
		})
	})
}

func TestEventBroker_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent publishers and subscribers do not race", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		const numPublishers = 5
		const numSubscribers = 5
		const eventsPerPublisher = 20

		var wg sync.WaitGroup

		// Start subscribers
		var channels []chan []byte
		for i := 0; i < numSubscribers; i++ {
			ch := broker.Subscribe()
			channels = append(channels, ch)
		}

		// Start publishers concurrently
		for i := 0; i < numPublishers; i++ {
			wg.Add(1)
			go func(publisherID int) {
				defer wg.Done()
				for j := 0; j < eventsPerPublisher; j++ {
					broker.Publish(EventStepCompleted, StepCompletedEvent{
						ScenarioIndex: publisherID,
						StepIndex:     j,
						Status:        "passed",
					})
				}
			}(i)
		}

		wg.Wait()

		// Each subscriber should have received events (at least some, possibly
		// dropped due to buffer overflow under heavy load, which is by design)
		for i, ch := range channels {
			count := 0
			// Drain buffered events
		drain:
			for {
				select {
				case <-ch:
					count++
				default:
					break drain
				}
			}
			require.Greater(t, count, 0, "subscriber %d should have received at least some events", i)
		}
	})
}

// =============================================================================
// WatchServer Tests
// =============================================================================

func TestWatchServer_StartAndShutdown(t *testing.T) {
	t.Run("starts on a free port and returns valid address", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ws, err := StartWatchServer(broker)
		require.NoError(t, err)
		defer ws.Shutdown(time.Second)

		addr := ws.Addr()
		require.True(t, strings.HasPrefix(addr, "http://localhost:"))
		// Port should be a valid number > 0
		require.NotEqual(t, "http://localhost:0", addr)
	})

	t.Run("shutdown completes without error", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ws, err := StartWatchServer(broker)
		require.NoError(t, err)

		// Shutdown should complete within timeout
		done := make(chan struct{})
		go func() {
			ws.Shutdown(2 * time.Second)
			close(done)
		}()

		select {
		case <-done:
			// success
		case <-time.After(3 * time.Second):
			t.Fatal("shutdown did not complete in time")
		}
	})
}

func TestWatchServer_IndexHandler(t *testing.T) {
	t.Run("GET / returns 200 with HTML content type", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ws, err := StartWatchServer(broker)
		require.NoError(t, err)
		defer ws.Shutdown(time.Second)

		resp, err := http.Get(ws.Addr() + "/")
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Contains(t, resp.Header.Get("Content-Type"), "text/html")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "Live Test Watch")
		require.Contains(t, string(body), "EventSource")
	})
}

func TestWatchServer_SSEHandler(t *testing.T) {
	t.Run("GET /events returns SSE content type and streams events", func(t *testing.T) {
		broker := NewEventBroker()
		ws, err := StartWatchServer(broker)
		require.NoError(t, err)
		defer func() {
			ws.Shutdown(time.Second)
			broker.Close()
		}()

		// Open SSE connection
		req, err := http.NewRequest("GET", ws.Addr()+"/events", nil)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")

		// Small delay so the handler's Subscribe() call completes
		time.Sleep(50 * time.Millisecond)

		// Publish an event while the SSE client is connected
		broker.Publish(EventRunStarted, RunStartedEvent{
			Total: 2,
			Scenarios: []ScenarioMetadataEvent{
				{Index: 0, Name: "first", Feature: "feat"},
				{Index: 1, Name: "second", Feature: "feat"},
			},
		})

		// Read from SSE stream
		scanner := bufio.NewScanner(resp.Body)
		var lines []string
		for scanner.Scan() {
			line := scanner.Text()
			lines = append(lines, line)
			// SSE events end with an empty line; stop after we get the first event
			if line == "" && len(lines) > 1 {
				break
			}
		}

		joined := strings.Join(lines, "\n")
		require.Contains(t, joined, "event: run_started")
		require.Contains(t, joined, `"total":2`)
	})

	t.Run("SSE client receives history on connect", func(t *testing.T) {
		broker := NewEventBroker()
		ws, err := StartWatchServer(broker)
		require.NoError(t, err)
		defer func() {
			ws.Shutdown(time.Second)
			broker.Close()
		}()

		// Publish events BEFORE SSE client connects
		broker.Publish(EventRunStarted, RunStartedEvent{Total: 1})
		broker.Publish(EventScenarioStarted, ScenarioStartedEvent{Index: 0})

		// Now connect SSE client
		req, err := http.NewRequest("GET", ws.Addr()+"/events", nil)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Read events - should get both history events
		scanner := bufio.NewScanner(resp.Body)
		var eventTypes []string
		emptyCount := 0
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				eventTypes = append(eventTypes, strings.TrimPrefix(line, "event: "))
			}
			if line == "" {
				emptyCount++
			}
			// Stop after receiving both events (2 empty lines = 2 SSE events)
			if emptyCount >= 2 {
				break
			}
		}

		require.Equal(t, []string{"run_started", "scenario_started"}, eventTypes)
	})

	t.Run("client disconnect does not panic", func(t *testing.T) {
		broker := NewEventBroker()
		ws, err := StartWatchServer(broker)
		require.NoError(t, err)
		defer func() {
			ws.Shutdown(time.Second)
			broker.Close()
		}()

		// Connect and immediately disconnect
		ctx, cancel := context.WithCancel(context.Background())
		req, err := http.NewRequestWithContext(ctx, "GET", ws.Addr()+"/events", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		cancel()

		// Give server time to handle the disconnect
		time.Sleep(50 * time.Millisecond)

		// Publish after disconnect — should not panic
		require.NotPanics(t, func() {
			broker.Publish(EventRunStarted, RunStartedEvent{Total: 1})
		})
	})

	t.Run("multiple concurrent SSE clients", func(t *testing.T) {
		broker := NewEventBroker()
		ws, err := StartWatchServer(broker)
		require.NoError(t, err)
		defer func() {
			ws.Shutdown(time.Second)
			broker.Close()
		}()

		const numClients = 3
		type clientResult struct {
			events []string
			err    error
		}

		results := make(chan clientResult, numClients)

		// Start multiple SSE clients
		for i := 0; i < numClients; i++ {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				req, err := http.NewRequestWithContext(ctx, "GET", ws.Addr()+"/events", nil)
				if err != nil {
					results <- clientResult{err: err}
					return
				}

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					results <- clientResult{err: err}
					return
				}
				defer resp.Body.Close()

				scanner := bufio.NewScanner(resp.Body)
				var events []string
				for scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, "event: ") {
						events = append(events, strings.TrimPrefix(line, "event: "))
					}
					if line == "" && len(events) > 0 {
						break
					}
				}
				results <- clientResult{events: events}
			}()
		}

		// Give clients time to connect
		time.Sleep(100 * time.Millisecond)

		// Publish an event
		broker.Publish(EventRunStarted, RunStartedEvent{Total: 5})

		// All clients should receive the event
		for i := 0; i < numClients; i++ {
			select {
			case r := <-results:
				require.NoError(t, r.err)
				require.Contains(t, r.events, "run_started", "client %d", i)
			case <-time.After(5 * time.Second):
				t.Fatalf("client %d timed out", i)
			}
		}
	})
}

// =============================================================================
// Event Payload Tests
// =============================================================================

func TestEventPayloads(t *testing.T) {
	t.Run("RunStartedEvent serializes correctly", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ch := broker.Subscribe()

		broker.Publish(EventRunStarted, RunStartedEvent{
			Total: 2,
			Scenarios: []ScenarioMetadataEvent{
				{Index: 0, Name: "Login success", Feature: "Auth", Rule: "Login", Tags: []string{"@smoke"}},
				{Index: 1, Name: "Login failure", Feature: "Auth", Rule: "Login", Tags: []string{"@smoke", "@negative"}},
			},
		})

		msg := string(<-ch)
		require.Contains(t, msg, `"total":2`)
		require.Contains(t, msg, `"name":"Login success"`)
		require.Contains(t, msg, `"feature":"Auth"`)
		require.Contains(t, msg, `"rule":"Login"`)
		require.Contains(t, msg, `"tags":["@smoke"]`)
		require.Contains(t, msg, `"tags":["@smoke","@negative"]`)
	})

	t.Run("StepCompletedEvent includes match_locs", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ch := broker.Subscribe()

		broker.Publish(EventStepCompleted, StepCompletedEvent{
			ScenarioIndex: 0,
			StepCategory:  "scenario",
			StepIndex:     0,
			Keyword:       "Given ",
			Text:          `user "alice" logs in with "secret"`,
			Status:        "passed",
			DurationMs:    5,
			MatchLocs:     []int{6, 11, 27, 33},
		})

		msg := string(<-ch)
		require.Contains(t, msg, `"match_locs":[6,11,27,33]`)
		require.Contains(t, msg, `"step_category":"scenario"`)
		require.Contains(t, msg, `"keyword":"Given "`)
	})

	t.Run("ScenarioCompletedEvent with error", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ch := broker.Subscribe()

		broker.Publish(EventScenarioCompleted, ScenarioCompletedEvent{
			Index:      2,
			Passed:     false,
			Error:      "expected 200 got 404",
			DurationMs: 1500,
		})

		msg := string(<-ch)
		require.Contains(t, msg, `"passed":false`)
		require.Contains(t, msg, `"error":"expected 200 got 404"`)
		require.Contains(t, msg, `"duration_ms":1500`)
	})

	t.Run("RunCompletedEvent with full summary", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ch := broker.Subscribe()

		broker.Publish(EventRunCompleted, RunCompletedEvent{
			DurationMs: 5000,
			Summary: RunCompletedSummary{
				ScenariosTotal:  10,
				ScenariosPassed: 8,
				ScenariosFailed: 2,
				StepsTotal:      30,
				StepsPassed:     25,
				StepsFailed:     2,
				StepsSkipped:    3,
			},
		})

		msg := string(<-ch)
		require.Contains(t, msg, `"scenarios_total":10`)
		require.Contains(t, msg, `"scenarios_passed":8`)
		require.Contains(t, msg, `"scenarios_failed":2`)
		require.Contains(t, msg, `"steps_total":30`)
		require.Contains(t, msg, `"steps_passed":25`)
		require.Contains(t, msg, `"steps_failed":2`)
		require.Contains(t, msg, `"steps_skipped":3`)
		require.Contains(t, msg, `"duration_ms":5000`)
	})
}

// =============================================================================
// Full Flow Integration Test
// =============================================================================

func TestWatchServer_FullFlow(t *testing.T) {
	t.Run("complete lifecycle: start, stream events, shutdown", func(t *testing.T) {
		broker := NewEventBroker()
		ws, err := StartWatchServer(broker)
		require.NoError(t, err)

		// Connect SSE client
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", ws.Addr()+"/events", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Simulate a full test run
		broker.Publish(EventRunStarted, RunStartedEvent{
			Total: 1,
			Scenarios: []ScenarioMetadataEvent{
				{Index: 0, Name: "test scenario", Feature: "test feature", Tags: []string{"@smoke"}},
			},
		})

		broker.Publish(EventScenarioStarted, ScenarioStartedEvent{Index: 0})

		broker.Publish(EventStepCompleted, StepCompletedEvent{
			ScenarioIndex: 0, StepCategory: "scenario", StepIndex: 0,
			Keyword: "Given ", Text: "a precondition", Status: "passed", DurationMs: 10,
		})

		broker.Publish(EventStepCompleted, StepCompletedEvent{
			ScenarioIndex: 0, StepCategory: "scenario", StepIndex: 1,
			Keyword: "When ", Text: "an action", Status: "passed", DurationMs: 20,
		})

		broker.Publish(EventStepCompleted, StepCompletedEvent{
			ScenarioIndex: 0, StepCategory: "scenario", StepIndex: 2,
			Keyword: "Then ", Text: "a result", Status: "passed", DurationMs: 5,
		})

		broker.Publish(EventScenarioCompleted, ScenarioCompletedEvent{
			Index: 0, Passed: true, DurationMs: 35,
		})

		broker.Publish(EventRunCompleted, RunCompletedEvent{
			DurationMs: 40,
			Summary: RunCompletedSummary{
				ScenariosTotal: 1, ScenariosPassed: 1, ScenariosFailed: 0,
				StepsTotal: 3, StepsPassed: 3, StepsFailed: 0, StepsSkipped: 0,
			},
		})

		// Read all events from the stream
		scanner := bufio.NewScanner(resp.Body)
		var eventTypes []string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				eventTypes = append(eventTypes, strings.TrimPrefix(line, "event: "))
			}
			// Stop after run_completed
			if strings.HasPrefix(line, "event: run_completed") {
				// Read data and empty line
				scanner.Scan() // data line
				scanner.Scan() // empty line
				break
			}
		}

		require.Equal(t, []string{
			"run_started",
			"scenario_started",
			"step_completed",
			"step_completed",
			"step_completed",
			"scenario_completed",
			"run_completed",
		}, eventTypes)

		// Shutdown
		ws.Shutdown(2 * time.Second)
		broker.Close()
	})
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestEventBroker_SlowClient(t *testing.T) {
	t.Run("slow client does not block other clients", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		slowCh := broker.Subscribe()
		fastCh := broker.Subscribe()

		// Fill up the slow client's buffer (capacity is 256)
		for i := 0; i < 300; i++ {
			broker.Publish(EventStepCompleted, StepCompletedEvent{
				ScenarioIndex: 0, StepIndex: i, Status: "passed",
			})
		}

		// Fast client should have events in its buffer
		count := 0
	drain:
		for {
			select {
			case <-fastCh:
				count++
			default:
				break drain
			}
		}
		require.Greater(t, count, 0, "fast client should have received events")

		// Slow client: drain what we can
		slowCount := 0
	drainSlow:
		for {
			select {
			case <-slowCh:
				slowCount++
			default:
				break drainSlow
			}
		}
		// Slow client receives at least its buffer worth (256 from subscribe history + possibly some live)
		require.Greater(t, slowCount, 0)
	})
}

func TestEventBroker_EmptyBroker(t *testing.T) {
	t.Run("subscribe to broker with no history returns empty channel", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ch := broker.Subscribe()

		select {
		case <-ch:
			t.Fatal("should not receive anything from empty broker")
		case <-time.After(50 * time.Millisecond):
			// expected
		}
	})
}

func TestWatchServer_MultipleServersOnDifferentPorts(t *testing.T) {
	t.Run("two servers can run simultaneously", func(t *testing.T) {
		broker1 := NewEventBroker()
		broker2 := NewEventBroker()
		defer broker1.Close()
		defer broker2.Close()

		ws1, err := StartWatchServer(broker1)
		require.NoError(t, err)
		defer ws1.Shutdown(time.Second)

		ws2, err := StartWatchServer(broker2)
		require.NoError(t, err)
		defer ws2.Shutdown(time.Second)

		require.NotEqual(t, ws1.Addr(), ws2.Addr())

		// Both should be reachable
		resp1, err := http.Get(ws1.Addr() + "/")
		require.NoError(t, err)
		resp1.Body.Close()
		require.Equal(t, http.StatusOK, resp1.StatusCode)

		resp2, err := http.Get(ws2.Addr() + "/")
		require.NoError(t, err)
		resp2.Body.Close()
		require.Equal(t, http.StatusOK, resp2.StatusCode)
	})
}

func TestWatchServer_ShutdownAfterConnection(t *testing.T) {
	t.Run("server shutdown closes active SSE connections gracefully", func(t *testing.T) {
		broker := NewEventBroker()
		ws, err := StartWatchServer(broker)
		require.NoError(t, err)

		// Connect SSE client
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", ws.Addr()+"/events", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Shutdown server while client is connected
		ws.Shutdown(2 * time.Second)
		broker.Close()

		// Reading from closed connection should eventually return EOF or error
		buf := make([]byte, 1024)
		_, readErr := resp.Body.Read(buf)
		// We expect either io.EOF or an error indicating the connection was closed
		if readErr != nil {
			require.True(t, readErr == io.EOF || strings.Contains(readErr.Error(), "closed") ||
				strings.Contains(readErr.Error(), "EOF"),
				"unexpected error: %v", readErr)
		}
	})
}

func TestWatchServer_AfterShutdownReturnsConnectionError(t *testing.T) {
	t.Run("requests after shutdown fail", func(t *testing.T) {
		broker := NewEventBroker()
		defer broker.Close()

		ws, err := StartWatchServer(broker)
		require.NoError(t, err)

		addr := ws.Addr()
		ws.Shutdown(time.Second)

		// Give the server a moment to fully stop
		time.Sleep(50 * time.Millisecond)

		// Requests to the shutdown server should fail
		_, err = http.Get(addr + "/")
		require.Error(t, err, "expected connection error after shutdown")
	})
}

// =============================================================================
// Event Type Constants Test
// =============================================================================

func TestEventTypeConstants(t *testing.T) {
	t.Run("event type constants have expected values", func(t *testing.T) {
		require.Equal(t, WatchEventType("run_started"), EventRunStarted)
		require.Equal(t, WatchEventType("scenario_started"), EventScenarioStarted)
		require.Equal(t, WatchEventType("step_completed"), EventStepCompleted)
		require.Equal(t, WatchEventType("scenario_completed"), EventScenarioCompleted)
		require.Equal(t, WatchEventType("run_completed"), EventRunCompleted)
	})

	t.Run("all event types are distinct", func(t *testing.T) {
		types := []WatchEventType{
			EventRunStarted, EventScenarioStarted, EventStepCompleted,
			EventScenarioCompleted, EventRunCompleted,
		}
		seen := map[WatchEventType]bool{}
		for _, et := range types {
			require.False(t, seen[et], "duplicate event type: %s", et)
			seen[et] = true
		}
	})
}

// =============================================================================
// Benchmark
// =============================================================================

func BenchmarkEventBroker_Publish(b *testing.B) {
	broker := NewEventBroker()
	defer broker.Close()

	// 3 subscribers
	for i := 0; i < 3; i++ {
		ch := broker.Subscribe()
		go func() {
			for range ch {
			}
		}()
	}

	event := StepCompletedEvent{
		ScenarioIndex: 0, StepIndex: 0,
		Keyword: "Given ", Text: "something", Status: "passed", DurationMs: 1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.Publish(EventStepCompleted, event)
	}
}

func BenchmarkEventBroker_Subscribe(b *testing.B) {
	broker := NewEventBroker()
	defer broker.Close()

	// Pre-fill history
	for i := 0; i < 100; i++ {
		broker.Publish(EventStepCompleted, StepCompletedEvent{StepIndex: i, Status: "passed"})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := broker.Subscribe()
		// Drain history
		for j := 0; j < 100; j++ {
			<-ch
		}
		broker.Unsubscribe(ch)
	}
}

// =============================================================================
// Helpers
// =============================================================================

// readSSEEvent reads a single SSE event from the reader, returning the event
// type and data.
func readSSEEvent(reader *bufio.Reader) (eventType, data string, err error) {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return eventType, data, fmt.Errorf("reading SSE: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		} else if line == "" {
			// End of this SSE event
			if eventType != "" || data != "" {
				return eventType, data, nil
			}
		}
	}
}
