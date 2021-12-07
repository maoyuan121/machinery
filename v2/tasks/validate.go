package tasks

import (
	"errors"
	"reflect"
)

var (
	// ErrTaskMustBeFunc ...
	ErrTaskMustBeFunc = errors.New("Task must be a func type")
	// ErrTaskReturnsNoValue ...
	ErrTaskReturnsNoValue = errors.New("Task must return at least a single value")
	// ErrLastReturnValueMustBeError ..
	ErrLastReturnValueMustBeError = errors.New("Last return value of a task must be error")
)

// ValidateTask 使用反射验证任务函数，并确保它具有正确的签名。
// 用作任务的函数必须至少返回一个值，最后一个返回类型必须是 error
func ValidateTask(task interface{}) error {
	v := reflect.ValueOf(task)
	t := v.Type()

	//  任务必须是一个函数
	if t.Kind() != reflect.Func {
		return ErrTaskMustBeFunc
	}

	// Task 至少有一个返回值
	if t.NumOut() < 1 {
		return ErrTaskReturnsNoValue
	}

	// 最后一个返回值必须是 error
	lastReturnType := t.Out(t.NumOut() - 1)
	errorInterface := reflect.TypeOf((*error)(nil)).Elem()
	if !lastReturnType.Implements(errorInterface) {
		return ErrLastReturnValueMustBeError
	}

	return nil
}
