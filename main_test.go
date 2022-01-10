package main

import (
	"fmt"
	"sort"
	"testing"

	"fasteraune.com/calendar_util"
)

func TestStack(t *testing.T) {
	s := NewStack("1234", "fredrik")
	urls := []string{"https://tp.uio.no/uit/timeplan/excel.php?type=course&sort=week&id[]=INF-3203%2C1&id[]=INF-3701%2C1", "https://tp.uio.no/ntnu/timeplan/excel.php?type=courseact&id%5B%5D=GEOG2023%C2%A4&id%5B%5D=KULMI2710%C2%A4&sem=22v&stop=1"}
	csv, err := calendar_util.ReadCsvEvents(urls)
	if err != nil {
		fmt.Println(err)
		return
	}
	s.Push(csv[0])
	event := s.Pop()

	if *event != csv[0] {
		t.Error("Expected", csv[0], "got", *event)
	}
}

func TestStackEmpty(t *testing.T) {
	s := NewStack("1234", "fredrik")
	event := s.Pop()
	if event != nil {
		t.Error("Expected nil got", event)
	}
}

func TestStackOrder(t *testing.T) {
	s := NewStack("1234", "fredrik")
	urls := []string{"https://tp.uio.no/uit/timeplan/excel.php?type=course&sort=week&id[]=INF-3203%2C1&id[]=INF-3701%2C1", "https://tp.uio.no/ntnu/timeplan/excel.php?type=courseact&id%5B%5D=GEOG2023%C2%A4&id%5B%5D=KULMI2710%C2%A4&sem=22v&stop=1"}
	csv, err := calendar_util.ReadCsvEvents(urls)
	if err != nil {
		fmt.Println(err)
		return
	}
	sort.Slice(csv, func(i, j int) bool {
		return csv[i].DtStart.After(csv[j].DtStart.Time)
	})
	for _, event := range csv {
		s.Push(event)
	}

	for i := len(csv) - 1; i >= 0; i-- {
		event := s.Pop()
		if *event != csv[i] {
			t.Error("Expected", csv[i], "got", *event)
		}
	}
}
