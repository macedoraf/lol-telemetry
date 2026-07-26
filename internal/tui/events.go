// Package tui implements the Bubble Tea interface for the lol-telemetry daemon client.
package tui

import (
	"fmt"
	"strings"

	"lol-telemetry/pkg/service"
)

// EventsState holds recent game events.
type EventsState struct {
	events []service.EventMessage
}

// Update prepends a new event and keeps the last 100.
func (s *EventsState) Update(event service.EventMessage) {
	s.events = append([]service.EventMessage{event}, s.events...)
	if len(s.events) > 100 {
		s.events = s.events[:100]
	}
}

// View renders the events panel.
func (s EventsState) View(width, height int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Recent Events") + "\n\n")
	if len(s.events) == 0 {
		b.WriteString(placeholderStyle.Render("Waiting for events..."))
	} else {
		for _, ev := range s.events {
			b.WriteString(fmt.Sprintf("[%.1f] %s\n", ev.EventTime, ev.EventName))
		}
	}
	return contentStyle.Width(width - 4).Height(height).Render(b.String())
}
