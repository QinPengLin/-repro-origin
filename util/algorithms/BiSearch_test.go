package algorithms

import (
	"fmt"
	"testing"
)

type MyElement struct {
	Score int
}

func (s MyElement) GetValue() int {
	return s.Score
}

func Test_BiSearch(t *testing.T) {
	var schedulePoolCfgList []MyElement = []MyElement{MyElement{10}, MyElement{12}, MyElement{14}, MyElement{16}} //
	index := BiSearch[int, MyElement](schedulePoolCfgList, 9, 1)
	index = BiSearch[int, MyElement](schedulePoolCfgList, 10, 1)
	index = BiSearch[int, MyElement](schedulePoolCfgList, 11, 1)
	index = BiSearch[int, MyElement](schedulePoolCfgList, 12, 1)
	index = BiSearch[int, MyElement](schedulePoolCfgList, 13, 1)
	index = BiSearch[int, MyElement](schedulePoolCfgList, 14, 1)
	index = BiSearch[int, MyElement](schedulePoolCfgList, 15, 1)
	index = BiSearch[int, MyElement](schedulePoolCfgList, 16, 1)
	index = BiSearch[int, MyElement](schedulePoolCfgList, 17, 1)
	fmt.Println(index)
}
