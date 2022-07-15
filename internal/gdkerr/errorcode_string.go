// Code generated by "stringer -type=ErrorCode"; DO NOT EDIT.

package gdkerr

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[OK-0]
	_ = x[Unknown-1]
	_ = x[NotFound-2]
	_ = x[AlreadyExists-3]
	_ = x[InvalidArgument-4]
	_ = x[Internal-5]
	_ = x[Unimplemented-6]
	_ = x[FailedPrecondition-7]
	_ = x[PermissionDenied-8]
	_ = x[ResourceExhausted-9]
	_ = x[Canceled-10]
	_ = x[DeadlineExceeded-11]
}

const _ErrorCode_name = "OKUnknownNotFoundAlreadyExistsInvalidArgumentInternalUnimplementedFailedPreconditionPermissionDeniedResourceExhaustedCanceledDeadlineExceeded"

var _ErrorCode_index = [...]uint8{0, 2, 9, 17, 30, 45, 53, 66, 84, 100, 117, 125, 141}

func (i ErrorCode) String() string {
	if i < 0 || i >= ErrorCode(len(_ErrorCode_index)-1) {
		return "ErrorCode(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ErrorCode_name[_ErrorCode_index[i]:_ErrorCode_index[i+1]]
}