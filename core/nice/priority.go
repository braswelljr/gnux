package nice

// Windows has no niceness; each process has a priority class instead. These
// are the classes nicex maps niceness onto (winbase.h's values), from least
// to most favorable. REALTIME is never used: it can starve the system.
const (
	idlePriority        = 0x00000040
	belowNormalPriority = 0x00004000
	normalPriority      = 0x00000020
	aboveNormalPriority = 0x00008000
	highPriority        = 0x00000080
)

// priorityClass is the class nearest to a niceness.
func priorityClass(niceness int) uint32 {
	switch {
	case niceness >= 15:
		return idlePriority
	case niceness > 0:
		return belowNormalPriority
	case niceness == 0:
		return normalPriority
	case niceness > -15:
		return aboveNormalPriority
	}
	return highPriority
}

// classNiceness is the niceness a priority class stands for, chosen so that
// priorityClass maps it back to the same class.
func classNiceness(class uint32) int {
	switch class {
	case idlePriority:
		return 19
	case belowNormalPriority:
		return 10
	case aboveNormalPriority:
		return -10
	case highPriority, 0x00000100: // REALTIME reads as the most favorable class
		return -20
	}
	return 0
}
