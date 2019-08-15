// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"time"
	"fmt"
	"log"
	"net/http"
	"os"

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

func calendarMain(client *http.Client, argv []string) {

	eventBuckets := make(map[string][]*calendar.Event)

	if len(argv) > 1 {
		fmt.Fprintln(os.Stderr, "Usage: calendar")
		return
	}

	svc, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to create Calendar service: %v", err)
	}

	listRes, err := svc.CalendarList.List().Fields("items/id").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve list of calendars: %v", err)
	}
	for _, v := range listRes.Items {
		log.Printf("Calendar ID: %v\n", v.Id)
	}

	if len(listRes.Items) > 0 {
		id := ""
		for _, v := range listRes.Items {
			if (v.Id == argv[0]) {
				id = v.Id
			}
			log.Printf("%s is %q", v.Id, v.Primary)
		}
		log.Printf("%s is the primary id", id)

		bucketFunctions := map[string]bucketFunc {
			"personal": isEventPersonal,
			"attended": isEventAcceptedBy(id),
			"1on1": fulfills([]bucketFunc {isEventAcceptedBy(id), isAttendeeCountInRange(2)}),
			"workshop": fulfills([]bucketFunc {isEventAcceptedBy(id), isAttendeeCountInRange(3, 15)}),
			"allhands": fulfills([]bucketFunc {isEventAcceptedBy(id), isAttendeeCountInRange(15, 1000)}),
			"short": fulfills([]bucketFunc {isEventAcceptedBy(id), isDurationInRange(0, 0.51)}),
			"regular": fulfills([]bucketFunc {isEventAcceptedBy(id), isDurationInRange(0.51, 1.1)}),
			"long": fulfills([]bucketFunc {isEventAcceptedBy(id), isDurationInRange(1.1, 4)}),
		}

		log.Printf("Calendar ID: %v\n", id)
		pageToken := ""
		for {
			req := svc.Events.List(id).SingleEvents(true).TimeMin(time.Now().AddDate(0, -3, 0).Format(TIME_FORMAT)).TimeMax(time.Now().Format(TIME_FORMAT)).Fields("items(summary,attendees,start,end)", "summary", "nextPageToken")
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

		weekly := make(map[string][]*calendar.Event)
		monthly := make(map[string][]*calendar.Event)
		for _, v := range eventBuckets["attended"] {
			_, month, week := getTimeSlots(v)
			monthly[month] = append(monthly[month], v)
			weekly[week] = append(weekly[week], v)
		}

		for bucket, events := range eventBuckets {
			count, total, average := stats(events)
			log.Printf("%q: %d %f %f \n", bucket, count, total, average)
		}

		var total float64
		total = 0

		for _, events := range weekly {
			_, totalPerWeek, _ := stats(events)
			total += totalPerWeek
		}

		log.Printf("average per week %f\n", total / float64(len(weekly)))

		for bucket, events := range monthly {
			count, total, average := stats(events)
			log.Printf("%q: %d %f %f \n", bucket, count, total, average)
		}

	}

}

func findMeInAttendees(attendees []*calendar.EventAttendee, email string) *calendar.EventAttendee {
	for _, v := range attendees {
		if v.Email == email {
			return v
		}
	}
	return nil
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
