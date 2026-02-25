package cmd

import "github.com/automagik-dev/workit/internal/googleapi"

var newCalendarService = googleapi.NewCalendar

const (
	scopeAll    = literalAll
	scopeSingle = "single"
	scopeFuture = "future"
)
