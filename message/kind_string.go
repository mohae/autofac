// Code generated by "stringer -type=Kind"; DO NOT EDIT

package message

import "fmt"

const _Kind_name = "UnknownEOTGenericCommandSysInfoFBSysInfoJSONClientConfCPUUtilizationLoadAvgMemInfoNetUsage"

var _Kind_index = [...]uint8{0, 7, 10, 17, 24, 33, 44, 54, 68, 75, 82, 90}

func (i Kind) String() string {
	if i < 0 || i >= Kind(len(_Kind_index)-1) {
		return fmt.Sprintf("Kind(%d)", i)
	}
	return _Kind_name[_Kind_index[i]:_Kind_index[i+1]]
}
