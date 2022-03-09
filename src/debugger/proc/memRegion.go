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
	return fmt.Sprintf("%#x-%#x [%s]", mr.Start, mr.End, mr.Ident)
}

func (mr MemRegion) Contents(pid int) []byte {
	return ReadFromMemFile(pid, mr.Start, int(mr.End-mr.Start))
}
