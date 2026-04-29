package firefox

import (
	"context"
	"encoding/json"
	"strings"

	"code.linenisgreat.com/chrest/go/libs/dewey/bravo/errors"
	"code.linenisgreat.com/chrest/go/src/bravo/bidi"
)

// InterceptedResponse is delivered to the channel returned by
// AddResponseIntercept whenever a top-level response matching the
// pattern is paused at the responseStarted phase. The caller MUST
// invoke either Session.ContinueResponse or Session.FailRequest
// with RequestID before the in-flight Navigate can return.
type InterceptedResponse struct {
	Navigation string
	RequestID  string
	IsBlocked  bool
	URL        string
	Status     int
	Headers    []HTTPHeader
	Intercepts []string
}

type interceptedResponseEvent struct {
	Context    string   `json:"context"`
	Navigation string   `json:"navigation"`
	IsBlocked  bool     `json:"isBlocked"`
	Intercepts []string `json:"intercepts"`
	Request    struct {
		Request string `json:"request"`
	} `json:"request"`
	Response struct {
		URL     string `json:"url"`
		Status  int    `json:"status"`
		Headers []struct {
			Name  string `json:"name"`
			Value struct {
				Value string `json:"value"`
			} `json:"value"`
		} `json:"headers"`
	} `json:"response"`
}

// AddResponseIntercept registers a network.responseStarted intercept
// scoped to this session's browsing context and the given URL pattern,
// and returns the intercept id plus a channel that receives intercept
// events. The channel is closed when RemoveIntercept is called.
func (s *Session) AddResponseIntercept(ctx context.Context, protocol, hostname string) (string, <-chan InterceptedResponse, error) {
	sub := s.conn.SubscribeWithFilter(
		[]string{"network.responseStarted"},
		func(ev bidi.EventFrame) bool {
			var peek interceptedResponseEvent
			if err := json.Unmarshal(ev.Params, &peek); err != nil {
				return false
			}
			return peek.Context == s.contextID && peek.Navigation != "" && peek.IsBlocked
		},
	)

	if _, err := s.conn.Send("session.subscribe", map[string]any{
		"events":   []string{"network.responseStarted"},
		"contexts": []string{s.contextID},
	}); err != nil {
		sub.Close()
		return "", nil, errors.Wrap(err)
	}

	result, err := s.conn.Send("network.addIntercept", map[string]any{
		"phases":   []string{"responseStarted"},
		"contexts": []string{s.contextID},
		"urlPatterns": []map[string]any{
			{"type": "pattern", "protocol": protocol, "hostname": hostname},
		},
	})
	if err != nil {
		sub.Close()
		return "", nil, errors.Wrap(err)
	}
	var added struct {
		Intercept string `json:"intercept"`
	}
	if err := json.Unmarshal(result, &added); err != nil {
		sub.Close()
		return "", nil, errors.Wrap(err)
	}

	out := make(chan InterceptedResponse, 4)
	go func() {
		defer close(out)
		for ev := range sub.Events {
			var decoded interceptedResponseEvent
			if err := json.Unmarshal(ev.Params, &decoded); err != nil {
				continue
			}
			headers := make([]HTTPHeader, 0, len(decoded.Response.Headers))
			for _, h := range decoded.Response.Headers {
				headers = append(headers, HTTPHeader{Name: h.Name, Value: h.Value.Value})
			}
			// Send the typed event to the caller. If the caller has stopped
			// reading (single-event intercept pattern: read one, classify, ignore
			// the rest), drop rather than block. The 4-slot buffer absorbs
			// short bursts; sustained backlog is treated as caller disinterest.
			select {
			case out <- InterceptedResponse{
				Navigation: decoded.Navigation,
				RequestID:  decoded.Request.Request,
				IsBlocked:  decoded.IsBlocked,
				URL:        decoded.Response.URL,
				Status:     decoded.Response.Status,
				Headers:    headers,
				Intercepts: decoded.Intercepts,
			}:
			case <-ctx.Done():
				return
			default:
				// Caller has stopped reading. Drop the event and keep draining
				// sub.Events so the subscription's send-side is never blocked.
			}
		}
	}()

	// Stash the subscription on the session so RemoveIntercept can close it.
	s.intercepts.Store(added.Intercept, sub)

	return added.Intercept, out, nil
}

// ContinueResponse releases a paused request, allowing the response
// to be delivered and Navigate to complete.
func (s *Session) ContinueResponse(ctx context.Context, requestID string) error {
	_, err := s.conn.Send("network.continueResponse", map[string]any{
		"request": requestID,
	})
	return errors.Wrap(err)
}

// FailRequest aborts a paused request. The corresponding Navigate
// call returns a BiDi error wrapping NS_ERROR_ABORT; callers in the
// HTTPError / Binary branches must recognise and swallow that
// specific error.
func (s *Session) FailRequest(ctx context.Context, requestID string) error {
	_, err := s.conn.Send("network.failRequest", map[string]any{
		"request": requestID,
	})
	return errors.Wrap(err)
}

// RemoveIntercept removes a previously-registered intercept and
// closes its event channel.
func (s *Session) RemoveIntercept(ctx context.Context, interceptID string) error {
	if v, ok := s.intercepts.LoadAndDelete(interceptID); ok {
		v.(*bidi.Subscription).Close()
	}
	_, err := s.conn.Send("network.removeIntercept", map[string]any{
		"intercept": interceptID,
	})
	return errors.Wrap(err)
}

// IsAbortedNavigation reports whether err is the BiDi error returned
// by Navigate after an explicit FailRequest.
func IsAbortedNavigation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "NS_ERROR_ABORT")
}
