package paralleltree

import (
	"log"
	"strconv"
	"sync/atomic"
)

// debug is whether or not debugging is enabled.
var debug = true

// Node is a node in the tree. It is designed to be executed by
// `ProcessConcurrently`.
type Node struct {
	// All fields are private to minimize the risk of the caller concurrently
	// mutating something while `ProcessConcurrently` is executing.

	// id is just for debug logs; these values should be unique for the whole
	// tree.
	id string

	// children are the child nodes.
	children []*Node

	// f is the work to do when visiting the node. This may not be nil.
	f func() error

	// acquired tells processNode whether or not the node is (or has been)
	// acquired.
	acquired abool

	// finished tells processNode whether or not the node is finished
	finished abool
}

// NewNode creates a new Node from the provided parameters.
//
// * `id` is an identifier (unique to the whole node tree including ancestors,
//   cousins, etc) used only for debugging.
// * `children` are the node's children nodes.
// * `f` is the function to execute when the node is visited. This may not be
//   nil.
func NewNode(id string, children []*Node, f func() error) *Node {
	return &Node{id: id, children: children, f: f}
}

// reset resets the 'acquired' and 'finished' properties of every node to
// false. This is to ensure that a tree which has been run before can be rerun.
func reset(n *Node) {
	set(&n.acquired, true)
	set(&n.finished, true)
	for _, child := range n.children {
		reset(child)
	}
}

// ProcessConcurrently concurrently visits each node in the tree (depth first,
// beginning at the leaves) and runs every node's `f()` such that a given
// parent node's `f()` is not executed before those of all of its children. The
// concurrency is controlled by the `concurrency` parameter, and on error, each
// worker process finishes whatever it is doing before exiting, and the most
// recently encountered error is returned.
//
// WARNING: Do not allow another invocation of ProcessConcurrently to run
// concurrently for `n` or any subset thereof.
func ProcessConcurrently(n *Node, concurrency int) error {
	// reset the tree so it can be rerun at a later time, if necessary.
	defer reset(n)

	errs := make(chan error)
	cancel := afalse
	for i := 0; i < concurrency; i++ {
		go func(i int) { errs <- processNode(strconv.Itoa(i), n, &cancel) }(i)
	}

	// Once we get an error, cancel. This will cause all workers to finish what
	// they are doing and then return. We will still await every worker, and
	// (for simplicity) we will return the last received error.
	var out error
	for i := 0; i < concurrency; i++ {
		if err := <-errs; err != nil {
			set(&cancel, true)
			out = err
		}
	}
	return out
}

func debugf(format string, v ...interface{}) {
	if debug {
		log.Printf(format, v...)
	}
}

type abool int32

func get(value *abool) bool {
	return atomic.LoadInt32((*int32)(value)) != int32(afalse)
}

func set(value *abool, boolValue bool) {
	var in int32
	if boolValue {
		in = int32(atrue)
	} else {
		in = int32(afalse)
	}
	atomic.StoreInt32((*int32)(value), in)
}

func swap(value *abool, old, new bool) bool {
	o, n := afalse, afalse
	if old {
		o = atrue
	}
	if new {
		n = atrue
	}
	return atomic.CompareAndSwapInt32((*int32)(value), int32(o), int32(n))
}

func toggleIfFalse(value *abool) bool { return swap(value, false, true) }

const (
	afalse abool = 0
	atrue  abool = 1
)

func acquired(n *Node) bool { return get(&n.acquired) }

func acquire(worker string, n *Node) bool {
	if toggleIfFalse(&n.acquired) {
		debugf("Worker %s acquired node %s", worker, n.id)
		return true
	}
	debugf("Worker %s failed to acquire node %s", worker, n.id)
	return false
}

func nextFreeChild(n *Node) *Node {
	for _, child := range n.children {
		if !acquired(child) {
			return child
		}
	}

	return nil
}

func processNode(worker string, n *Node, cancel *abool) error {
	// If the node is nil, then we're finished.
	if n == nil {
		debugf("Worker %s found a nil node", worker)
		return nil
	}

	// For as long as there are free children, process them.
	for {
		// Check to see if we've been canceled.
		if get(cancel) {
			debugf(
				"Worker %s got the canceled signal; exiting node %s",
				worker,
				n.id,
			)
			return nil
		}

		if child := nextFreeChild(n); child != nil {
			debugf(
				"Worker %s is moving from parent %s into child %s",
				worker,
				n.id,
				child.id,
			)
			if err := processNode(worker, child, cancel); err != nil {
				return err
			}
			continue
		}

		debugf(
			"Worker %s found no more free children on node %s",
			worker,
			n.id,
		)
		break
	}

	// Only process the current node if there are no more children in-flight.
	// Otherwise, move onto the node's next sibling. The worker that finishes
	// the last child will process this node.
	for _, child := range n.children {
		if !get(&child.finished) {
			// debugf(
			// 	"%s's child %s is not finished; returning to parent",
			// 	n.id,
			// 	child.id,
			// )
			return nil
		}
	}

	// If there are no more free children, process the current node if it is
	// available before moving onto the node's next sibling.
	if acquire(worker, n) {
		debugf("Worker %s is beginning work on node %s", worker, n.id)
		if err := n.f(); err != nil {
			return err
		}
		set(&n.finished, true)
	}

	// Move onto the node's next sibling.
	debugf("Worker %s: returning from %s", worker, n.id)
	return nil
}

// func mknode(id string, children ...*Node) *Node {
// 	return NewNode(
// 		id,
// 		children,
// 		func() error { time.Sleep(1 * time.Second); return nil },
// 	)
// }
//
// func main() {
// 	if err := ProcessConcurrently(
// 		mknode(
// 			"root",
// 			mknode("root.0"),
// 			mknode(
// 				"root.1",
// 				mknode("root.1.0"),
// 				mknode("root.1.1"),
// 				mknode("root.1.2"),
// 				mknode("root.1.3"),
// 				mknode("root.1.4"),
// 				mknode("root.1.5"),
// 				mknode("root.1.6"),
// 			),
// 			mknode("root.2"),
// 		),
// 		3,
// 	); err != nil {
// 		log.Fatal(err)
// 	}
// }
