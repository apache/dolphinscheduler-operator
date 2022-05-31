/*
Copyright 2022 Nobolity.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"fmt"
	"strconv"
	"strings"
)

type Member struct {
	Name string
	// Kubernetes namespace this member runs in.
	Namespace string
	// ID field can be 0, which is unknown ID.
	// We know the ID of a member when we get the member information from dbmaster,
	// but not from Kubernetes pod list.
	ID              uint64
	Phase           string
	Created         bool
	RunningAndReady bool
	Version         string
}

func (m Member) Ordinal() int {
	idx := strings.LastIndex(m.Name, "-")
	id, _ := strconv.ParseInt(m.Name[idx+1:], 10, 32)
	return int(id)
}

type MemberSet map[string]*Member

// the set of all members of s1 that are not members of s2
func (ms MemberSet) Diff(other MemberSet) MemberSet {
	diff := MemberSet{}
	for n, m := range ms {
		if _, ok := other[n]; !ok {
			diff[n] = m
		}
	}
	return diff
}

func (ms MemberSet) Get(id int) *Member {
	for _, m := range ms {
		idx := strings.LastIndex(m.Name, "-")
		mid, _ := strconv.ParseInt(m.Name[idx+1:], 10, 32)
		if int(mid) == id {
			return m
		}
	}
	return nil
}

// IsEqual tells whether two member sets are equal by checking
// - they have the same set of members and member equality are judged by Name only.
func (ms MemberSet) IsEqual(other MemberSet) bool {
	if ms.Size() != other.Size() {
		return false
	}
	for n := range ms {
		if _, ok := other[n]; !ok {
			return false
		}
	}
	return true
}

func (ms MemberSet) Size() int {
	return len(ms)
}

func (ms MemberSet) String() string {
	var mstring []string

	for m, v := range ms {
		mstring = append(mstring, fmt.Sprintf("%s:%s", m, v.Version))
	}
	return strings.Join(mstring, ",")
}

func (ms MemberSet) PickOne() *Member {
	for _, m := range ms {
		return m
	}
	panic("empty")
}

func (ms MemberSet) Add(m *Member) {
	ms[m.Name] = m
}

func (ms MemberSet) Remove(name string) {
	delete(ms, name)
}

func (ms MemberSet) Ordinals() map[int]bool {
	ids := map[int]bool{}
	for _, m := range ms {
		ids[m.Ordinal()] = true
	}
	return ids
}

func (ms MemberSet) Names() []string {
	names := make([]string, 0)
	for _, m := range ms {
		names = append(names, m.Name)
	}
	return names
}

func (ms MemberSet) Duplicate() MemberSet {
	r := MemberSet{}
	for k, v := range ms {
		r[k] = v
	}
	return r
}

func allMembersHealth(ms MemberSet) bool {
	for _, m := range ms {
		if !m.RunningAndReady {
			return false
		}
	}
	return true
}
