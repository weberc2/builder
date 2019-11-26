package paralleltree

import (
	"sync/atomic"
	"testing"
	"time"
)

func init() {
	// Toggling on debug because this has some side effect which causes
	// concurrency bugs to be much more likely to appear (also the test output
	// is helpful).
	debug = true
}

func TestAllNodesExecuteExactlyOnce(t *testing.T) {
	t.Parallel()

	var childCount int32
	var parentCount int32
	concurrency := 8
	if err := ProcessConcurrently(
		NewNode(
			"parent",
			[]*Node{
				NewNode(
					"child",
					nil,
					func() error {
						atomic.AddInt32(&childCount, 1)
						// Sleep for long enough that other workers have time
						// to be scheduled and potentially enter this function
						// if indeed there is a bug that allows them to do so.
						time.Sleep(10 * time.Millisecond)
						return nil
					},
				),
			},
			func() error {
				atomic.AddInt32(&parentCount, 1)
				// Sleep for long enough that other workers have time to be
				// scheduled and potentially enter this function if indeed
				// there is a bug that allows them to do so.
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		),
		concurrency,
	); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if childCount != 1 {
		t.Errorf(
			"Expected child was executed exactly once; found %d executions",
			childCount,
		)
	}
	if parentCount != 1 {
		t.Errorf(
			"Expected parent was executed exactly once; found %d executions",
			parentCount,
		)
	}
}

func TestOrder(t *testing.T) {
	t.Parallel()

	// An atomic boolean value that we will use to determine if the parent ran
	// before the child finished.
	value := afalse

	// We'll run two concurrent processes (goroutines)
	concurrency := 2

	if err := ProcessConcurrently(
		NewNode(
			"parent",
			[]*Node{
				NewNode(
					"child",
					nil,
					func() error {
						// Wait long enough that the second process should be
						// scheduled if there is indeed a concurrency bug.
						time.Sleep(10 * time.Millisecond)

						// Set the value from false to true. The parent's
						// function will look at this value to determine
						// whether or not the child function finished running.
						set(&value, true)
						return nil
					},
				),
			},
			func() error {
				// Read the value. If it's false, it means that the child
				// didn't finish before the parent began executing.
				if !get(&value) {
					t.Errorf("Parent began before child finished.")
				}
				return nil
			},
		),
		concurrency,
	); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}
