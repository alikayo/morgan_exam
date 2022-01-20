package main

import (
	"errors"
	"log"
	"time"
)

var DateTimeFormats = []string{
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"1/2/2006 15:04:05",
	"1/2/2006 15:04",
	"1/2/06 15:04:05",
	"1/2/06 15:04",
	"01/02/2006 15:04:05",
	"01/02/2006 15:04",
	"01/02/06 15:04:05",
	"01/02/06 15:04",
	"2006-01-02",
	"01/02/2006",
	"01/02/06",
}

func DateTimeParse(s string) (time.Time, error) {
	for i := 0; i < len(DateTimeFormats); i++ {
		layout := DateTimeFormats[i]
		//log.Printf("trying %s on %s\n", layout, s)
		dt, er := time.Parse(layout, s)
		if er == nil {
			return dt, nil
		}
	}
	log.Printf("unknown format %s\n", s)
	return time.Time{}, errors.New("datetime format not on list")
}
