package nice

import "testing"

func TestPriorityClass(t *testing.T) {
	for n := MinNiceness; n <= MaxNiceness; n++ {
		class := priorityClass(n)
		if back := priorityClass(classNiceness(class)); back != class {
			t.Fatalf("niceness %d: class %#x maps back to %#x", n, class, back)
		}
	}
	tests := map[int]uint32{19: idlePriority, 10: belowNormalPriority, 0: normalPriority, -5: aboveNormalPriority, -20: highPriority}
	for n, want := range tests {
		if got := priorityClass(n); got != want {
			t.Errorf("priorityClass(%d) = %#x, want %#x", n, got, want)
		}
	}
}
