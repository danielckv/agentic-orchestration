package dispatcher

import (
	"testing"
)

func TestTrackRejection(t *testing.T) {
	h := &HITLManager{
		blocked: make(map[string]int),
	}

	// 1st rejection: should not escalate
	if h.TrackRejection("task-1") {
		t.Fatal("expected no escalation on 1st rejection")
	}

	// 2nd rejection: should not escalate
	if h.TrackRejection("task-1") {
		t.Fatal("expected no escalation on 2nd rejection")
	}

	// 3rd rejection: should escalate
	if !h.TrackRejection("task-1") {
		t.Fatal("expected escalation on 3rd rejection")
	}

	// 4th rejection: should still escalate
	if !h.TrackRejection("task-1") {
		t.Fatal("expected escalation on 4th rejection")
	}
}

func TestTrackRejectionIndependentTasks(t *testing.T) {
	h := &HITLManager{
		blocked: make(map[string]int),
	}

	// Two different tasks tracked independently
	h.TrackRejection("task-a")
	h.TrackRejection("task-a")
	h.TrackRejection("task-b")

	// task-a at 2 rejections, task-b at 1
	if h.TrackRejection("task-b") {
		t.Fatal("task-b should not escalate at 2 rejections")
	}
	if !h.TrackRejection("task-a") {
		t.Fatal("task-a should escalate at 3 rejections")
	}
}
