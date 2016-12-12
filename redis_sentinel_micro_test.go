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

func TestParseResponseNullInput(t *testing.T) {

	var R Redis

	resultBool := R.ParseResponse("")

	t.Logf("ParseResponse =%v", resultBool)

	if resultBool {
		t.Fail()
	}
}

func TestParseResponseValid(t *testing.T) {

	var R Redis

	sample_input := `# Replication\r\nrole:slave\r\nmaster_host:172.31.10.90\r\nmaster_port:6381\r\nmaster_link_status:up\r\nmaster_last_io_seconds_ago:9\r\nmaster_sync_in_progress:0\r\nslave_repl_offset:44983\r\nslave_priority:100\r\nslave_read_only:1\r\nconnected_slaves:0\r\nmaster_repl_offset:0\r\nrepl_backlog_active:0\r\nrepl_backlog_size:1048576\r\nrepl_backlog_first_byte_offset:0\r\nrepl_backlog_histlen:0\r\n`
	if !R.ParseResponse(sample_input) {
		t.Fail()
	}

	t.Logf("Valid ParseResult = %v", R)

	if R.Role != "slave" {
		t.Fail()
	}
	if R.Priority != 100 {
		t.Fail()
	}
	if R.LastUpdated != 9 {
		t.Fail()
	}
	if R.SyncBytes != 44983 {
		t.Fail()
	}

	t.Logf("R=%v", R)

}
