package app

import "log"

func SortEvents(flatEvents []*Event) []*Event {
	eventMap := make(map[string]*Event)
	nestedEvents := make([]*Event, 0)

	// Create a map of events with their IDs as keys
	for _, event := range flatEvents {
		eventMap[event.EventID] = event
	}

	// Iterate over the events to build the nested structure
	for _, event := range flatEvents {
		log.Println(event.InReplyTo)
		parentID := event.InReplyTo
		parentEvent, parentExists := eventMap[parentID]

		if parentExists {
			// If the parent event exists, add the current event as its child
			parentEvent.Children = append(parentEvent.Children, event)
		} else {
			// If there is no parent event, add the event to the top-level
			nestedEvents = append(nestedEvents, event)
		}

		// If the event has children, recursively nest them
		if event.Children != nil {
			event.Children = SortEvents(event.Children)
		}
	}

	return nestedEvents
}
