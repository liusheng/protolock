package protolock

import "fmt"

var (
	// ruleFuncs provides a complete list of all funcs to be run to compare
	// a set of Protolocks. This list should be updated as new RuleFunc's
	// are added to this package.
	ruleFuncs = []RuleFunc{
		NoUsingReservedFields,
		NoRemovingReservedFields,
		NoChangeFieldIDs,
		NoChangeFieldTypes,
		NoRenamingFields,
		NoRemovingRPCs,
	}

	strictMode = true
	debug      = false
)

// SetStrictMode enables the user to toggle strict mode on and off.
func SetStrictMode(mode bool) {
	strictMode = mode
}

// SetDebug enables the user to toggle debug mode on and off.
func SetDebug(status bool) {
	debug = status
}

// RuleFunc defines the common signature for a function which can compare
// Protolock states and determine if issues exist.
type RuleFunc func(current, updated Protolock) ([]Warning, bool)

// lockReservedIDsMap:
// table of filepath -> message name -> reserved field ID -> times ID encountered
// i.e.
/*
	["test.proto"] 	-> ["Test"] -> [1] -> 1

			-> ["User"] -> [1] -> 1
				       [2] -> 1
				       [3] -> 1

			-> ["Plan"] -> [1] -> 1
				       [2] -> 1
				       [3] -> 1
*/
type lockReservedIDsMap map[string]map[string]map[int]int

// lockReservedNamesMap:
// table of filepath -> message name -> reserved field name -> times name encountered
// i.e.
/*
	["test.proto"] 	-> ["Test"] -> ["field_one"] -> 1

			-> ["User"] -> ["field_one"] -> 1
				       ["field_two"] -> 1
				       ["field_three"] -> 1

			-> ["Plan"] -> ["field_one"] -> 1
				       ["field_two"] -> 1
				       ["field_three"] -> 1
*/
type lockReservedNamesMap map[string]map[string]map[string]int

// NoUsingReservedFields compares the current vs. updated Protolock definitions
// and will return a list of warnings if any message's previously reserved fields
// are now being used as part of the same message.
func NoUsingReservedFields(cur, upd Protolock) ([]Warning, bool) {
	if debug {
		beginRuleDebug("NoUsingReservedFields")
	}
	reservedIDMap, reservedNameMap := getReservedFields(cur)

	// add each messages field name/number to the existing list identified as
	// reserved to analyze
	for _, def := range upd.Definitions {
		if reservedIDMap[def.Filepath] == nil {
			reservedIDMap[def.Filepath] = make(map[string]map[int]int)
		}
		if reservedNameMap[def.Filepath] == nil {
			reservedNameMap[def.Filepath] = make(map[string]map[string]int)
		}
		for _, msg := range def.Def.Messages {
			for _, field := range msg.Fields {
				if reservedIDMap[def.Filepath][msg.Name] == nil {
					reservedIDMap[def.Filepath][msg.Name] = make(map[int]int)
				}
				if reservedNameMap[def.Filepath][msg.Name] == nil {
					reservedNameMap[def.Filepath][msg.Name] = make(map[string]int)
				}
				reservedIDMap[def.Filepath][msg.Name][field.ID]++
				reservedNameMap[def.Filepath][msg.Name][field.Name]++
			}
		}
	}

	var warnings []Warning
	// if the field ID was encountered more than once per message, then it
	// is known to be a re-use of a reserved field and a warning should be
	// returned for each occurrance
	for filepath, m := range reservedIDMap {
		for msgName, mm := range m {
			for id, count := range mm {
				if count > 1 {
					msg := fmt.Sprintf(
						"%s is re-using ID: %d, a reserved field",
						msgName, id,
					)
					warnings = append(warnings, Warning{
						Filepath: filepath,
						Message:  msg,
					})
				}
			}
		}
	}
	// if the field name was encountered more than once per message, then it
	// is known to be a re-use of a reserved field and a warning should be
	// returned for each occurrance
	for filepath, m := range reservedNameMap {
		for msgName, mm := range m {
			for name, count := range mm {
				if count > 1 {
					msg := fmt.Sprintf(
						"%s is re-using name: %s, a reserved field",
						msgName, name,
					)
					warnings = append(warnings, Warning{
						Filepath: filepath,
						Message:  msg,
					})
				}
			}
		}
	}

	if debug {
		concludeRuleDebug("NoUsingReservedFields", warnings)
	}

	if warnings != nil {
		return warnings, false
	}

	return nil, true
}

// NoRemovingReservedFields compares the current vs. updated Protolock definitions
// and will return a list of warnings if any reserved field has been removed.
func NoRemovingReservedFields(cur, upd Protolock) ([]Warning, bool) {
	if !strictMode {
		return nil, true
	}

	if debug {
		beginRuleDebug("NoRemovingReservedFields")
	}

	var warnings []Warning
	// check that all reserved fields on current Protolock remain in the
	// updated Protolock
	curReservedIDMap, curReservedNameMap := getReservedFields(cur)
	updReservedIDMap, updReservedNameMap := getReservedFields(upd)
	for filepath, msgMap := range curReservedIDMap {
		for msgName, idMap := range msgMap {
			for id := range idMap {
				if _, ok := updReservedIDMap[filepath][msgName][id]; !ok {
					msg := fmt.Sprintf(
						"%s is missing ID: %d, a reserved field",
						msgName, id,
					)
					warnings = append(warnings, Warning{
						Filepath: filepath,
						Message:  msg,
					})
				}
			}
		}
	}
	for filepath, msgMap := range curReservedNameMap {
		for msgName, nameMap := range msgMap {
			for name := range nameMap {
				if _, ok := updReservedNameMap[filepath][msgName][name]; !ok {
					msg := fmt.Sprintf(
						"%s is missing name: %s, a reserved field",
						msgName, name,
					)
					warnings = append(warnings, Warning{
						Filepath: filepath,
						Message:  msg,
					})
				}
			}
		}
	}

	if debug {
		concludeRuleDebug("NoRemovingReservedFields", warnings)
	}

	if warnings != nil {
		return warnings, false
	}

	return nil, true
}

// NoChangeFieldIDs compares the current vs. updated Protolock definitions and
// will return a list of warnings if any field ID number has been changed.
func NoChangeFieldIDs(cur, upd Protolock) ([]Warning, bool) {
	return nil, true
}

// NoChangeFieldTypes compares the current vs. updated Protolock definitions and
// will return a list of warnings if any field type has been changed.
func NoChangeFieldTypes(cur, upd Protolock) ([]Warning, bool) {
	return nil, true
}

// NoRenamingFields compares the current vs. updated Protolock definitions and
// will return a list of warnings if any message's previous fields have been
// renamed.
func NoRenamingFields(cur, upd Protolock) ([]Warning, bool) {
	if !strictMode {
		return nil, true
	}

	return nil, true
}

// NoRemovingRPCs compares the current vs. updated Protolock definitions and
// will return a list of warnings if any RPCs provided by a Service have been
// removed.
func NoRemovingRPCs(cur, upd Protolock) ([]Warning, bool) {
	if !strictMode {
		return nil, true
	}
	return nil, true
}

// getReservedFields gets all the reserved field numbers and names, and stashes
// them in a lockReservedIDsMap and lockReservedNamesMap to be checked against.
func getReservedFields(lock Protolock) (lockReservedIDsMap, lockReservedNamesMap) {
	reservedIDMap := make(lockReservedIDsMap)
	reservedNameMap := make(lockReservedNamesMap)

	for _, def := range lock.Definitions {
		if reservedIDMap[def.Filepath] == nil {
			reservedIDMap[def.Filepath] = make(map[string]map[int]int)
		}
		if reservedNameMap[def.Filepath] == nil {
			reservedNameMap[def.Filepath] = make(map[string]map[string]int)
		}
		for _, msg := range def.Def.Messages {
			for _, id := range msg.ReservedIDs {
				if reservedIDMap[def.Filepath][msg.Name] == nil {
					reservedIDMap[def.Filepath][msg.Name] = make(map[int]int)
				}
				reservedIDMap[def.Filepath][msg.Name][id]++
			}
			for _, name := range msg.ReservedNames {
				if reservedNameMap[def.Filepath][msg.Name] == nil {
					reservedNameMap[def.Filepath][msg.Name] = make(map[string]int)
				}
				reservedNameMap[def.Filepath][msg.Name][name]++
			}
		}
	}

	return reservedIDMap, reservedNameMap
}

func beginRuleDebug(name string) {
	fmt.Println("run rule:", name)
}

func concludeRuleDebug(name string, warnings []Warning) {
	fmt.Println("warnings:", len(warnings))
	for i, w := range warnings {
		msg := fmt.Sprintf("%d). %s [%s]", i+1, w.Message, w.Filepath)
		fmt.Println(msg)
	}
	fmt.Println("end:", name)
	fmt.Println("===")
}
