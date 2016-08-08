// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"sort"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/utils/set"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2/txn"
)

// machineRemovalDoc indicates that this machine needs to be removed
// and any necessary provider-level cleanup should now be done.
type machineRemovalDoc struct {
	DocID     string `bson:"_id"`
	MachineID string `bson:"machine-id"`
}

// MarkForRemoval requests that this machine be removed after any
// needed provider-level cleanup is done.
func (m *Machine) MarkForRemoval() (err error) {
	defer errors.DeferredAnnotatef(&err, "cannot remove machine %s", m.doc.Id)
	if m.doc.Life != Dead {
		return errors.Errorf("machine is not dead")
	}
	ops := []txn.Op{{
		C:  machinesC,
		Id: m.doc.DocID,
		// Check that the machine is still dead (and implicitly that
		// it still exists).
		Assert: isDeadDoc,
	}, {
		C:      machineRemovalsC,
		Id:     m.globalKey(),
		Insert: &machineRemovalDoc{MachineID: m.Id()},
		// No assert here - it's ok if the machine has already been
		// marked. The id will prevent duplicates.
	}}
	return onAbort(m.st.runTransaction(ops), errors.Errorf("machine is not dead"))
}

// AllMachineRemovals returns (the ids of) all of the machines that
// need to be removed but need provider-level cleanup.
func (st *State) AllMachineRemovals() ([]string, error) {
	removals, close := st.getCollection(machineRemovalsC)
	defer close()

	var docs []machineRemovalDoc
	err := removals.Find(nil).All(&docs)
	if err != nil {
		return nil, errors.Trace(err)
	}
	results := make([]string, len(docs))
	for i := range docs {
		results[i] = docs[i].MachineID
	}
	return results, nil
}

func (st *State) allMachinesMatching(query bson.D) ([]*Machine, error) {
	machines, close := st.getCollection(machinesC)
	defer close()

	var docs []machineDoc
	err := machines.Find(query).All(&docs)
	if err != nil {
		return nil, errors.Trace(err)
	}
	results := make([]*Machine, len(docs))
	for i, doc := range docs {
		results[i] = newMachine(st, &doc)
	}
	return results, nil
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func collectMissingMachineIds(expectedIds []string, machines []*Machine) []string {
	expectedSet := set.NewStrings(expectedIds...)
	actualSet := set.NewStrings()
	for _, machine := range machines {
		actualSet.Add(machine.Id())
	}
	return expectedSet.Difference(actualSet).SortedValues()
}

// CompleteMachineRemovals finishes the removal of the specified
// machines. The machines must have been marked for removal
// previously. Unknown machine ids are ignored so that this is
// idempotent.
func (st *State) CompleteMachineRemovals(ids ...string) error {
	removals, err := st.AllMachineRemovals()
	if err != nil {
		return errors.Trace(err)
	}
	removalSet := set.NewStrings(removals...)
	query := bson.D{{"machineid", bson.D{{"$in", ids}}}}
	machinesToRemove, err := st.allMachinesMatching(query)
	if err != nil {
		return errors.Trace(err)
	}

	if len(machinesToRemove) < len(ids) {
		missingMachines := collectMissingMachineIds(ids, machinesToRemove)
		logger.Debugf("skipping nonexistent machine%s: %s",
			plural(len(missingMachines)),
			strings.Join(missingMachines, ", "),
		)
	}

	var ops []txn.Op
	var missingRemovals []string
	for _, machine := range machinesToRemove {
		if !removalSet.Contains(machine.Id()) {
			missingRemovals = append(missingRemovals, machine.Id())
			continue
		}

		ops = append(ops, txn.Op{
			C:      machineRemovalsC,
			Id:     machine.globalKey(),
			Remove: true,
		})
		removeMachineOps, err := machine.removeOps()
		if err != nil {
			return errors.Trace(err)
		}
		ops = append(ops, removeMachineOps...)
	}
	// We should complain about machines that still exist but haven't
	// been marked for removal.
	if len(missingRemovals) > 0 {
		sort.Strings(missingRemovals)
		return errors.Errorf(
			"cannot remove machine%s %s: not marked for removal",
			plural(len(missingRemovals)),
			strings.Join(missingRemovals, ", "),
		)
	}

	return st.runTransaction(ops)
}
