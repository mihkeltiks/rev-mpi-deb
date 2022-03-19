package proc

import (
	"fmt"
)

type MemRegion struct {
	Start    uint64
	End      uint64
	Ident    string
	Contents []byte
}

func (mr MemRegion) String() string {
	return fmt.Sprintf("start-%#x end-%#x (size %d) %s", mr.Start, mr.End, mr.End-mr.Start, mr.Ident)
}

func (mr MemRegion) ContentsFromFile(pid int) []byte {
	return ReadFromMemFile(pid, mr.Start, int(mr.End-mr.Start))
}
