package proc

import (
	"fmt"
)

type MemRegion struct {
	Start uint64
	End   uint64
	Ident string
}

func (mr MemRegion) String() string {
	return fmt.Sprintf("start-%#x len-%#x %s", mr.Start, mr.End, mr.Ident)
}

func (mr MemRegion) Contents(pid int) []byte {
	return ReadFromMemFile(pid, mr.Start, int(mr.End-mr.Start))
}
