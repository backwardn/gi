// Code generated by "stringer -type=Dims"; DO NOT EDIT.

package mat32

import (
	"errors"
	"strconv"
)

var _ = errors.New("dummy error")

const _Dims_name = "XYZWDimsN"

var _Dims_index = [...]uint8{0, 1, 2, 3, 4, 9}

func (i Dims) String() string {
	if i < 0 || i >= Dims(len(_Dims_index)-1) {
		return "Dims(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Dims_name[_Dims_index[i]:_Dims_index[i+1]]
}

func (i *Dims) FromString(s string) error {
	for j := 0; j < len(_Dims_index)-1; j++ {
		if s == _Dims_name[_Dims_index[j]:_Dims_index[j+1]] {
			*i = Dims(j)
			return nil
		}
	}
	return errors.New("String: " + s + " is not a valid option for type: Dims")
}