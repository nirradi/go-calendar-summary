// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"time"
	"fmt"
	"log"
	"net/http"
	calendar "google.golang.org/api/calendar/v3"
)
const TIME_FORMAT string = "2006-01-02T15:04:05-07:00"

// calendarMain is an example that demonstrates calling the Calendar API.
// Its purpose is to test out the ability to get maps of struct objects.
//
// Example usage:
//   go build -o go-api-demo *.go
//   go-api-demo -clientid="my-clientid" -secret="my-secret" calendar

type bucketFunc func(*calendar.Event) bool

func getCalendars(client *http.Client) []string {

	svc, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to create Calendar service: %v", err)
	}

	listRes, err := svc.CalendarList.List().Fields("items/id").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve list of calendars: %v", err)
	}
	var result []string
	for _, v := range listRes.Items {
		result = append(result, v.Id)
	}

	return result
}

func getEventSummary(client *http.Client, calendarName string) map[string][]*calendar.Event {

	eventBuckets := make(map[string][]*calendar.Event)

	svc, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to create Calendar service: %v", err)
	}

	id := calendarName

		log.Printf("%s is the primary id", id)

		bucketFunctions := map[string]bucketFunc {
			"attended": isEventAcceptedBy(id),
			"1on1": fulfills([]bucketFunc {isEventAcceptedBy(id), isAttendeeCountInRange(2)}),
			"3 to 6 people": fulfills([]bucketFunc {isEventAcceptedBy(id), isAttendeeCountInRange(3, 6)}),
			"6 to 15 people": fulfills([]bucketFunc {isEventAcceptedBy(id), isAttendeeCountInRange(6, 15)}),
			"more than 15 people": fulfills([]bucketFunc {isEventAcceptedBy(id), isAttendeeCountInRange(15, 1000)}),
/*			"short": fulfills([]bucketFunc {isEventAcceptedBy(id), isDurationInRange(0, 0.51)}),
			"regular": fulfills([]bucketFunc {isEventAcceptedBy(id), isDurationInRange(0.51, 1.1)}),
			"long": fulfills([]bucketFunc {isEventAcceptedBy(id), isDurationInRange(1.1, 4)}),*/
		}

		log.Printf("Calendar ID: %v\n", id)
		pageToken := ""
		for {
			req := svc.Events.List(id).SingleEvents(true).TimeMin(time.Now().AddDate(0, -1, 0).Format(TIME_FORMAT)).TimeMax(time.Now().Format(TIME_FORMAT)).Fields("items(summary,attendees,start,end)", "summary", "nextPageToken")
			if pageToken != "" {
				req.PageToken(pageToken)
			}
			res, err := req.Do()

			if err != nil {
				log.Fatalf("Unable to retrieve calendar events list: %v", err)
			}
			for _, v := range res.Items {

				for bucket, check := range bucketFunctions {
					if check(v) {
						eventBuckets[bucket] = append(eventBuckets[bucket], v)
					}
				}
			}
			if res.NextPageToken == "" {
				break
			}
			pageToken = res.NextPageToken;
		}

		summary := summarizeEvents(eventBuckets)
		for key, value := range summary {
			log.Printf("%s : %s", key, value)
		}

		return eventBuckets
}

func findMeInAttendees(attendees []*calendar.EventAttendee, email string) *calendar.EventAttendee {
	for _, v := range attendees {
		if v.Email == email {
			return v
		}
	}
	return nil
}

func summarizeEvents(eventBuckets map[string][]*calendar.Event) map[string]string {

	summary := make(map[string]string)

	daily := make(map[string][]*calendar.Event)
	weekly := make(map[string][]*calendar.Event)
	monthly := make(map[string][]*calendar.Event)

	for _, v := range eventBuckets["attended"] {
		day, month, week := getTimeSlots(v)
		daily[day] = append(daily[day], v)
		monthly[month] = append(monthly[month], v)
		weekly[week] = append(weekly[week], v)
	}

	allCount, allTotal, _ := stats(eventBuckets["attended"])
	summary["All Meetings"] = fmt.Sprintf("All Meetings: count=%d total hours=%f", allCount, allTotal)

	for bucket, events := range eventBuckets {
		count, total, _ := stats(events)
		summary[fmt.Sprintf("%q meetings", bucket)] = fmt.Sprintf("percent of count=%f%% percent of hours=%f%%", 100.0 * float64(count)/float64(allCount), 100.0 * float64(total)/float64(allTotal))
	}

	var total float64
	total = 0

	for _, events := range weekly {
		_, totalPerWeek, _ := stats(events)
		total += totalPerWeek
	}

	summary["average per week"] = fmt.Sprintf("%f hours", total / float64(len(weekly)))

	total = 0

	for _, events := range daily {
		_, totalPerDay, _ := stats(events)
		total += totalPerDay
	}

	summary["average per day"] = fmt.Sprintf("%f hours", total / float64(len(daily)))

	for bucket, events := range monthly {
		count, total, _ := stats(events)
		summary[fmt.Sprintf("%q", bucket)] = fmt.Sprintf(" count=%d total hours=%f",  count, total)
	}

	return summary
}

func getDuration(event *calendar.Event) float64 {
	start := event.Start.DateTime
	end := event.End.DateTime
	t1, err := time.Parse(TIME_FORMAT, start)
	if err != nil {
		return 0
	}
	t2, err := time.Parse(TIME_FORMAT, end)
	if err != nil {
		return 0
	}

	return t2.Sub(t1).Hours()
}

func isEventPersonal(event *calendar.Event) bool {
	return len(event.Attendees) == 0
}

func isEventAcceptedBy(email string) bucketFunc {
	return isEventStatusForEmail([]string {"tentative", "needsAction", "accepted"}, email)
}

func isEventStatusForEmail(statuses []string, email string) bucketFunc {
	return func(event *calendar.Event) bool {
		if attendee := findMeInAttendees(event.Attendees, email); attendee != nil {
			for _, a := range statuses {
				if a == attendee.ResponseStatus {
					return true
				}
			}
		}

		return false
	}
}

func isEventNotDeclinedBy(email string) bucketFunc {
	return func(event *calendar.Event) bool {
		if attendee := findMeInAttendees(event.Attendees, email); attendee != nil {
			return attendee.ResponseStatus != "declined"
		} else {
			return false
		}
	}
}

func isAttendeeCountInRange(params ...int) bucketFunc {
	return func(event *calendar.Event) bool {
		attendeeCount := len(event.Attendees)
		switch len(params) {
		case 1: return attendeeCount == params[0]
		case 2: return attendeeCount >= params[0] && attendeeCount < params[1]
		default: panic("isAttendeeCountInRange should receive either 1 or 2 params");
		}
		return false
	}
}

func isDurationInRange(params ...float64) bucketFunc {
	return func(event *calendar.Event) bool {
		duration := getDuration(event)
		switch(len(params)) {
		case 1: return duration == params[0]
		case 2: return duration >= params[0] && duration < params[1]
		default: panic("isDurationInRange should receive either 1 or 2 params");
		}
		return false

	}
}

func fulfills(funcs []bucketFunc) bucketFunc {
	return func(event *calendar.Event) bool {
		result := true
		for _, v := range funcs {
			result = result && v(event)
		}

		return result
	}
}

func getTimeSlots(event *calendar.Event) (string, string, string) {
	t,e := time.Parse(TIME_FORMAT, event.Start.DateTime)
	if e != nil {
		return "-", "-", "-"
	}
	day := t.Day()
	month := t.Month().String()
	_, week := t.ISOWeek()
	return fmt.Sprintf("%s %d", month, day), month, fmt.Sprintf("WW %d", week)
}

func stats(events []*calendar.Event) (int, float64, float64) {
	var total float64
	total = 0

	for _, event := range events {
		duration := getDuration(event)
		total += duration
	}
	count := len(events)
	average := total / float64(count)
	return count, total, average
}

/*

	(accepted only)
	attendees:
		0 (--> personal time)
		2 (1 : 1)
		3-8 (work group)
		>8 all hands

	#/% of organizer
	#/% of accepted
	#/% of declined

	meeting length
		less than 30
		30
		60
		more than 60

	average daily
	average weekly
	total monthly
	# of hours with no meetings

*/
