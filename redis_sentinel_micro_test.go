package main

import (
	"strings"
	"testing"
)

func TestCollectStats_InValidInput(t *testing.T) {
	R, err := CollectStats("")
	if err == nil {
		t.Fail()
	}
	if R != nil {
		t.Fail()
	}
	if !strings.Contains(err.Error(), COLLECTSTATS_INVALID_INPUT) {
		t.Fail()
	}
}

func TestCollectStats_ServerNotReachable(t *testing.T) {

	R, Err := CollectStats("localhost:6379")

	if Err == nil {
		t.Fail()
	}

	if R != nil {
		t.Fail()
	}

	if !strings.Contains(Err.Error(), COLLECTSTATS_SERVER_NOT_REACHABLE) {
		t.Fail()
	}
}

func TestCollectStatsAll_NilInput(t *testing.T) {

	Servers := CollectStatsAll([]string{})

	if Servers != nil {
		t.Fail()
	}
}

func TestCollectStatsAll_NoReachable(t *testing.T) {

	Servers := CollectStatsAll([]string{"localhost:6379"})

	if Servers != nil {
		t.Fail()
	}
}
